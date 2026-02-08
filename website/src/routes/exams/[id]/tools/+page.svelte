<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { apiFetch } from '$lib/api';
	import { notifications } from '$lib/notifications';

	const examID = $derived(page.params.id);

	let tools = $state([]);
	let lectures = $state([]);
	let loading = $state(true);
	let error = $state(null);

	// Form state
	let selectedLecture = $state('');
	let type = $state('guide');
	let length = $state('medium');
	let creating = $state(false);

	async function fetchData() {
		try {
			tools = await apiFetch(`/api/tools?exam_id=${examID}`);
			lectures = await apiFetch(`/api/lectures?exam_id=${examID}`);
		} catch (e) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	async function createTool() {
		creating = true;
		try {
			const res = await apiFetch('/api/tools', {
				method: 'POST',
				body: {
					exam_id: examID,
					lecture_id: selectedLecture,
					type,
					length
				}
			});
			notifications.success('Generation job created. You can track progress in the Activity page.');
			window.location.href = '/jobs';
		} catch (e) {
			notifications.error('Failed: ' + e.message);
		} finally {
			creating = false;
		}
	}

	async function deleteTool(toolID: string) {
		if (!confirm('Delete this tool?')) return;
		try {
			await apiFetch('/api/tools', {
				method: 'DELETE',
				body: { tool_id: toolID, exam_id: examID }
			});
			tools = tools.filter(t => t.id !== toolID);
		} catch (e) {
			notifications.error('Failed: ' + e.message);
		}
	}

	onMount(fetchData);
</script>

<div style="display: flex; justify-content: space-between; align-items: flex-start; gap: var(--space-lg); margin-bottom: var(--space-lg); flex-wrap: wrap;">
	<h1>Study Tools</h1>
	<a href="/exams/{examID}" class="button" style="min-width: 140px;">Back to Course</a>
</div>

{#if loading}
	<p>Loading tools...</p>
{:else if error}
	<div class="error">{error}</div>
{:else}
	<h2 style="margin-top: var(--space-xl);">Existing Tools</h2>
	<table>
		<thead>
			<tr>
				<th>Title</th>
				<th>Type</th>
				<th>Created At</th>
				<th>Actions</th>
			</tr>
		</thead>
		<tbody>
			{#each tools as tool}
				<tr>
					<td>{tool.title}</td>
					<td>{tool.type}</td>
					<td>{new Date(tool.created_at).toLocaleString()}</td>
					<td style="display: flex; gap: var(--space-md); align-items: center;">
						<a href="/exams/{examID}/tools/{tool.id}">View</a>
						<button onclick={() => deleteTool(tool.id)} class="danger" style="border: none; padding: 0; min-width: auto; height: auto; background: transparent;">Delete</button>
					</td>
				</tr>
			{:else}
				<tr><td colspan="4">No tools generated yet.</td></tr>
			{/each}
		</tbody>
	</table>
{/if}
