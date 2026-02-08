<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { apiFetch } from '$lib/api';
	import { socketManager } from '$lib/socket';
	import { goto } from '$app/navigation';
	import { Circle, X, Check } from 'lucide-svelte';

	const examID = $derived(page.params.id);
	const lectureID = $derived(page.params.lecture_id);

	let lecture = $state(null);
	let transcript = $state(null);
	let documents = $state([]);
	let recordings = $state([]);
	let loading = $state(true);
	let error = $state(null);

	// Tool generation state
	let toolType = $state('guide');
	let toolLength = $state('medium');
	let generating = $state(false);

	async function fetchData() {
		try {
			const [newLec, newTrans, newDocs, newMedia] = await Promise.all([
				apiFetch(`/api/lectures/details?lecture_id=${lectureID}&exam_id=${examID}`),
				apiFetch(`/api/transcripts?lecture_id=${lectureID}`).catch(() => null),
				apiFetch(`/api/documents?lecture_id=${lectureID}`),
				apiFetch(`/api/media?lecture_id=${lectureID}`)
			]);
			lecture = newLec;
			transcript = newTrans;
			documents = newDocs;
			recordings = newMedia;
		} catch (e) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	async function deleteLecture() {
		if (!confirm('Are you sure you want to delete this lecture?')) return;
		try {
			await apiFetch('/api/lectures', {
				method: 'DELETE',
				body: { lecture_id: lectureID, exam_id: examID }
			});
			goto(`/exams/${examID}`);
		} catch (e) {
			console.error('Delete failed:', e);
		}
	}

	async function generateTool() {
		generating = true;
		try {
			await apiFetch('/api/tools', {
				method: 'POST',
				body: {
					exam_id: examID,
					lecture_id: lectureID,
					type: toolType,
					length: toolLength
				}
			});
			window.location.href = '/jobs';
		} catch (e) {
			alert('Failed to start generation: ' + e.message);
		} finally {
			generating = false;
		}
	}

	function formatTime(ms: number) {
		const totalSeconds = Math.floor(ms / 1000);
		const minutes = Math.floor(totalSeconds / 60);
		const seconds = totalSeconds % 60;
		return `${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
	}

	onMount(() => {
		fetchData();
		const unsubscribe = socketManager.subscribe(`lecture:${lectureID}`, (msg) => {
			if (msg.type === 'lecture:updated') {
				fetchData();
			}
		});
		return unsubscribe;
	});
</script>

{#if loading}
	<p>Loading lecture details...</p>
{:else if error}
	<div class="error">{error}</div>
	<a href="/exams/{examID}" class="button">Back to Course</a>
{:else if lecture}
	<div style="display: flex; justify-content: space-between; align-items: flex-start; gap: var(--space-lg); flex-wrap: wrap;">
		<div style="flex: 1; min-width: 300px;">
			<h1 style="margin-bottom: var(--space-xs);">{lecture.title}</h1>
			<p style="margin: 0; color: #666;">{lecture.description || 'No description provided'}</p>
			
			{#if lecture.status === 'processing'}
				<p style="font-size: 13px; color: var(--accent-color); margin-top: var(--space-md); display: flex; align-items: center; gap: var(--space-sm);">
					<Circle size={10} fill="currentColor" />
					We're currently processing your materials. This may take a few minutes.
				</p>
			{:else if lecture.status === 'failed'}
				<p style="font-size: 13px; color: var(--error-color); margin-top: var(--space-md); display: flex; align-items: center; gap: var(--space-sm);">
					<X size={12} strokeWidth={3} />
					There was an issue processing this lecture. Please check your files.
				</p>
			{:else}
				<p style="font-size: 13px; color: #226622; margin-top: var(--space-md); display: flex; align-items: center; gap: var(--space-sm);">
					<Check size={12} strokeWidth={3} />
					All set! Your materials are ready.
				</p>
			{/if}
		</div>
		<div style="display: flex; gap: var(--space-sm); align-items: center; margin-top: var(--space-xs);">
			<a href="/exams/{examID}" class="button" style="min-width: 140px;">Back to Course</a>
			<button onclick={deleteLecture} class="danger" style="min-width: 140px;">Delete Lecture</button>
		</div>
	</div>

	<div class="lecture-grid" style="margin-top: var(--space-xl); display: grid; grid-template-columns: 2fr 1fr; gap: var(--space-xl); align-items: start;">
		<div>
			{#if recordings.length > 0}
				<div style="margin-bottom: var(--space-lg);">
					<h2>Recordings</h2>
					<div style="display: flex; flex-direction: column; gap: var(--space-sm); margin-top: var(--space-md);">
						{#each recordings as recording}
							<div class="badge" style="display: flex; justify-content: space-between; align-items: center; padding: var(--space-sm) var(--space-md); text-transform: none; font-weight: normal; letter-spacing: normal;">
								<span style="font-family: var(--font-mono); font-size: 13px;">{recording.original_filename || recording.file_path.split('/').pop()}</span>
								<span style="color: #888; font-size: 12px;">{formatTime(recording.duration_milliseconds)}</span>
							</div>
						{/each}
					</div>
				</div>
			{/if}

			<h2>Transcript</h2>
			{#if transcript && transcript.segments}
				<div style="max-height: 70vh; overflow-y: auto; padding-top: var(--space-md);">
					{#each transcript.segments as segment}
						<p style="margin-bottom: var(--space-sm); display: flex; align-items: flex-start; gap: var(--space-md);">
							<span style="color: #888; font-family: var(--font-mono); font-size: 12px; padding-top: 2px; flex-shrink: 0; min-width: 45px;">
								{formatTime(segment.start_millisecond)}
							</span>
							<span style="flex: 1;">{segment.text}</span>
						</p>
					{/each}
				</div>
			{:else}
				<p style="color: #666; margin-top: var(--space-md);">Transcript is not available or still processing.</p>
			{/if}
		</div>

		<div>
			<h2>Reference Materials</h2>
			<div style="display: flex; flex-direction: column; gap: var(--space-md); margin-top: var(--space-md);">
				{#each documents as doc}
					<div style="padding-bottom: var(--space-md); border-bottom: 1px solid var(--border-color);">
						<div style="font-weight: 600; font-size: 14px;">{doc.title}</div>
						<div style="font-size: 12px; color: #666; margin: var(--space-xs) 0 var(--space-sm) 0; display: flex; align-items: center; gap: var(--space-sm); min-height: 18px;">
							{#if doc.extraction_status === 'pending' || doc.extraction_status === 'processing'}
								<span style="color: var(--accent-color); display: flex; align-items: center; gap: var(--space-sm);">
									<Circle size={8} fill="currentColor" />
									Preparing pages...
								</span>
							{:else if doc.extraction_status === 'failed'}
								<span style="color: var(--error-color); display: flex; align-items: center; gap: var(--space-sm);">
									<X size={10} strokeWidth={3} />
									Failed to extract
								</span>
							{:else}
								<span style="color: #226622; display: flex; align-items: center; gap: var(--space-sm);">
									<Check size={12} strokeWidth={3} />
									<span>{doc.page_count} {doc.page_count === 1 ? 'page' : 'pages'} • All set!</span>
								</span>
							{/if}
						</div>
						<a href="/exams/{examID}/lectures/{lectureID}/documents/{doc.id}" class="button" style="min-width: auto; height: 28px; padding: 0 12px; font-size: 12px;">View Pages</a>
					</div>
				{:else}
					<p style="color: #666;">No reference materials found.</p>
				{/each}
			</div>

			<div style="margin-top: var(--space-xl); padding-top: var(--space-lg); border-top: 1px solid var(--border-color);">
				<h3>Study Materials</h3>
				<p style="font-size: 13px; color: #666; margin-bottom: var(--space-md);">Create a guide, flashcards, or a quiz to help you learn this lecture.</p>
				
				<div style="display: flex; flex-direction: column; gap: var(--space-md);">
					<label style="font-size: 13px; display: block;">
						What would you like to create?
						<select bind:value={toolType} style="margin-top: var(--space-xs);">
							<option value="guide">Study Guide</option>
							<option value="flashcard">Flashcards</option>
							<option value="quiz">Practice Quiz</option>
						</select>
					</label>

					{#if toolType === 'guide'}
						<label style="font-size: 13px; display: block;">
							How detailed should it be?
							<select bind:value={toolLength} style="margin-top: var(--space-xs);">
								<option value="short">Summary</option>
								<option value="medium">Standard</option>
								<option value="long">Detailed</option>
							</select>
						</label>
					{/if}

					<button 
						onclick={generateTool} 
						disabled={generating || lecture.status !== 'ready'} 
						style="margin-top: var(--space-xs); width: 100%;"
					>
						{generating ? 'Starting...' : 'Generate'}
					</button>
					
					{#if lecture.status !== 'ready'}
						<p style="font-size: 11px; color: #888; text-align: center; margin-top: var(--space-xs);">You can generate materials once the lecture is fully processed.</p>
					{/if}
				</div>
			</div>
		</div>
	</div>
{/if}

<style>
	@media (max-width: 1024px) {
		.lecture-grid {
			grid-template-columns: 1fr !important;
		}
	}
</style>
