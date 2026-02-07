<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { apiFetch } from '$lib/api';
	import { goto } from '$app/navigation';

	let id = $derived(page.params.id);
	let exam = $state(null);
	let lectures = $state([]);
	let loading = $state(true);
	let error = $state(null);
	let isEditing = $state(false);
	let editTitle = $state('');
	let editDescription = $state('');
	let saving = $state(false);

	async function fetchData() {
		loading = true;
		try {
			exam = await apiFetch(`/api/exams/details?exam_id=${id}`);
			lectures = await apiFetch(`/api/lectures?exam_id=${id}`);
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
			alert('Failed to save: ' + e.message);
		} finally {
			saving = false;
		}
	}

	async function deleteExam() {
		if (!confirm('Are you sure you want to delete this exam and all its data?')) return;
		try {
			await apiFetch('/api/exams', {
				method: 'DELETE',
				body: { exam_id: id }
			});
			goto('/exams');
		} catch (e) {
			alert('Delete failed: ' + e.message);
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
			<h1 style="margin-bottom: var(--space-xs);">{exam.title}</h1>
			<p style="margin: 0; color: #666;">{exam.description || 'No description provided'}</p>
		</div>
		<div style="display: flex; gap: var(--space-md); align-items: center; margin-top: var(--space-xs);">
			<a href="/exams" class="button" style="min-width: 140px;">Back to Courses</a>
			<button onclick={deleteExam} class="danger" style="min-width: 140px;">Delete Course</button>
		</div>
	</div>

	<hr />

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
						<td>{lecture.title}</td>
						<td>{lecture.specified_date ? new Date(lecture.specified_date).toLocaleDateString() : '-'}</td>
						<td>
							<span class="badge">{lecture.status}</span>
						</td>
						<td>
							<a href="/exams/{id}/lectures/{lecture.id}">Details</a>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	{/if}

	<div style="margin-top: var(--space-xl);">
		<h2>Study Tools & Chat</h2>
		<div style="display: flex; gap: var(--space-md); margin-top: var(--space-md);">
			<a href="/exams/{id}/chat" class="button" style="flex: 1;">
				Chat Assistant
			</a>
			<a href="/exams/{id}/tools" class="button" style="flex: 1;">
				Study Guides & Quizzes
			</a>
		</div>
	</div>
{/if}
