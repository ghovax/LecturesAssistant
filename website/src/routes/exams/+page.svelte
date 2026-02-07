<script lang="ts">
	import { onMount } from 'svelte';

	let exams = $state([]);
	let loading = $state(true);
	let error = $state(null);

	async function fetchExams() {
		try {
			const res = await fetch('/api/exams', {
				headers: {
					'X-Requested-With': 'XMLHttpRequest'
				}
			});
			if (!res.ok) throw new Error('Failed to fetch exams');
			const json = await res.json();
			exams = json.data || [];
		} catch (e) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	onMount(fetchExams);
</script>

<h1>Exams</h1>

{#if loading}
	<p>Loading exams...</p>
{:else if error}
	<div class="error">{error}</div>
{:else}
	<div class="card" style="display: flex; gap: 8px;">
		<button onclick={fetchExams}>Refresh</button>
		<a href="/exams/create" class="button">Create New Exam</a>
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
