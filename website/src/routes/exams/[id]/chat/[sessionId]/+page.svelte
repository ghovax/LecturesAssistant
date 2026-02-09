<script lang="ts">
    import { onMount, onDestroy, tick } from 'svelte';
    import { page } from '$app/state';
    import { api } from '$lib/api/client';
    import { notifications } from '$lib/stores/notifications.svelte';
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';
    import Tile from '$lib/components/Tile.svelte';
    import { Send, User, Bot, Sparkles, Search, MessageSquare, BookOpen, Layers } from 'lucide-svelte';

    let { id: examId, sessionId } = $derived(page.params);
    let exam = $state<any>(null);
    let session = $state<any>(null);
    let messages = $state<any[]>([]);
    let otherSessions = $state<any[]>([]);
    let allLectures = $state<any[]>([]);
    let includedLectureIds = $state<string[]>([]);
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
            notifications.success('Study context updated.');
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
        socket = new WebSocket(`ws://localhost:3000/api/socket?session_token=${token}`);
        
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

    onMount(loadData);
    onDestroy(() => socket?.close());
</script>

{#if exam && session}
    <Breadcrumb items={[
        { label: 'My Studies', href: '/exams' }, 
        { label: exam.title, href: `/exams/${examId}` }, 
        { label: 'Study Chats', href: `/exams/${examId}` },
        { label: session.title || 'Untitled Chat', active: true }
    ]} />

    <div class="d-flex justify-content-between align-items-center mb-3">
        <h2>{session.title || 'Study Assistant'}</h2>
    </div>

    <div class="container-fluid p-0">
        <div class="row">
            <!-- Sidebar: Session Info & Context Selection -->
            <div class="col-lg-3 col-md-4 order-md-2">
                <div class="d-flex justify-content-between align-items-center mb-2">
                    <h3 class="m-0 border-0">Study Context</h3>
                    {#if updatingContext}
                        <span class="village-spinner" style="width: 1rem; height: 1rem;"></span>
                    {/if}
                </div>
                
                <div class="well bg-white p-0 overflow-hidden border mb-4">
                    <div class="bg-light px-3 py-2 border-bottom small fw-bold text-uppercase text-muted">
                        <Layers size={12} class="me-1" /> Active Lectures
                    </div>
                    <div class="list-group list-group-flush small" style="max-height: 30vh; overflow-y: auto;">
                        {#each allLectures as lecture}
                            <label class="list-group-item d-flex align-items-start gap-2 cursor-pointer">
                                <input 
                                    type="checkbox" 
                                    class="form-check-input mt-1 flex-shrink-0" 
                                    checked={includedLectureIds.includes(lecture.id)}
                                    onchange={() => toggleLecture(lecture.id)}
                                    disabled={updatingContext}
                                />
                                <div class="text-truncate">
                                    <div class="fw-bold">{lecture.title}</div>
                                    <div class="text-muted" style="font-size: 0.65rem;">
                                        {lecture.status === 'ready' ? 'Ready to study' : 'Preparing...'}
                                    </div>
                                </div>
                            </label>
                        {/each}
                        {#if allLectures.length === 0}
                            <div class="p-3 text-center text-muted">No lectures available.</div>
                        {/if}
                    </div>
                </div>

                <h3>Session Details</h3>
                <div class="well mb-4 small">
                    <table class="table table-sm table-borderless m-0">
                        <tbody>
                            <tr>
                                <td style="width: 40%"><strong>Status</strong></td>
                                <td><span class="badge bg-success">Ready to study</span></td>
                            </tr>
                            <tr>
                                <td><strong>Scope</strong></td>
                                <td>{includedLectureIds.length} Lectures</td>
                            </tr>
                            <tr>
                                <td><strong>Started</strong></td>
                                <td>{new Date(session.created_at).toLocaleDateString()}</td>
                            </tr>
                        </tbody>
                    </table>
                </div>

                {#if otherSessions.length > 0}
                    <h3>Other Chats</h3>
                    <div class="linkTiles tileSizeMd">
                        {#each otherSessions as other}
                            <Tile href="/exams/{examId}/chat/{other.id}" icon="談" title={other.title || 'Untitled Chat'}>
                                {#snippet description()}
                                    Switch to this study session.
                                {/snippet}
                            </Tile>
                        {/each}
                    </div>
                {/if}
            </div>

            <!-- Main Content: Chat -->
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
                            <span class="glyphicon m-0"><Send size={18} /></span>
                        </button>
                    </div>
                </form>

                <div class="chat-viewport mb-5" bind:this={messageContainer} style="height: 65vh; overflow-y: auto;">
                    {#if (!messages || messages.length === 0) && !streamingMessage}
                        <div class="well bg-white text-center p-5 text-muted">
                            <Bot size={48} class="mb-3 opacity-25" />
                            <p>I'm your dedicated study assistant. Select the lectures you want me to use from the sidebar, and ask a question above!</p>
                        </div>
                    {/if}

                    {#each messages ?? [] as msg}
                        {#if msg.role === 'assistant'}
                            <div class="char-results mb-4">
                                <div class="well bg-white p-4 shadow-sm border-start border-4 border-success">
                                    <div class="row">
                                        <div lang="ja" class="col-xl-1 col-md-2 text-center d-none d-md-block">
                                            <Bot size={32} class="text-success" />
                                        </div>
                                        <div class="col-xl-11 col-md-10">
                                            <div class="small fw-bold text-uppercase text-success mb-2">Assistant</div>
                                            <div class="message-content wordBriefContent">
                                                {msg.content}
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        {:else}
                            <div class="wordBrief mb-4 ms-md-5">
                                <div class="bg-light border-start border-4 border-primary p-3 shadow-sm">
                                    <div class="small fw-bold text-primary text-uppercase mb-2" style="font-size: 0.7rem; letter-spacing: 0.1em;">Question</div>
                                    <div class="message-content wordBriefTitle">
                                        {msg.content}
                                    </div>
                                </div>
                            </div>
                        {/if}
                    {/each}

                    {#if streamingMessage}
                        <div class="char-results mb-4">
                            <div class="well bg-white p-4 shadow-sm border-start border-4 border-success">
                                <div class="row">
                                    <div lang="ja" class="col-xl-1 col-md-2 text-center d-none d-md-block">
                                        <Bot size={32} class="text-success" />
                                    </div>
                                    <div class="col-xl-11 col-md-10">
                                        <div class="small fw-bold text-uppercase text-success mb-2">Assistant</div>
                                        <div class="message-content wordBriefContent">
                                            {streamingMessage}
                                            <span class="village-spinner d-inline-block ms-2" style="width: 1rem; height: 1rem;"></span>
                                        </div>
                                    </div>
                                </div>
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
</style>