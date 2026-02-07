<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { apiFetch } from '$lib/api';

	const examID = $derived(page.params.id);

	let sessions = $state([]);
	let loading = $state(true);
	let error = $state(null);
	let newTitle = $state('');
	let creating = $state(false);

	async function fetchData() {
		try {
			sessions = await apiFetch(`/api/chat/sessions?exam_id=${examID}`);
		} catch (e) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	async function createSession() {
		creating = true;
		try {
			const session = await apiFetch('/api/chat/sessions', {
				method: 'POST',
				body: { exam_id: examID, title: newTitle || 'New Chat' }
			});
			window.location.href = `/exams/${examID}/chat/${session.id}`;
		} catch (e) {
			alert('Failed: ' + e.message);
		} finally {
			creating = false;
		}
	}

	onMount(fetchData);
</script>

<div style="display: flex; justify-content: space-between; align-items: flex-start; gap: var(--space-lg); margin-bottom: var(--space-lg); flex-wrap: wrap;">
	<h1>Chat Sessions</h1>
	<a href="/exams/{examID}" class="button" style="min-width: 140px;">Back to Course</a>
</div>

<div class="card" style="margin-top: 24px;">
	<h3>New Session</h3>
	<div style="display: flex; gap: 8px;">
		<input type="text" bind:value={newTitle} placeholder="Chat Title..." />
		<button onclick={createSession} disabled={creating}>{creating ? 'Creating...' : 'Create'}</button>
	</div>
</div>

{#if loading}
	<p>Loading sessions...</p>
{:else if error}
	<div class="error">{error}</div>
{:else}
	<table style="margin-top: 24px;">
		<thead>
			<tr>
				<th>Title</th>
				<th>Last Active</th>
				<th>Actions</th>
			</tr>
		</thead>
		<tbody>
			{#each sessions as s}
				<tr>
					<td>{s.title || 'Untitled Chat'}</td>
					<td>{new Date(s.updated_at).toLocaleString()}</td>
					<td>
						<a href="/exams/{examID}/chat/{s.id}">Open Chat</a>
					</td>
				</tr>
			{:else}
				<tr><td colspan="3">No chat sessions yet.</td></tr>
			{/each}
		</tbody>
	</table>
{/if}
