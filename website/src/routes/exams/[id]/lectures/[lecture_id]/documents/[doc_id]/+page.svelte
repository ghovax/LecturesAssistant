<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { apiFetch } from '$lib/api';

	const examID = $derived(page.params.id);
	const lectureID = $derived(page.params.lecture_id);
	const docID = $derived(page.params.doc_id);

	let document = $state(null);
	let pages = $state([]);
	let loading = $state(true);
	let error = $state(null);

	async function fetchData() {
		try {
			document = await apiFetch(`/api/documents/details?document_id=${docID}&lecture_id=${lectureID}`);
			pages = await apiFetch(`/api/documents/pages?document_id=${docID}&lecture_id=${lectureID}`);
		} catch (e) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	onMount(fetchData);
</script>

{#if loading}
	<p>Loading document pages...</p>
{:else if error}
	<div class="error">{error}</div>
	<a href="/exams/{examID}/lectures/{lectureID}">Back to Lecture</a>
{:else if document}
	<div style="display: flex; justify-content: space-between; align-items: flex-start; gap: var(--space-lg); margin-bottom: var(--space-lg); flex-wrap: wrap;">
		<h1>{document.title}</h1>
		<a href="/exams/{examID}/lectures/{lectureID}" class="button" style="min-width: 140px;">Back to Lecture</a>
	</div>
	
		<div style="margin-top: 24px; display: flex; flex-direction: column; gap: 32px;">
		{#each pages as p}
			<div class="card document-page-card" style="display: grid; grid-template-columns: 300px 1fr; gap: var(--space-lg);">
				<div>
					<img 
						src="/api/documents/pages/image?document_id=${docID}&lecture_id=${lectureID}&page_number=${p.page_number}" 
						alt="Page {p.page_number}"
						style="width: 100%; border: 1px solid var(--border-color);"
					/>
					<p style="text-align: center; margin-top: 8px;"><strong>Page {p.page_number}</strong></p>
				</div>
				<div>
					<h3>Extracted Information</h3>
					<div style="font-size: 0.9em; white-space: pre-wrap; background: #f9f9f9; padding: 12px; border-radius: var(--radius);">
						{p.extracted_text}
					</div>
				</div>
			</div>
		{/each}
	</div>
{/if}
