<script lang="ts">
	import { apiFetch } from '$lib/api';
	import { goto } from '$app/navigation';

	let title = $state('');
	let description = $state('');
	let saving = $state(false);
	let error = $state(null);

	async function createExam() {
		saving = true;
		error = null;
		try {
			const exam = await apiFetch('/api/exams', {
				method: 'POST',
				body: { title, description }
			});
			goto(`/exams/${exam.id}`);
		} catch (e) {
			error = e.message;
		} finally {
			saving = false;
		}
	}
</script>

<h1>Create Exam</h1>

{#if error}
	<div class="error">{error}</div>
{/if}

<div class="card">
	<form onsubmit={(e) => { e.preventDefault(); createExam(); }}>
		<label>
			Title
			<input type="text" bind:value={title} required placeholder="e.g. Biology 101" />
		</label>
		<label>
			Description (optional)
			<textarea bind:value={description} placeholder="Short description of the course"></textarea>
		</label>
		<div style="margin-top: 16px;">
			<button type="submit" disabled={saving}>
				{saving ? 'Creating...' : 'Create Exam'}
			</button>
			<a href="/exams" style="margin-left: 8px;">Cancel</a>
		</div>
	</form>
</div>

<style>
	label { display: block; margin-bottom: 16px; }
	input, textarea { margin-top: 4px; }
	textarea { height: 100px; resize: vertical; }
</style>
