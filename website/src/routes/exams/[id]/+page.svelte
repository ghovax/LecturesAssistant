<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { apiFetch } from '$lib/api';
	import { notifications } from '$lib/notifications';
	import { goto } from '$app/navigation';
	import { Circle, X, Check } from 'lucide-svelte';

	let id = $derived(page.params.id);
	let exam = $state(null);
	let lectures = $state([]);
	let loading = $state(true);
	let error = $state(null);
	let isEditing = $state(false);
	let editTitle = $state('');
	let editDescription = $state('');
	let saving = $state(false);
	let activeTasks = $state([]);

	async function fetchData() {
		loading = true;
		try {
			const [examData, lecturesData, jobsData] = await Promise.all([
				apiFetch(`/api/exams/details?exam_id=${id}`),
				apiFetch(`/api/lectures?exam_id=${id}`),
				apiFetch(`/api/jobs?course_id=${id}`)
			]);
			exam = examData;
			lectures = lecturesData;
			activeTasks = (jobsData || []).filter(task => task.status === 'RUNNING' || task.status === 'PENDING');
			editTitle = exam.title;
			editDescription = exam.description || '';
		} catch (e) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	async function saveChanges() {
		saving = true;
		try {
			const updated = await apiFetch('/api/exams', {
				method: 'PATCH',
				body: {
					exam_id: id,
					title: editTitle,
					description: editDescription
				}
			});
			exam = updated;
			isEditing = false;
		} catch (e) {
			notifications.error('Failed to save: ' + e.message);
		} finally {
			saving = false;
		}
	}

	async function deleteCourse() {
		if (!confirm('Are you sure you want to delete this course and all its data?')) return;
		try {
			await apiFetch('/api/exams', {
				method: 'DELETE',
				body: { exam_id: id }
			});
			goto('/exams');
		} catch (e) {
			notifications.error('Delete failed: ' + e.message);
		}
	}

	async function deleteLecture(lectureID: string) {
		if (!confirm('Are you sure you want to delete this lecture?')) return;
		try {
			await apiFetch('/api/lectures', {
				method: 'DELETE',
				body: { lecture_id: lectureID, exam_id: id }
			});
			lectures = lectures.filter(l => l.id !== lectureID);
		} catch (e) {
			console.error('Delete failed:', e);
		}
	}

	onMount(fetchData);
</script>

{#if loading}
	<p>Loading exam details...</p>
{:else if error}
	<div class="error">{error}</div>
	<a href="/exams">Back to list</a>
{:else if exam}
	<div style="display: flex; justify-content: space-between; align-items: flex-start; gap: var(--space-lg); flex-wrap: wrap;">
		<div style="flex: 1; min-width: 300px;">
			{#if isEditing}
				<div style="display: flex; flex-direction: column; gap: var(--space-sm);">
					<input type="text" bind:value={editTitle} placeholder="Course Title" />
					<textarea bind:value={editDescription} placeholder="Course Description" style="min-height: 60px;"></textarea>
					<div style="display: flex; gap: var(--space-sm); margin-top: var(--space-xs);">
						<button onclick={saveChanges} disabled={saving} style="min-width: 80px;">{saving ? 'Saving...' : 'Save'}</button>
						<button onclick={() => isEditing = false} style="min-width: 80px; background: transparent;">Cancel</button>
					</div>
				</div>
			{:else}
				<h1 style="margin-bottom: var(--space-xs);">{exam.title}</h1>
				<p style="margin: 0; color: #666;">{exam.description || 'No description provided'}</p>
			{/if}
		</div>
		<div style="display: flex; gap: var(--space-sm); align-items: center; margin-top: var(--space-xs);">
			<a href="/exams" class="button" style="min-width: 140px;">Back to Courses</a>
			{#if !isEditing}
				<button onclick={() => isEditing = true} style="min-width: 100px;">Edit Info</button>
			{/if}
			<button onclick={deleteCourse} class="danger" style="min-width: 140px;">Delete Course</button>
		</div>
	</div>

	<hr />

	{#if activeTasks.length > 0}
		<div style="margin-bottom: var(--space-xl);">
			<h2>Active Tasks</h2>
			<div style="display: flex; flex-direction: column; gap: var(--space-sm);">
				{#each activeTasks as task}
					<div class="card" style="display: flex; justify-content: space-between; align-items: center; padding: var(--space-sm) var(--space-md); margin-bottom: 0;">
						<div style="display: flex; align-items: center; gap: var(--space-sm);">
							<span class="spin" style="color: var(--accent-color); display: inline-block;">●</span>
							<strong>{task.type.replace(/_/g, ' ')}</strong>
							<span style="font-size: 12px; color: #666;">{task.progress_message_text}</span>
						</div>
						<div style="display: flex; align-items: center; gap: var(--space-md);">
							<div style="width: 100px; height: 4px; background: #eee; border-radius: 2px; overflow: hidden;">
								<div style="width: {task.progress}%; height: 100%; background: var(--accent-color); transition: width 0.3s ease;"></div>
							</div>
							<span style="font-size: 12px; color: #888; width: 35px;">{task.progress}%</span>
						</div>
					</div>
				{/each}
			</div>
		</div>
	{/if}

	<div style="display: flex; justify-content: space-between; align-items: center; margin-top: var(--space-lg);">
		<h2>Lectures</h2>
		<a href="/exams/{id}/lectures/create" class="button">Add Lecture</a>
	</div>

	{#if lectures.length === 0}
		<p style="color: #666; margin-top: var(--space-md);">No lectures yet for this course.</p>
	{:else}
		<table>
			<thead>
				<tr>
					<th>Title</th>
					<th>Date</th>
					<th>Status</th>
					<th>Actions</th>
				</tr>
			</thead>
			<tbody>
				{#each lectures as lecture}
					<tr>
						<td><strong>{lecture.title}</strong></td>
						<td>{lecture.specified_date ? new Date(lecture.specified_date).toLocaleDateString() : 'N/A'}</td>
						<td>
							<div style="display: flex; align-items: center; gap: var(--space-xs);">
								{#if lecture.status === 'processing'}
									<Circle size={10} fill="var(--accent-color)" color="var(--accent-color)" />
									<span style="color: var(--accent-color); font-size: 12px;">Processing</span>
								{:else if lecture.status === 'failed'}
									<X size={12} strokeWidth={3} color="var(--error-color)" />
									<span style="color: var(--error-color); font-size: 12px;">Failed</span>
								{:else}
									<div style="display: flex; align-items: center; gap: var(--space-xs); color: #226622;">
										<Check size={12} strokeWidth={3} />
										<span class="badge" style="background: #f0fff0; border-color: #cceecc; color: inherit; border-style: solid; font-weight: normal; padding: 2px 4px;">All set!</span>
									</div>
								{/if}
							</div>
						</td>
						<td>
							<div style="display: flex; gap: var(--space-md); align-items: center;">
								<a href="/exams/{id}/lectures/{lecture.id}">View Details</a>
								<button onclick={() => deleteLecture(lecture.id)} class="danger" style="border: none; padding: 0; min-width: auto; height: auto; background: transparent; font-size: 13px;">Delete</button>
							</div>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	{/if}

	<div style="margin-top: var(--space-xl);">
		<h2>Study Tools and Chat</h2>
		<div style="display: flex; gap: var(--space-md); margin-top: var(--space-md);">
			<a href="/exams/{id}/chat" class="button" style="flex: 1;">
				Chat Assistant
			</a>
			<a href="/exams/{id}/tools" class="button" style="flex: 1;">
				Study Guides and Quizzes
			</a>
		</div>
	</div>
{/if}
