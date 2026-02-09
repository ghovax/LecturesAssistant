<script lang="ts">
    import { onMount, onDestroy } from 'svelte';
    import { page } from '$app/state';
    import { api } from '$lib/api/client';
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';
    import { Send, User, Bot, Sparkles } from 'lucide-svelte';

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
            socket?.send(JSON.stringify({
                type: 'subscribe',
                channel: `chat:${session.id}`
            }));
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
        if (!input.trim() || sending) return;
        
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

    <div class="chat-container">
        <div class="chat-header">
            <Sparkles size={20} class="text-success" />
            <span class="ms-2 fw-bold">Study Chat</span>
        </div>

        <div class="messages-list" id="messageList">
            {#each messages as msg}
                <div class="message-wrapper {msg.role}">
                    <div class="avatar">
                        {#if msg.role === 'user'}<User size={16} />{:else}<Bot size={16} />{/if}
                    </div>
                    <div class="message-bubble">
                        {msg.content}
                    </div>
                </div>
            {/each}
            
            {#if streamingMessage}
                <div class="message-wrapper assistant">
                    <div class="avatar"><Bot size={16} /></div>
                    <div class="message-bubble">
                        {streamingMessage}
                    </div>
                </div>
            {/if}

            {#if sending && !streamingMessage}
                <div class="text-muted small ms-5 ps-2">Thinking...</div>
            {/if}
        </div>

        <div class="chat-input-area">
            <form onsubmit={(e) => { e.preventDefault(); sendMessage(); }} class="input-group">
                <input 
                    type="text" 
                    class="form-control" 
                    placeholder="Ask about the lecture..." 
                    bind:value={input}
                    disabled={sending}
                />
                <button type="submit" class="btn btn-success" disabled={sending || !input.trim()}>
                    <Send size={18} />
                </button>
            </form>
        </div>
    </div>
{/if}

<style>
    .chat-container {
        height: 70vh;
        display: flex;
        flex-direction: column;
        background: #fff;
        border: 1px solid #dee2e6;
        border-radius: 0.75rem;
        overflow: hidden;
        box-shadow: var(--tile-shadow);
    }

    .chat-header {
        padding: 1rem;
        background: #f8f9fa;
        border-bottom: 1px solid #dee2e6;
        display: flex;
        align-items: center;
    }

    .messages-list {
        flex-grow: 1;
        overflow-y: auto;
        padding: 1.5rem;
        display: flex;
        flex-direction: column;
        gap: 1rem;
    }

    .message-wrapper {
        display: flex;
        gap: 0.75rem;
        max-width: 85%;
    }

    .message-wrapper.user {
        flex-direction: row-reverse;
        align-self: flex-end;
    }

    .avatar {
        width: 32px;
        height: 32px;
        border-radius: 50%;
        background: #e9ecef;
        display: flex;
        align-items: center;
        justify-content: center;
        flex-shrink: 0;
    }

    .user .avatar { background: var(--primary-color); color: #fff; }
    .assistant .avatar { background: var(--success-color); color: #fff; }

    .message-bubble {
        padding: 0.75rem 1rem;
        border-radius: 1rem;
        font-size: 0.95rem;
        line-height: 1.4;
        white-space: pre-wrap;
    }

    .user .message-bubble {
        background: var(--primary-color);
        color: #fff;
        border-bottom-right-radius: 0.25rem;
    }

    .assistant .message-bubble {
        background: #f1f3f5;
        color: #333;
        border-bottom-left-radius: 0.25rem;
    }

    .chat-input-area {
        padding: 1rem;
        background: #fff;
        border-top: 1px solid #dee2e6;
    }

    .chat-input-area input {
        border-radius: 2rem;
        padding-left: 1.25rem;
    }

    .chat-input-area button {
        border-radius: 50%;
        width: 40px;
        height: 40px;
        padding: 0;
        display: flex;
        align-items: center;
        justify-content: center;
        margin-left: 0.5rem;
    }
</style>
