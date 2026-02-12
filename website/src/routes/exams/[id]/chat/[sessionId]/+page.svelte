<script lang="ts">
    import { onMount, onDestroy, tick } from 'svelte';
    import { page } from '$app/state';
    import { api } from '$lib/api/client';
    import { notifications } from '$lib/stores/notifications.svelte';
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';
    import Tile from '$lib/components/Tile.svelte';
    import { Send, User, Bot, Sparkles, Search, MessageSquare, BookOpen, Layers, Square, CheckSquare, Lock } from 'lucide-svelte';

    let { id: examId, sessionId } = $derived(page.params);
    let exam = $state<any>(null);
    let session = $state<any>(null);
    let messages = $state<any[]>([]);
    let otherSessions = $state<any[]>([]);
    let allLectures = $state<any[]>([]);
    let includedLectureIds = $state<string[]>([]);
    let usedLectureIds = $state<string[]>([]);
    let input = $state('');
    let loading = $state(true);
    let sending = $state(false);
    let updatingContext = $state(false);
    let socket: WebSocket | null = null;
    let streamingMessage = $state('');
    let messageContainer: HTMLDivElement | null = $state(null);

    async function loadData() {
        loading = true;
        try {
            const [examData, details, allSessions, lecturesData] = await Promise.all([
                api.getExam(examId),
                api.request('GET', `/chat/sessions/details?session_id=${sessionId}&exam_id=${examId}`),
                api.request('GET', `/chat/sessions?exam_id=${examId}`),
                api.listLectures(examId)
            ]);
            
            exam = examData;
            session = details.session;
            messages = details.messages ?? [];
            includedLectureIds = details.context?.included_lecture_ids ?? [];
            usedLectureIds = details.context?.used_lecture_ids ?? [];
            otherSessions = (allSessions ?? []).filter((s: any) => s.id !== sessionId).slice(0, 3);
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
            await api.request('PATCH', '/chat/sessions/context', {
                session_id: sessionId,
                included_lecture_ids: includedLectureIds,
                included_tool_ids: [] // Future-proofing
            });
        } catch (e: any) {
            notifications.error('Failed to update study context: ' + (e.message || e));
        } finally {
            updatingContext = false;
        }
    }

    function toggleLecture(id: string) {
        if (includedLectureIds.includes(id)) {
            includedLectureIds = includedLectureIds.filter(i => i !== id);
        } else {
            includedLectureIds = [...includedLectureIds, id];
        }
        updateContext();
    }

    function setupWebSocket() {
        const token = localStorage.getItem('session_token');
        const baseUrl = api.getBaseUrl().replace('http', 'ws');
        socket = new WebSocket(`${baseUrl}/socket?session_token=${token}`);
        
        socket.onopen = () => {
            if (sessionId) {
                socket?.send(JSON.stringify({
                    type: 'subscribe',
                    channel: `chat:${sessionId}`
                }));
            }
        };

        socket.onmessage = async (event) => {
            const data = JSON.parse(event.data);
            if (data.type === 'chat:token') {
                streamingMessage = data.payload.accumulated_text;
                scrollToBottom();
            } else if (data.type === 'chat:complete') {
                messages = [...messages, data.payload];
                streamingMessage = '';
                sending = false;
                await tick();
                scrollToBottom();
            }
        };
    }

    async function sendMessage() {
        if (!input.trim() || sending || !sessionId) return;
        
        const userMsg = { role: 'user', content: input, created_at: new Date().toISOString() };
        messages = [...messages, userMsg];
        
        // Locally lock used IDs immediately
        const newUsed = new Set([...usedLectureIds, ...includedLectureIds]);
        usedLectureIds = Array.from(newUsed);

        const content = input;
        input = '';
        sending = true;
        
        await tick();
        scrollToBottom();

        try {
            await api.request('POST', '/chat/messages', {
                session_id: sessionId,
                content
            });
        } catch (e: any) {
            notifications.error(e.message || e);
            sending = false;
        }
    }

    function scrollToBottom() {
        if (messageContainer) {
            messageContainer.scrollTop = messageContainer.scrollHeight;
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
    <Breadcrumb items={[
        { label: 'My Studies', href: '/exams' }, 
        { label: exam.title, href: `/exams/${examId}` }, 
        { label: 'Conversations', href: `/exams/${examId}` },
        { label: session.title || 'Untitled Chat', active: true }
    ]} />

    <div class="d-flex justify-content-between align-items-center mb-3">
        <h2>{session.title || 'AI Assistant'}</h2>
    </div>

    <div class="container-fluid p-0">
        <div class="row">
            <!-- Sidebar: Context Selection & Other Sessions -->
            <div class="col-lg-3 col-md-4 order-md-2">
                <div class="bg-white border mb-4">
                    <div class="standard-header">
                        <div class="header-title">
                            <span class="header-glyph" lang="ja">基</span>
                            <span class="header-text">Knowledge Base</span>
                        </div>
                        {#if updatingContext}
                            <div class="spinner-border spinner-border-sm text-success" role="status">
                                <span class="visually-hidden">Syncing...</span>
                            </div>
                        {/if}
                    </div>
                    <div class="p-3 bg-light border-bottom">
                        <p class="text-muted small mb-0">Choose the lessons I should use to answer your questions.</p>
                    </div>
                    
                    <div class="bg-white p-0 overflow-auto" style="max-height: 40vh;">
                    {#each allLectures as lecture}
                        {@const isLocked = usedLectureIds.includes(lecture.id)}
                        {@const isIncluded = includedLectureIds.includes(lecture.id)}
                        <button 
                            onclick={() => !isLocked && toggleLecture(lecture.id)}
                            class="w-100 border-0 border-bottom last-child-border-0 p-3 text-start d-flex align-items-center gap-3"
                            style="background: {isIncluded ? 'rgba(86, 143, 39, 0.05)' : 'transparent'}; 
                                   transition: all 0.15s ease; 
                                   cursor: {isLocked ? 'not-allowed' : 'pointer'};
                                   opacity: {isLocked ? 0.8 : 1};"
                            disabled={updatingContext || isLocked}
                            title={isLocked ? 'This context is locked because it was already used in this chat.' : ''}
                        >
                            <div class="flex-shrink-0">
                                {#if isLocked}
                                    <div class="d-flex align-items-center justify-content-center" style="width: 1.1rem; height: 1.1rem; color: #568f27;">
                                        <Lock size={14} />
                                    </div>
                                {:else}
                                    <div class="border" style="width: 1.1rem; height: 1.1rem; border-color: {isIncluded ? '#568f27' : '#ccc'} !important; background: {isIncluded ? '#568f27' : 'transparent'};">
                                        {#if isIncluded}
                                            <div class="d-flex align-items-center justify-content-center h-100">
                                                <div style="width: 0.4rem; height: 0.4rem; background: white;"></div>
                                            </div>
                                        {/if}
                                    </div>
                                {/if}
                            </div>
                            <span class="small {isIncluded ? 'fw-bold text-success' : 'text-dark'}" style="line-height: 1.2;">
                                {lecture.title}
                            </span>
                        </button>
                    {/each}
                    {#if allLectures.length === 0}
                        <div class="p-4 text-center"><p class="text-muted mb-0">No lessons available yet.</p></div>
                    {/if}
                </div>

                {#if otherSessions.length > 0}
                    <div class="bg-white border mb-4">
                        <div class="standard-header">
                            <div class="header-title">
                                <span class="header-glyph" lang="ja">談</span>
                                <span class="header-text">Recent Chats</span>
                            </div>
                        </div>
                        <div class="linkTiles tileSizeMd w-100 m-0 d-flex flex-column align-items-center p-2">
                            {#each otherSessions as other}
                                <Tile href="/exams/{examId}/chat/{other.id}" icon="談" title={other.title || 'Untitled Chat'}>
                                    {#snippet description()}
                                        Switch to this study session.
                                    {/snippet}
                                </Tile>
                            {/each}
                        </div>
                    </div>
                {/if}
            </div>

            <!-- Main Content: Chat History -->
            <div class="col-lg-9 col-md-8 order-md-1">
                <form onsubmit={(e) => { e.preventDefault(); sendMessage(); }} class="mb-4">
                    <div class="input-group dictionary-style mb-3 shadow-sm">
                        <input 
                            type="text" 
                            class="form-control" 
                            placeholder="Ask about your lectures or reference documents..." 
                            bind:value={input}
                            disabled={sending}
                        />
                        <button class="btn btn-success" type="submit" disabled={sending || !input.trim()}>
                            <span class="glyphicon m-0"><Search size={18} /></span>
                        </button>
                    </div>
                </form>

                <div class="chat-viewport mb-5" bind:this={messageContainer} style="height: 65vh; overflow-y: auto;">
                    {#if (!messages || messages.length === 0) && !streamingMessage}
                        <div class="well bg-white text-center p-5 text-muted border shadow-sm">
                            <Bot size={48} class="mb-3 opacity-25" />
                            <p>I'm your personal AI assistant. Pick the lessons you want to talk about from the Knowledge Base, and ask a question above!</p>
                        </div>
                    {/if}

                    {#each messages ?? [] as msg, i}
                        {#if msg.role === 'assistant'}
                            {@const prevMsg = messages[i-1]}
                            <div class="bg-white p-0 overflow-hidden mb-5 border shadow-none">
                                <div class="standard-header">
                                    <div class="header-title overflow-hidden">
                                        <span class="header-glyph" lang="ja">案</span>
                                        <span class="header-text">Assistant</span>
                                        {#if prevMsg && prevMsg.role === 'user'}
                                            <span class="text-muted small text-truncate ms-2 border-start ps-2" style="font-weight: normal; opacity: 0.6; text-transform: none;">
                                                {prevMsg.content}
                                            </span>
                                        {/if}
                                    </div>
                                    <span class="text-muted small flex-shrink-0" style="font-size: 0.75rem;">
                                        {new Date(msg.created_at || Date.now()).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                                    </span>
                                </div>
                                <div class="p-4 prose">
                                    {#if msg.content_html}
                                        {@html msg.content_html}
                                    {:else}
                                        <p>{msg.content}</p>
                                    {/if}
                                </div>
                            </div>
                        {/if}
                    {/each}

                    {#if streamingMessage}
                        <div class="bg-white p-0 overflow-hidden mb-5 border shadow-none">
                            <div class="standard-header">
                                <div class="header-title overflow-hidden">
                                    <span class="header-glyph" lang="ja">案</span>
                                    <span class="header-text">Assistant</span>
                                    {#if messages.length > 0 && messages[messages.length-1].role === 'user'}
                                        <span class="text-muted small text-truncate ms-2 border-start ps-2" style="font-weight: normal; opacity: 0.6; text-transform: none;">
                                            {messages[messages.length-1].content}
                                        </span>
                                    {/if}
                                </div>
                                <div class="spinner-border spinner-border-sm text-success" role="status">
                                    <span class="visually-hidden">Thinking...</span>
                                </div>
                            </div>
                            <div class="p-4 prose">
                                <p>{streamingMessage}</p>
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
{:else if loading}
    <div class="text-center p-5">
        <div class="village-spinner mx-auto"></div>
    </div>
{/if}

<style>
    .message-content {
        white-space: pre-wrap;
    }
    
    .cursor-pointer {
        cursor: pointer;
    }

    /* Scrollbar styling */
    .chat-viewport::-webkit-scrollbar {
        width: 8px;
    }
    .chat-viewport::-webkit-scrollbar-track {
        background: transparent;
    }
    .chat-viewport::-webkit-scrollbar-thumb {
        background: #ddd;
    }
    .chat-viewport::-webkit-scrollbar-thumb:hover {
        background: #ccc;
    }

    .active-context {
        background-color: #568f27 !important;
        color: white !important;
        border-color: #568f27 !important;
    }
</style>
