<script lang="ts">
	import { onMount } from 'svelte';

	let jobs = $state([]);
	let loading = $state(true);
	let error = $state(null);

	async function fetchJobs() {
		try {
			const res = await fetch('/api/jobs', {
				headers: {
					'X-Requested-With': 'XMLHttpRequest'
				}
			});
			if (!res.ok) throw new Error('Failed to fetch jobs');
			const json = await res.json();
			jobs = json.data || [];
		} catch (e) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		fetchJobs();
		const interval = setInterval(fetchJobs, 5000);
		return () => clearInterval(interval);
	});
</script>

<h1>Activity</h1>

{#if loading}
	<p>Loading activity...</p>
{:else if error}
	<div class="error">{error}</div>
{:else}
	<table>
		<thead>
			<tr>
				<th>Task</th>
				<th>Status</th>
				<th>Progress</th>
				<th>Update</th>
				<th>Cost</th>
				<th>Started</th>
			</tr>
		</thead>
		<tbody>
			{#each jobs as job}
				<tr>
					<td>{job.type.replace(/_/g, ' ')}</td>
					<td>
						<span class="badge">{job.status}</span>
						{#if job.status === 'COMPLETED' && job.type === 'PUBLISH_MATERIAL'}
							{@const result = JSON.parse(job.result || '{}')}
							<a href="/api/exports/download?path={encodeURIComponent(result.file_path)}" download style="display: block; font-size: 0.8em; margin-top: 4px;">Download</a>
						{/if}
					</td>
					<td>{job.progress}%</td>
					<td>{job.progress_message_text || '-'}</td>
					<td>${job.estimated_cost?.toFixed(4) || '0.0000'}</td>
					<td>{new Date(job.created_at).toLocaleString()}</td>
				</tr>
			{:else}
				<tr>
					<td colspan="6">No recent activity.</td>
				</tr>
			{/each}
		</tbody>
	</table>
{/if}
