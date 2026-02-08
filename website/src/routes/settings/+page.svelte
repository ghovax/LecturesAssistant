<script lang="ts">
	import { onMount } from 'svelte';
	import { apiFetch } from '$lib/api';

	let settings = $state(null);
	let loading = $state(true);
	let error = $state(null);
	let saving = $state(false);

	async function fetchSettings() {
		try {
			settings = await apiFetch('/api/settings');
		} catch (e) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	async function saveSettings() {
		saving = true;
		try {
			await apiFetch('/api/settings', {
				method: 'PATCH',
				body: settings
			});
			alert('Settings saved successfully');
		} catch (e) {
			alert('Error saving settings: ' + e.message);
		} finally {
			saving = false;
		}
	}

	onMount(fetchSettings);
</script>

<h1>Settings</h1>

{#if loading}
	<p>Loading settings...</p>
{:else if error}
	<div class="error">{error}</div>
{:else if settings}
	<div style="margin-bottom: var(--space-xl);">
		<h3>LLM Configuration</h3>
		<label>
			Provider
			<input type="text" bind:value={settings.llm.provider} />
		</label>
		<label>
			Default Model
			<input type="text" bind:value={settings.llm.model} />
		</label>
		<label>
			Language
			<input type="text" bind:value={settings.llm.language} />
		</label>
	</div>

	<div style="margin-bottom: var(--space-xl);">
		<h3>Transcription</h3>
		<label>
			Provider
			<input type="text" bind:value={settings.transcription.provider} />
		</label>
		<label>
			Model
			<input type="text" bind:value={settings.transcription.model} />
		</label>
	</div>

	<button onclick={saveSettings} disabled={saving} style="min-width: 160px;">
		{saving ? 'Saving...' : 'Save Settings'}
	</button>
{/if}

<style>
	label {
		display: block;
		margin-bottom: 16px;
	}
	input {
		margin-top: 4px;
	}
</style>
