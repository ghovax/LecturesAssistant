<script lang="ts">
	import { onMount } from 'svelte';
	import { apiFetch } from '$lib/api';
	import { socketManager } from '$lib/socket';
	import { CheckCircle2, XCircle, Loader2, Clock, ChevronRight, Square } from 'lucide-svelte';

	let jobs = $state([]);
	let courses = $state([]);
	let lectures = $state([]);
	let loading = $state(true);
	let error = $state(null);

	async function fetchData() {
		try {
			const [jobsData, coursesData] = await Promise.all([
				apiFetch('/api/jobs'),
				apiFetch('/api/exams')
			]);
			
			jobs = jobsData || [];
			courses = coursesData || [];
			
			// For each unique lecture ID in jobs, fetch its details if possible
			const lectureIDs = [...new Set(jobs.map(j => {
				const p = JSON.parse(j.payload || '{}');
				return p.lecture_id;
			}).filter(id => !!id))];
			
			// Simple batch fetch for lecture names if needed, or just rely on IDs for now
			// To keep it simple and efficient, we'll just use the IDs or common course names
		} catch (e) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	function getCourseName(courseID: string) {
		return courses.find(c => c.id === courseID)?.title || 'Global Tasks';
	}

	// Helper to group jobs
	function groupJobs(jobsList: any[]) {
		const groups: Record<string, any[]> = {};
		
		for (const job of jobsList) {
			const payload = JSON.parse(job.payload || '{}');
			const courseID = payload.exam_id || 'global';
			const courseName = getCourseName(courseID);
			
			if (!groups[courseName]) groups[courseName] = [];
			groups[courseName].push(job);
		}
		
		return Object.entries(groups).sort((a, b) => a[0].localeCompare(b[0]));
	}

	const groupedActivity = $derived(groupJobs(jobs));

	onMount(() => {
		fetchData();
		const unsubscribe = socketManager.subscribe('jobs:all', () => fetchData()); 
		return unsubscribe;
	});

	function formatAction(type: string) {
		const actionMap: Record<string, string> = {
			'TRANSCRIBE_MEDIA': 'Transcribing Audio',
			'INGEST_DOCUMENTS': 'Reading Reference Materials',
			'BUILD_MATERIAL': 'Creating Study Guide',
			'PUBLISH_MATERIAL': 'Exporting to PDF/Docx',
			'DOWNLOAD_GOOGLE_DRIVE': 'Importing from Drive'
		};
		return actionMap[type] || type.replace(/_/g, ' ');
	}

	async function cancelJob(jobID: string) {
		if (!confirm('Stop this task?')) return;
		try {
			await apiFetch('/api/jobs', {
				method: 'DELETE',
				body: { job_id: jobID }
			});
			fetchData();
		} catch (e) {
			alert('Failed to cancel: ' + e.message);
		}
	}
</script>

<h1>Activity</h1>

{#if loading}
	<p>Loading activity...</p>
{:else if error}
	<div class="error">{error}</div>
{:else}
	{#each groupedActivity as [courseName, courseJobs]}
		<div style="margin-bottom: var(--space-xl);">
			<h2 style="font-size: 14px; color: #888; text-transform: uppercase; letter-spacing: 1px; border-bottom: 1px solid var(--border-color); padding-bottom: var(--space-xs); display: flex; align-items: center; gap: var(--space-sm);">
				<ChevronRight size={14} /> {courseName}
			</h2>
			
			<table style="margin-top: var(--space-sm);">
				<thead>
					<tr>
						<th>Action</th>
						<th>Status</th>
						<th>Progress</th>
						<th>Latest Update</th>
						<th style="text-align: right;">Cost</th>
						<th style="text-align: right;">Time</th>
					</tr>
				</thead>
				<tbody>
					{#each courseJobs as job}
						<tr>
							<td>
								<strong>{formatAction(job.type)}</strong>
							</td>
							<td>
								<div style="display: flex; align-items: center; gap: var(--space-sm);">
									{#if job.status === 'COMPLETED'}
										<CheckCircle2 size={14} color="#226622" />
									{:else if job.status === 'FAILED'}
										<XCircle size={14} color="var(--error-color)" />
									{:else if job.status === 'RUNNING'}
										<Loader2 size={14} color="var(--accent-color)" class="spin" />
									{:else if job.status === 'CANCELLED'}
										<XCircle size={14} color="#888" />
									{:else}
										<Clock size={14} color="#888" />
									{/if}
									<span style="font-size: 12px; font-weight: 600;">{job.status}</span>
									
									{#if job.status === 'RUNNING' || job.status === 'PENDING'}
										<button 
											onclick={() => cancelJob(job.id)} 
											class="danger" 
											style="height: 24px; min-width: 24px; width: 24px; padding: 0; margin-left: var(--space-sm);"
											title="Stop task"
										>
											<Square size={10} fill="currentColor" />
										</button>
									{/if}
								</div>
							</td>
							<td>
								<div style="width: 60px; height: 4px; background: #eee; border-radius: 2px; overflow: hidden; margin-bottom: 4px;">
									<div style="width: {job.progress}%; height: 100%; background: var(--accent-color); transition: width 0.3s ease;"></div>
								</div>
								<span style="font-size: 11px; color: #888;">{job.progress}%</span>
							</td>
							<td>
								<span style="font-size: 13px;">{job.progress_message_text || '-'}</span>
								{#if job.status === 'COMPLETED' && job.type === 'PUBLISH_MATERIAL'}
									{@const result = JSON.parse(job.result || '{}')}
									<a href="/api/exports/download?path={encodeURIComponent(result.file_path)}" download class="button" style="height: 24px; min-width: auto; padding: 0 8px; font-size: 11px; margin-top: 4px;">Download</a>
								{/if}
							</td>
							<td style="text-align: right; color: #666; font-size: 12px;">
								${job.estimated_cost?.toFixed(4) || '0.0000'}
							</td>
							<td style="text-align: right; color: #888; font-size: 12px;">
								{new Date(job.created_at).toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'})}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{:else}
		<p style="color: #666; margin-top: var(--space-xl); text-align: center;">No recent activity found.</p>
	{/each}
{/if}
