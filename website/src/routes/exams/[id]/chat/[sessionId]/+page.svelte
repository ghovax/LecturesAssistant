<script lang="ts">
    import { onMount, onDestroy, tick } from 'svelte';
    import { page } from '$app/state';
    import { api } from '$lib/api/client';
    import { notifications } from '$lib/stores/notifications.svelte';
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';
    import Tile from '$lib/components/Tile.svelte';
    import { Send, User, Bot, Sparkles, Search, MessageSquare, BookOpen, Layers, Square, CheckSquare } from 'lucide-svelte';

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
                
                <div class="list-group shadow-sm small overflow-auto mb-4" style="max-height: 35vh;">
                    {#each allLectures as lecture}
                        <button 
                            onclick={() => toggleLecture(lecture.id)}
                            class="list-group-item list-group-item-action d-flex align-items-center gap-3 text-start {includedLectureIds.includes(lecture.id) ? 'active-context' : ''}"
                            disabled={updatingContext}
                        >
                            <div class="flex-shrink-0">
                                {#if includedLectureIds.includes(lecture.id)}
                                    <CheckSquare size={18} class="text-white" />
                                {:else}
                                    <Square size={18} class="text-muted" />
                                {/if}
                            </div>
                            <div class="flex-grow-1 overflow-hidden">
                                <div class="fw-bold text-truncate" title={lecture.title}>{lecture.title}</div>
                                <div class="{includedLectureIds.includes(lecture.id) ? 'text-white-50' : 'text-muted'} text-truncate" style="font-size: 0.75rem;">
                                    {lecture.status === 'ready' ? 'Ready to study' : 'Preparing...'}
                                </div>
                            </div>
                        </button>
                    {/each}
                    {#if allLectures.length === 0}
                        <div class="p-3 text-center text-muted">No lectures available.</div>
                    {/if}
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
                            <span class="glyphicon m-0"><Send size={18} /></span>
                        </button>
                    </div>
                </form>

                <div class="chat-viewport mb-5" bind:this={messageContainer} style="height: 65vh; overflow-y: auto;">
                    {#if (!messages || messages.length === 0) && !streamingMessage}
                        <div class="well bg-white text-center p-5 text-muted border shadow-sm">
                            <Bot size={48} class="mb-3 opacity-25" />
                            <p>I'm your dedicated study assistant. Select the lectures you want me to use from the sidebar, and ask a question above!</p>
                        </div>
                    {/if}

                    {#each messages ?? [] as msg}
                        <div class="well bg-white p-0 overflow-hidden mb-4 border shadow-sm {msg.role === 'user' ? 'ms-md-5 border-primary border-opacity-25' : 'me-md-5 border-success border-opacity-25'}">
                            <div class="bg-light px-4 py-2 border-bottom d-flex justify-content-between align-items-center">
                                <span class="fw-bold small {msg.role === 'user' ? 'text-primary' : 'text-success'}">
                                    {msg.role === 'user' ? 'Question' : 'Assistant'}
                                </span>
                                <span class="text-muted small" style="font-size: 0.7rem;">
                                    {new Date(msg.created_at || Date.now()).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                                </span>
                            </div>
                            <div class="p-4">
                                <div class="message-content {msg.role === 'user' ? 'fw-bold' : ''}" style="font-size: 1rem; line-height: 1.6;">
                                    {msg.content}
                                </div>
                            </div>
                        </div>
                    {/each}

                    {#if streamingMessage}
                        <div class="well bg-white p-0 overflow-hidden mb-4 border border-success border-opacity-25 shadow-sm me-md-5">
                            <div class="bg-light px-4 py-2 border-bottom d-flex justify-content-between align-items-center">
                                <span class="fw-bold small text-success">Assistant</span>
                                <span class="village-spinner" style="width: 0.8rem; height: 0.8rem;"></span>
                            </div>
                            <div class="p-4">
                                <div class="message-content" style="font-size: 1rem; line-height: 1.6;">
                                    {streamingMessage}
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
