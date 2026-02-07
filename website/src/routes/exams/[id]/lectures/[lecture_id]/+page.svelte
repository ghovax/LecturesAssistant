<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { apiFetch } from '$lib/api';

	const examID = $derived(page.params.id);
	const lectureID = $derived(page.params.lecture_id);

	let lecture = $state(null);
	let transcript = $state(null);
	let documents = $state([]);
	let loading = $state(true);
	let error = $state(null);

	async function fetchData() {
		try {
			lecture = await apiFetch(`/api/lectures/details?lecture_id=${lectureID}&exam_id=${examID}`);
			try {
				transcript = await apiFetch(`/api/transcripts?lecture_id=${lectureID}`);
			} catch (e) {
				// Transcript might not be ready yet
			}
			documents = await apiFetch(`/api/documents?lecture_id=${lectureID}`);
		} catch (e) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	onMount(fetchData);
</script>

{#if loading}
	<p>Loading lecture details...</p>
{:else if error}
	<div class="error">{error}</div>
	<a href="/exams/{examID}">Back to Exam</a>
{:else if lecture}
	<div style="display: flex; justify-content: space-between; align-items: flex-start; gap: var(--space-lg);">
		<div style="flex: 1;">
			<h1>{lecture.title}</h1>
			<p style="margin: 0;">{lecture.description || 'No description'}</p>
			<div class="badge" style="margin-top: var(--space-sm);">{lecture.status}</div>
		</div>
		<div style="margin-top: 4px;">
			<a href="/exams/{examID}" class="button">Back to Course</a>
		</div>
	</div>

	<div style="margin-top: var(--space-lg); display: grid; grid-template-columns: 2fr 1fr; gap: var(--space-lg);">
		<div>
			<h2>Transcript</h2>
			{#if transcript && transcript.segments}
				<div class="card" style="max-height: 600px; overflow-y: auto;">
					{#each transcript.segments as segment}
						<p>
							<small style="color: #888;">[{Math.floor(segment.start_millisecond / 1000)}s]</small>
							{segment.text}
						</p>
					{/each}
				</div>
			{:else}
				<div class="card">
					<p>Transcript is not available or still processing.</p>
				</div>
			{/if}
		</div>

		<div>
			<h2>Reference Materials</h2>
			{#each documents as doc}
				<div class="card" style="padding: var(--space-sm);">
					<strong>{doc.title}</strong>
					<p style="font-size: 0.8em; margin: var(--space-xs) 0;">{doc.page_count} pages • {doc.extraction_status}</p>
					<a href="/exams/{examID}/lectures/{lectureID}/documents/{doc.id}" style="font-size: 0.9em;">View Pages</a>
				</div>
			{:else}
				<p>No reference documents.</p>
			{/each}
		</div>
	</div>
{/if}
