<script lang="ts">
	import { onMount } from 'svelte';

	let settings = $state(null);
	let loading = $state(true);
	let error = $state(null);
	let saving = $state(false);

	async function fetchSettings() {
		try {
			const res = await fetch('/api/settings', {
				headers: {
					'X-Requested-With': 'XMLHttpRequest'
				}
			});
			if (!res.ok) throw new Error('Failed to fetch settings');
			const json = await res.json();
			settings = json.data;
		} catch (e) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	async function saveSettings() {
		saving = true;
		try {
			const res = await fetch('/api/settings', {
				method: 'PATCH',
				headers: {
					'Content-Type': 'application/json',
					'X-Requested-With': 'XMLHttpRequest'
				},
				body: JSON.stringify(settings)
			});
			if (!res.ok) throw new Error('Failed to save settings');
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
	<div class="card">
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

	<div class="card">
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

	<button onclick={saveSettings} disabled={saving}>
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
