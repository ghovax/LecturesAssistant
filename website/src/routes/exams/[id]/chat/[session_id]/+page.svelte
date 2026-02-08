<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/state';
	import { apiFetch } from '$lib/api';
	import { notifications } from '$lib/notifications';

	const examID = $derived(page.params.id);
	const sessionID = $derived(page.params.session_id);

	let messages = $state([]);
	let contextConfiguration = $state({ included_lecture_ids: [], included_tool_ids: [] });
	let lectures = $state([]);
	let loading = $state(true);
	let newMessage = $state('');
	let sending = $state(false);
	let socket = $state(null);
	let streamingMessage = $state('');

	async function fetchData() {
		try {
			const res = await apiFetch(`/api/chat/sessions/details?session_id=${sessionID}&exam_id=${examID}`);
			messages = res.messages || [];
			contextConfiguration = res.context;
			lectures = await apiFetch(`/api/lectures?exam_id=${examID}`);
		} catch (e) {
			console.error(e);
		} finally {
			loading = false;
		}
	}

	async function updateContext() {
		try {
			await apiFetch('/api/chat/sessions/context', {
				method: 'PATCH',
				body: { session_id: sessionID, ...contextConfiguration }
			});
		} catch (e) {
			notifications.error('Context update failed: ' + e.message);
		}
	}

	async function sendMessage() {
		if (!newMessage.trim() || sending) return;
		sending = true;
		const userMsg = { role: 'user', content: newMessage, created_at: new Date().toISOString() };
		messages = [...messages, userMsg];
		const content = newMessage;
		newMessage = '';

		try {
			await apiFetch('/api/chat/messages', {
				method: 'POST',
				body: { session_id: sessionID, content }
			});
			// The actual response will come via WebSocket
		} catch (e) {
			notifications.error('Failed to send message: ' + e.message);
			sending = false;
		}
	}

	function setupWebSocket() {
		const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
		const host = window.location.host;
		socket = new WebSocket(`${protocol}//${host}/api/socket`);

		socket.onopen = () => {
			socket.send(JSON.stringify({ type: 'subscribe', channel: `chat:${sessionID}` }));
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

	onMount(() => {
		fetchData();
		setupWebSocket();
	});

	onDestroy(() => {
		if (socket) socket.close();
	});
</script>

<div class="chat-container" style="display: grid; grid-template-columns: 1fr 300px; gap: var(--space-lg); height: calc(100vh - 120px);">
	<div style="display: flex; flex-direction: column; min-width: 0;">
		<h1>Chat Assistant</h1>
		
		<div style="flex: 1; overflow-y: auto; padding: var(--space-md); border: 1px solid var(--border-color); margin-bottom: var(--space-md); border-radius: var(--radius);">
			{#each messages as msg}
				<div class="card" style="margin-bottom: var(--space-md); padding: var(--space-md); background: {msg.role === 'user' ? '#f0f7ff' : '#fff'}">
					<small><strong>{msg.role.toUpperCase()}</strong></small>
					<p style="white-space: pre-wrap; margin: var(--space-sm) 0 0 0;">{msg.content}</p>
				</div>
			{/each}
			{#if streamingMessage}
				<div class="card" style="margin-bottom: var(--space-md); padding: var(--space-md); background: #fff; border-style: dashed;">
					<small><strong>ASSISTANT (streaming...)</strong></small>
					<p style="white-space: pre-wrap; margin: var(--space-sm) 0 0 0;">{streamingMessage}</p>
				</div>
			{/if}
		</div>

		<form onsubmit={(e) => { e.preventDefault(); sendMessage(); }} style="display: flex; gap: var(--space-sm);">
			<input type="text" bind:value={newMessage} placeholder="Type your message..." disabled={sending} />
			<button type="submit" disabled={sending || !newMessage.trim()}>Send</button>
		</form>
	</div>

	<aside class="chat-sidebar" style="width: 100%; border-left: 1px solid var(--border-color); padding-left: var(--space-md); border-right: none;">
		<h3>Source Materials</h3>
		<p style="font-size: 0.8em; color: #666;">Select which lectures the AI should use for this chat.</p>
		
		<div style="margin-top: var(--space-md);">
			{#each lectures as l}
				<label style="display: block; margin-bottom: var(--space-sm); font-size: 0.9em;">
					<input 
						type="checkbox" 
						value={l.id} 
						checked={contextConfiguration.included_lecture_ids.includes(l.id)}
						onchange={(e) => {
							if (e.target.checked) {
								contextConfiguration.included_lecture_ids = [...contextConfiguration.included_lecture_ids, l.id];
							} else {
								contextConfiguration.included_lecture_ids = contextConfiguration.included_lecture_ids.filter(id => id !== l.id);
							}
							updateContext();
						}}
					/>
					{l.title}
				</label>
			{/each}
		</div>
	</aside>
</div>
