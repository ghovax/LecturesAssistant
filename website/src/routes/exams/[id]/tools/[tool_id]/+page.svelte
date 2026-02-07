<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { apiFetch } from '$lib/api';

	const examID = $derived(page.params.id);
	const toolID = $derived(page.params.tool_id);

	let tool = $state(null);
	let loading = $state(true);
	let error = $state(null);
	let exporting = $state(false);

	async function fetchData() {
		try {
			tool = await apiFetch(`/api/tools/details?tool_id=${toolID}&exam_id=${examID}`);
		} catch (e) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	async function exportTool(format: string) {
		exporting = true;
		try {
			const res = await apiFetch('/api/tools/export', {
				method: 'POST',
				body: { tool_id: toolID, exam_id: examID, format }
			});
			alert('Export job created: ' + res.job_id + '. Check the Jobs page for the download link.');
		} catch (e) {
			alert('Export failed: ' + e.message);
		} finally {
			exporting = false;
		}
	}

	onMount(fetchData);
</script>

{#if loading}
	<p>Loading tool...</p>
{:else if error}
	<div class="error">{error}</div>
{:else if tool}
	<div style="display: flex; justify-content: space-between; align-items: start;">
		<div>
			<h1>{tool.title}</h1>
			<p class="badge">{tool.type}</p>
		</div>
		<div style="display: flex; gap: 8px;">
			<button onclick={() => exportTool('pdf')} disabled={exporting}>Export PDF</button>
			<button onclick={() => exportTool('docx')} disabled={exporting}>Export DOCX</button>
			<button onclick={() => exportTool('md')} disabled={exporting}>Export MD</button>
			<a href="/exams/{examID}/tools" class="badge">Back</a>
		</div>
	</div>

	<div class="card" style="margin-top: 24px; white-space: pre-wrap; font-family: sans-serif; line-height: 1.6;">
		{#if tool.type === 'guide'}
			{tool.content}
		{:else}
			<!-- For flashcards/quizzes, the content is JSON -->
			<pre>{JSON.stringify(JSON.parse(tool.content), null, 2)}</pre>
		{/if}
	</div>
{/if}
