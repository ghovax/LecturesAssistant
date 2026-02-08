<script lang="ts">
	import { onMount } from 'svelte';
	import { apiFetch } from '$lib/api';

	let exams = $state([]);
	let loading = $state(true);
	let error = $state(null);

	async function fetchExams() {
		try {
			exams = await apiFetch('/api/exams');
		} catch (e) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	onMount(fetchExams);
</script>

<h1>Courses</h1>

{#if loading}
	<p>Loading courses...</p>
{:else if error}
	<div class="error">{error}</div>
{:else}
	<div style="display: flex; gap: var(--space-sm); margin-bottom: var(--space-lg);">
		<button onclick={fetchExams}>Refresh</button>
		<a href="/exams/create" class="button">Add New Course</a>
	</div>

	<table>
		<thead>
			<tr>
				<th>Title</th>
				<th>Description</th>
				<th>Created At</th>
				<th>Actions</th>
			</tr>
		</thead>
		<tbody>
			{#each exams as exam}
				<tr>
					<td>{exam.title}</td>
					<td>{exam.description || '-'}</td>
					<td>{new Date(exam.created_at).toLocaleString()}</td>
					<td>
						<a href="/exams/{exam.id}">View</a>
					</td>
				</tr>
			{:else}
				<tr>
					<td colspan="4">No exams found.</td>
				</tr>
			{/each}
		</tbody>
	</table>
{/if}
