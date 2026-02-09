<script lang="ts">
    import { onMount, onDestroy } from 'svelte';
    import { page } from '$app/state';
    import { api } from '$lib/api/client';
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';
    import { Send, User, Bot, Sparkles, Search } from 'lucide-svelte';

    let examId = $derived(page.params.id);
    let exam = $state<any>(null);
    let session = $state<any>(null);
    let messages = $state<any[]>([]);
    let input = $state('');
    let loading = $state(true);
    let sending = $state(false);
    let socket: WebSocket | null = null;
    let streamingMessage = $state('');

    async function initChat() {
        try {
            exam = await api.getExam(examId);
            const sessions = await api.request('GET', `/chat/sessions?exam_id=${examId}`);
            
            if (sessions.length > 0) {
                const details = await api.request('GET', `/chat/sessions/details?session_id=${sessions[0].id}&exam_id=${examId}`);
                session = details.session;
                messages = details.messages;
            } else {
                session = await api.request('POST', '/chat/sessions', { exam_id: examId, title: 'Study Session' });
                messages = [];
            }
            
            setupWebSocket();
        } catch (e) {
            console.error(e);
        } finally {
            loading = false;
        }
    }

    function setupWebSocket() {
        const token = localStorage.getItem('session_token');
        socket = new WebSocket(`ws://localhost:3000/api/socket?session_token=${token}`);
        
        socket.onopen = () => {
            if (session?.id) {
                socket?.send(JSON.stringify({
                    type: 'subscribe',
                    channel: `chat:${session.id}`
                }));
            }
        };

        socket.onmessage = (event) => {
            const data = JSON.parse(event.data);
            if (data.type === 'chat:token') {
                streamingMessage = data.payload.accumulated_text;
            } else if (data.type === 'chat:complete') {
                messages = [...messages, data.payload];
                streamingMessage = '';
                sending = false;
            }
        };
    }

    async function sendMessage() {
        if (!input.trim() || sending || !session?.id) return;
        
        const userMsg = { role: 'user', content: input };
        messages = [...messages, userMsg];
        const content = input;
        input = '';
        sending = true;

        try {
            await api.request('POST', '/chat/messages', {
                session_id: session.id,
                content
            });
        } catch (e) {
            alert(e);
            sending = false;
        }
    }

    onMount(initChat);
    onDestroy(() => socket?.close());
</script>

{#if exam}
    <Breadcrumb items={[{ label: 'My Studies', href: '/exams' }, { label: exam.title, href: `/exams/${examId}` }, { label: 'Study Chat', active: true }]} />

    <h2>Study Assistant: {exam.title}</h2>

    <form onsubmit={(e) => { e.preventDefault(); sendMessage(); }} class="mb-4">
        <div class="input-group dictionary-style mb-3">
            <input 
                type="text" 
                class="form-control" 
                placeholder={session ? "Ask about your lectures or reference documents..." : "Initializing session..."} 
                bind:value={input}
                disabled={sending || !session}
            />
            <button class="btn btn-primary" type="submit" disabled={sending || !input.trim() || !session}>
                <span class="glyphicon"><Search size={18} /></span>
            </button>
        </div>
    </form>

    <div class="container-fluid p-0">
        <div class="row">
            <div class="col-lg-12">
                {#if messages.length === 0 && !streamingMessage}
                    <div class="well text-center p-5 text-muted">
                        <Bot size={48} class="mb-3 opacity-25" />
                        <p>I'm your dedicated study assistant. Ask a question above to begin exploring your materials!</p>
                    </div>
                {/if}

                {#each messages as msg}
                    {#if msg.role === 'assistant'}
                        <div class="char-results mb-4">
                            <div class="well bg-white p-4 shadow-sm border">
                                <div class="row">
                                    <div lang="ja" class="col-xl-1 col-md-2 text-center">
                                        <Bot size={40} class="text-success" />
                                    </div>
                                    <div class="col-xl-11 col-md-10">
                                        <h3 class="mt-0 border-0 pt-0">Assistant</h3>
                                        <div class="message-content fs-5" style="line-height: 1.6;">
                                            {msg.content}
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>
                    {:else}
                        <div class="wordBrief mb-4">
                            <div class="japaneseText px-3 py-2 bg-light border-start border-4 border-primary">
                                <table lang="ja" class="ruby">
                                    <tbody>
                                        <tr class="furi"><th>QUESTION</th></tr>
                                        <tr><td>{msg.content}</td></tr>
                                    </tbody>
                                </table>
                            </div>
                        </div>
                    {/if}
                {/each}

                {#if streamingMessage}
                    <div class="char-results mb-4">
                        <div class="well bg-white p-4 shadow-sm border">
                            <div class="row">
                                <div lang="ja" class="col-xl-1 col-md-2 text-center">
                                    <Bot size={40} class="text-success" />
                                </div>
                                <div class="col-xl-11 col-md-10">
                                    <h3 class="mt-0 border-0 pt-0">Assistant</h3>
                                    <div class="message-content fs-5">
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
{/if}

<style>
    .message-content {
        white-space: pre-wrap;
    }
    
    h3 {
        margin-bottom: 0.5rem;
        font-weight: bold;
    }
</style>