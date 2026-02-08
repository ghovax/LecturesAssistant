<script lang="ts">
	import { page } from '$app/state';
	import { apiFetch } from '$lib/api';
	import { goto } from '$app/navigation';

	let examID = $derived(page.params.id);
	let title = $state('');
	let description = $state('');
	let date = $state('');
	let mediaFiles = $state([]);
	let documentFiles = $state([]);
	let saving = $state(false);
	let error = $state(null);
	let uploadStatus = $state('');

	async function uploadAndCreate() {
		saving = true;
		error = null;
		uploadStatus = 'Initializing...';
		try {
			// In this developer tool, we'll use direct multipart upload for simplicity, 
			// as the server supports it in handleCreateLecture.
			const formData = new FormData();
			formData.append('exam_id', examID);
			formData.append('title', title);
			formData.append('description', description);
			formData.append('specified_date', date);

			for (let file of mediaFiles) {
				formData.append('media', file);
			}
			for (let file of documentFiles) {
				formData.append('documents', file);
			}

			uploadStatus = 'Uploading files and creating lecture... Please wait, for larger files may take longer...';
			const lecture = await apiFetch('/api/lectures', {
				method: 'POST',
				body: formData
			});

			goto(`/exams/${examID}/lectures/${lecture.id}`);
		} catch (e) {
			error = e.message;
			uploadStatus = '';
		} finally {
			saving = false;
		}
	}

	function handleFileChange(event: any, type: 'media' | 'document') {
		const files = Array.from(event.target.files);
		if (type === 'media') {
			mediaFiles = [...mediaFiles, ...files];
		} else {
			documentFiles = [...documentFiles, ...files];
		}
	}
</script>

<div style="display: flex; justify-content: space-between; align-items: flex-start; gap: var(--space-lg); margin-bottom: var(--space-lg);">
	<h1>Add Lecture</h1>
	<a href="/exams/{examID}" class="button" style="min-width: 140px;">Back to Course</a>
</div>

{#if error}
	<div class="error">{error}</div>
{/if}

<div style="max-width: 800px;">
	<form onsubmit={(e) => { e.preventDefault(); uploadAndCreate(); }}>
		<div style="margin-bottom: var(--space-xl);">
			<h3>Basic Information</h3>
			<label>
				Lecture Title
				<input type="text" bind:value={title} required placeholder="e.g. Introduction to Cells" />
			</label>
			<label>
				Description (optional)
				<textarea bind:value={description} placeholder="What is this lecture about?"></textarea>
			</label>
			<label>
				Date (optional)
				<input type="date" bind:value={date} />
			</label>
		</div>

		<div style="margin-bottom: var(--space-xl);">
			<h3>Media Files</h3>
			<p style="font-size: 13px; color: #666; margin-bottom: var(--space-md);">Upload audio or video recordings of the lecture.</p>
			<div style="margin-bottom: var(--space-md);">
				<input type="file" multiple accept="audio/*,video/*" onchange={(e) => handleFileChange(e, 'media')} style="width: auto; border: none; background: transparent; padding: 0;" />
			</div>
			{#if mediaFiles.length > 0}
				<ul style="font-size: 13px; color: #666; margin-top: var(--space-sm); padding-left: var(--space-lg);">
					{#each mediaFiles as file}
						<li>{file.name} ({(file.size / 1024 / 1024).toFixed(2)} MB)</li>
					{/each}
				</ul>
			{/if}
		</div>

		<div style="margin-bottom: var(--space-xl);">
			<h3>Reference Materials</h3>
			<p style="font-size: 13px; color: #666; margin-bottom: var(--space-md);">Upload slides, PDFs, or other documents discussed in class.</p>
			<div style="margin-bottom: var(--space-md);">
				<input type="file" multiple accept=".pdf,.pptx,.docx" onchange={(e) => handleFileChange(e, 'document')} style="width: auto; border: none; background: transparent; padding: 0;" />
			</div>
			{#if documentFiles.length > 0}
				<ul style="font-size: 13px; color: #666; margin-top: var(--space-sm); padding-left: var(--space-lg);">
					{#each documentFiles as file}
						<li>{file.name} ({(file.size / 1024 / 1024).toFixed(2)} MB)</li>
					{/each}
				</ul>
			{/if}
		</div>

		{#if uploadStatus}
			<p style="color: var(--accent-color); margin-bottom: var(--space-md); font-size: 13px;">{uploadStatus}</p>
		{/if}

		<div style="margin-top: var(--space-xl); padding-top: var(--space-lg); border-top: 1px solid var(--border-color); display: flex; gap: var(--space-md);">
			<button type="submit" disabled={saving || !title} style="min-width: 160px;">
				{saving ? 'Uploading...' : 'Create Lecture'}
			</button>
			<a href="/exams/{examID}" class="button" style="background: transparent; min-width: 100px;">Cancel</a>
		</div>
	</form>
</div>

<style>
	label { display: block; margin-bottom: var(--space-md); }
	input[type="text"], input[type="date"], textarea { margin-top: var(--space-xs); }
	textarea { height: 100px; resize: vertical; }
	h3 { border-bottom: 1px solid var(--border-color); padding-bottom: var(--space-xs); margin-bottom: var(--space-md); }
</style>
