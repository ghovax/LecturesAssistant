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

			uploadStatus = 'Uploading files and creating lecture... (This may take a while for large files)';
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

<h1>Add Lecture</h1>

{#if error}
	<div class="error">{error}</div>
{/if}

<div class="card">
	<form onsubmit={(e) => { e.preventDefault(); uploadAndCreate(); }}>
		<label>
			Lecture Title
			<input type="text" bind:value={title} required placeholder="e.g. Introduction to Cells" />
		</label>
		<label>
			Description (optional)
			<textarea bind:value={description}></textarea>
		</label>
		<label>
			Date (optional)
			<input type="date" bind:value={date} />
		</label>

		<div style="margin-top: 24px;">
			<h3>Media (Audio/Video)</h3>
			<input type="file" multiple accept="audio/*,video/*" onchange={(e) => handleFileChange(e, 'media')} />
			<ul style="font-size: 0.9em; color: #666;">
				{#each mediaFiles as file}
					<li>{file.name} ({(file.size / 1024 / 1024).toFixed(2)} MB)</li>
				{/each}
			</ul>
		</div>

		<div style="margin-top: 16px;">
			<h3>Reference Documents (PDF/PPTX/DOCX)</h3>
			<input type="file" multiple accept=".pdf,.pptx,.docx" onchange={(e) => handleFileChange(e, 'document')} />
			<ul style="font-size: 0.9em; color: #666;">
				{#each documentFiles as file}
					<li>{file.name} ({(file.size / 1024 / 1024).toFixed(2)} MB)</li>
				{/each}
			</ul>
		</div>

		{#if uploadStatus}
			<p style="color: var(--accent-color); margin-top: 16px;">{uploadStatus}</p>
		{/if}

		<div style="margin-top: 24px;">
			<button type="submit" disabled={saving || !title}>
				{saving ? 'Uploading...' : 'Create Lecture'}
			</button>
			<a href="/exams/{examID}" style="margin-left: 8px;">Cancel</a>
		</div>
	</form>
</div>

<style>
	label { display: block; margin-bottom: 16px; }
	input[type="text"], input[type="date"], textarea { margin-top: 4px; }
	textarea { height: 80px; resize: vertical; }
</style>
