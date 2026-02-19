<script lang="ts">
  import { onMount, onDestroy, tick } from "svelte";
  import { page } from "$app/state";
  import { browser } from "$app/environment";
  import { api } from "$lib/api/client";
  import { notifications } from "$lib/stores/notifications.svelte";
  import Breadcrumb from "$lib/components/Breadcrumb.svelte";
  import Tile from "$lib/components/Tile.svelte";
  import VerticalTileList from "$lib/components/navigation/VerticalTileList.svelte";
  import {
    Send,
    User,
    Bot,
    Sparkles,
    Search,
    MessageSquare,
    BookOpen,
    Layers,
    Square,
    CheckSquare,
    Lock,
    X,
  } from "lucide-svelte";

  let { id: examId, sessionId } = $derived(page.params);
  let exam = $state<any>(null);
  let session = $state<any>(null);
  let messages = $state<any[]>([]);
  let otherSessions = $state<any[]>([]);
  let allLectures = $state<any[]>([]);
  let includedLectureIds = $state<string[]>([]);
  let usedLectureIds = $state<string[]>([]);
  let input = $state("");
  let loading = $state(true);
  let sending = $state(false);
  let updatingContext = $state(false);
  let socket: WebSocket | null = null;
  let streamingMessage = $state("");
  let messageContainer: HTMLDivElement | null = $state(null);

  async function loadData() {
    loading = true;
    try {
      const [examData, details, allSessions, lecturesData] = await Promise.all([
        api.getExam(examId!),
        api.request(
          "GET",
          `/chat/sessions/details?session_id=${sessionId}&exam_id=${examId}`,
        ),
        api.request("GET", `/chat/sessions?exam_id=${examId}`),
        api.listLectures(examId!),
      ]);

      exam = examData;
      session = details.session;
      messages = details.messages ?? [];
      includedLectureIds = details.context?.included_lecture_ids ?? [];
      usedLectureIds = details.context?.used_lecture_ids ?? [];
      otherSessions = (allSessions ?? [])
        .filter((s: any) => s.id !== sessionId)
        .slice(0, 3);
      allLectures = lecturesData ?? [];

      setupWebSocket();
      scrollToBottom();
    } catch (e) {
      console.error(e);
    } finally {
      loading = false;
    }
  }

  async function updateContext() {
    updatingContext = true;
    try {
      await api.request("PATCH", "/chat/sessions/context", {
        session_id: sessionId,
        included_lecture_ids: includedLectureIds,
        included_tool_ids: [], // Future-proofing
      });
    } catch (e: any) {
      notifications.error(
        "Failed to update study context: " + (e.message || e),
      );
    } finally {
      updatingContext = false;
    }
  }

  function toggleLecture(id: string) {
    if (includedLectureIds.includes(id)) {
      includedLectureIds = includedLectureIds.filter((i) => i !== id);
    } else {
      includedLectureIds = [...includedLectureIds, id];
    }
    updateContext();
  }

  function setupWebSocket() {
    if (!browser || !sessionId) return;

    if (socket) {
      socket.close();
    }

    const token = localStorage.getItem("session_token");
    const baseUrl = api.getBaseUrl().replace("http", "ws");
    socket = new WebSocket(
      `${baseUrl}/socket?session_token=${token}&subscribe_chat=${sessionId}`,
    );

    socket.onopen = () => {
      // Already auto-subscribed via query param, but we keep this for redundancy or fallback
      if (sessionId) {
        socket?.send(
          JSON.stringify({
            type: "subscribe",
            channel: `chat:${sessionId}`,
          }),
        );
      }
    };

    socket.onmessage = async (event) => {
      const data = JSON.parse(event.data);
      if (data.type === "chat:token") {
        streamingMessage = data.payload.accumulated_text;
        scrollToBottom();
      } else if (data.type === "chat:complete") {
        messages = [...messages, data.payload];
        streamingMessage = "";
        sending = false;
        await tick();
        scrollToBottom();
      } else if (data.type === "chat:error") {
        notifications.error(
          data.payload.error || "An error occurred during generation",
        );
        sending = false;
        streamingMessage = "";
      } else if (data.type === "job:progress") {
        if (data.payload.Status === "FAILED") {
          notifications.error(
            `Background task failed: ${data.payload.Error || "Unknown error"}`,
          );
        }
      }
    };
  }

  async function sendMessage() {
    if (!input.trim() || sending || !sessionId) return;

    const userMsg = {
      role: "user",
      content: input,
      created_at: new Date().toISOString(),
    };
    messages = [...messages, userMsg];

    // Locally lock used IDs immediately
    const newUsed = new Set([...usedLectureIds, ...includedLectureIds]);
    usedLectureIds = Array.from(newUsed);

    const content = input;
    input = "";
    sending = true;

    await tick();
    scrollToBottom();

    try {
      const serverMsg = await api.request("POST", "/chat/messages", {
        session_id: sessionId,
        content,
      });
      // Update the message in the list with the server-side version (containing metadata)
      messages = messages.map((m) => (m === userMsg ? serverMsg : m));
    } catch (e: any) {
      notifications.error(e.message || e);
      sending = false;
    }
  }

  function scrollToBottom() {
    if (browser) {
      window.scrollTo({
        top: document.body.scrollHeight,
        behavior: "smooth",
      });
    }
  }

  $effect(() => {
    if (examId && sessionId) {
      loadData();
    }
  });

  onDestroy(() => socket?.close());
</script>

{#if exam && session}
  <Breadcrumb
    items={[
      { label: "My Studies", href: "/exams" },
      { label: exam.title, href: `/exams/${examId}` },
      { label: "Conversations", href: `/exams/${examId}` },
      { label: session.title || "Untitled Chat", active: true },
    ]}
  />

  <header class="page-header">
    <div class="d-flex align-items-center gap-3">
      <h1 class="page-title m-0">{session.title || "AI Assistant"}</h1>
    </div>
  </header>

  <div class="container-fluid p-0">
    <div class="row g-4">
      <!-- Sidebar: Context & Selection -->
      <div class="col-lg-4 order-lg-2">
        <div class="bg-white border mb-4">
          <div class="standard-header">
            <div class="header-title">
              <span class="header-text">Knowledge Base</span>
            </div>
            {#if updatingContext}
              <div
                class="spinner-border spinner-border-sm text-orange"
                role="status"
              ></div>
            {/if}
          </div>
          <div class="p-3">
            <p class="text-muted small mb-3">
              Select which lessons to include in this conversation's context.
            </p>
            <VerticalTileList direction="vertical" class="adaptive-width">
              {#each allLectures as lecture}
                {@const isUsed = usedLectureIds.includes(lecture.id)}
                {@const isIncluded = includedLectureIds.includes(lecture.id)}
                <Tile
                  icon=""
                  title={lecture.title}
                  onclick={() => !isUsed && toggleLecture(lecture.id)}
                  disabled={isUsed}
                  class="{isUsed ? 'tile-locked' : ''}"
                >
                  {#snippet description()}
                    {lecture.description || "Lesson materials."}
                  {/snippet}

                  {#snippet actions()}
                    <div class="lecture-toggle-wrapper">
                      {#if isUsed}
                        <Lock size={14} class="lock-icon" />
                      {/if}
                      <div
                        class="village-toggle {isIncluded || isUsed
                          ? 'is-active'
                          : ''} {isUsed ? 'is-locked' : 'cursor-pointer'}"
                        title={isUsed
                          ? "Already used in this conversation"
                          : "Toggle lesson"}
                        onclick={(e) => {
                          e.stopPropagation();
                          !isUsed && toggleLecture(lecture.id);
                        }}
                      ></div>
                    </div>
                  {/snippet}
                </Tile>
              {:else}
                <div class="p-4 text-center text-muted small border">
                  No lessons found in this subject.
                </div>
              {/each}
            </VerticalTileList>
          </div>
        </div>
      </div>

      <!-- Main Content: Chat History -->
      <div class="col-lg-8 order-lg-1">
        <form
          onsubmit={(e) => {
            e.preventDefault();
            sendMessage();
          }}
          class="mb-4"
        >
          <div class="input-group dictionary-style mb-3">
            <input
              type="text"
              class="form-control cozy-input"
              placeholder="Ask about your lectures or reference documents..."
              bind:value={input}
              disabled={sending}
            />
            <button
              class="btn btn-success"
              type="submit"
              disabled={sending || !input.trim()}
            >
              <Search size={18} />
            </button>
          </div>
        </form>

        <div class="chat-viewport mb-3" bind:this={messageContainer}>
          {#if (!messages || messages.length === 0) && !streamingMessage}
            <div class="well text-center p-5 text-muted border">
              <Bot size={48} class="mb-3 opacity-25" />
              <p>
                Select lessons from your Knowledge Base and start a
                conversation.
              </p>
            </div>
          {/if}

          {#each messages ?? [] as msg, i}
            {#if msg.role === "assistant"}
              {@const prevMsg = messages[i - 1]}
              <div class="bg-white p-0 mb-4 border">
                <div class="standard-header">
                  <div class="header-title">
                    <span class="header-text">Assistant</span>
                    {#if prevMsg && prevMsg.role === "user"}
                      <span
                        class="text-muted small text-truncate ms-3 fw-normal"
                        style="opacity: 0.7; text-transform: none; font-style: italic; font-size: 0.9rem;"
                      >
                        “{prevMsg.content}”
                      </span>
                    {/if}
                  </div>
                  <div class="d-flex align-items-center gap-3">
                    <span
                      class="text-muted small flex-shrink-0"
                      style="font-size: 0.85rem;"
                    >
                      {new Date(
                        msg.created_at || Date.now(),
                      ).toLocaleTimeString([], {
                        hour: "2-digit",
                        minute: "2-digit",
                      })}
                    </span>
                  </div>
                </div>
                <div class="p-4 prose">
                  {#if msg.content_html}
                    {@html msg.content_html}
                  {:else}
                    <p style="white-space: pre-wrap;">{msg.content}</p>
                  {/if}
                </div>
              </div>
            {/if}
          {/each}

          {#if streamingMessage}
            <div class="bg-white p-0 mb-4 border">
              <div class="standard-header">
                <div class="header-title">
                  <span class="header-text">Assistant</span>
                  {#if messages.length > 0 && messages[messages.length - 1].role === "user"}
                    <span
                      class="text-muted small text-truncate ms-3 fw-normal"
                      style="opacity: 0.7; text-transform: none; font-style: italic;"
                    >
                      “{messages[messages.length - 1].content}”
                    </span>
                  {/if}
                </div>
                <div class="d-flex align-items-center gap-3">
                  {#if messages.length > 0 && messages[messages.length - 1].role === "user"}
                    {@const lastUserMsg = messages[messages.length - 1]}
                    {@const metadata = lastUserMsg.metadata
                      ? typeof lastUserMsg.metadata === "string"
                        ? JSON.parse(lastUserMsg.metadata)
                        : lastUserMsg.metadata
                      : null}
                    {#if metadata?.new_lecture_titles?.length > 0}
                      <span
                        class="text-muted small"
                        style="font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.05em;"
                      >
                        Associating: {metadata.new_lecture_titles.join(", ")}
                      </span>
                    {/if}
                  {/if}
                  <div class="spinner-border spinner-border-sm" role="status">
                    <span class="visually-hidden">Thinking...</span>
                  </div>
                </div>
              </div>
              <div class="p-4 prose">
                <p style="white-space: pre-wrap;">{streamingMessage}</p>
              </div>
            </div>
          {/if}

          {#if sending && !streamingMessage}
            <div class="text-center p-4">
              <div class="village-spinner mx-auto"></div>
            </div>
          {/if}
        </div>
      </div>
    </div>
  </div>
{/if}

{#if loading}
  <div class="text-center p-5">
    <div class="d-flex flex-column align-items-center gap-3">
      <div class="village-spinner mx-auto" role="status"></div>
      <p class="text-muted mb-0">Connecting to assistant...</p>
    </div>
  </div>
{/if}

<style lang="scss">
  .no-shift-bold {
    display: inline-grid;
    text-align: left;
  }

  .no-shift-bold::after {
    content: attr(data-text);
    grid-area: 1 / 1;
    font-weight: bold;
    visibility: hidden;
    height: 0;
  }

  .no-shift-bold > span {
    grid-area: 1 / 1;
  }

  /* Kakimashou Toggle Switch */
  .lecture-toggle-wrapper {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .lock-icon {
    color: var(--gray-500);
    flex-shrink: 0;
  }

  .village-toggle {
    position: relative;
    width: 2.5rem;
    height: 1.25rem;
    background: #eee;
    border: 1px solid #ccc;
    flex-shrink: 0;
    transition: all 0.2s ease;
  }

  .village-toggle::after {
    content: "";
    position: absolute;
    top: 1px;
    left: 1px;
    width: calc(1.25rem - 4px);
    height: calc(1.25rem - 4px);
    background: #fff;
    box-shadow: 1px 1px 2px rgba(0, 0, 0, 0.2);
    transition: all 0.2s ease;
  }

  .village-toggle.is-active {
    background: var(--orange);
    border-color: var(--orange);
  }

  .village-toggle.is-active::after {
    left: calc(2.5rem - 1.25rem + 1px);
  }

  .village-toggle.is-locked.is-active {
    background: rgba(234, 126, 12, 0.5);
    border-color: rgba(234, 126, 12, 0.5);
    cursor: not-allowed;
  }

  /* Input group alignment */
  .dictionary-style {
    .btn {
      height: 2.5rem;
      font-size: 0.8rem;
    }
  }

  .village-toggle.is-locked::after {
    background: rgba(255, 255, 255, 0.7);
  }
</style>
