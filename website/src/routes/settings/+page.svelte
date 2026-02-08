<script lang="ts">
	import { onMount } from 'svelte';
	import { apiFetch } from '$lib/api';
	import { notifications } from '$lib/notifications';

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
			notifications.success('Settings saved successfully');
		} catch (e) {
			notifications.error('Error saving settings: ' + e.message);
		} finally {
			saving = false;
		}
	}

	// Helper to handle the "model or object" structure from the backend
	function getModelValue(modelConfig: any) {
		if (typeof modelConfig === 'string') return modelConfig;
		return modelConfig?.model || '';
	}

	function setModelValue(path: string, value: string) {
		const parts = path.split('.');
		let current = settings;
		for (let i = 0; i < parts.length - 1; i++) {
			current = current[parts[i]];
		}
		const lastPart = parts[parts.length - 1];
		
		// If it doesn't exist, create it as a string by default
		if (current[lastPart] === undefined) {
			current[lastPart] = value;
			return;
		}

		if (typeof current[lastPart] === 'string') {
			current[lastPart] = value;
		} else {
			current[lastPart].model = value;
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
	<div style="display: flex; flex-direction: column; gap: var(--space-xl); max-width: 800px; padding-bottom: var(--space-xl);">
		<section>
			<h2>General Settings</h2>
			<div style="display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-md); margin-top: var(--space-md);">
				<label>
					Main provider
					<select bind:value={settings.llm.provider}>
						<option value="openrouter">OpenRouter</option>
						<option value="ollama">Ollama (Local)</option>
					</select>
				</label>
				<label>
					Default model
					<input type="text" bind:value={settings.llm.model} placeholder="e.g. anthropic/claude-3.5-sonnet" />
				</label>
				<label>
					Preferred language
					<input type="text" bind:value={settings.llm.language} placeholder="e.g. en-US" />
				</label>
				<label style="display: flex; align-items: center; gap: var(--space-sm); padding-top: 24px; cursor: pointer;">
					<input type="checkbox" bind:checked={settings.llm.enable_documents_matching} style="width: auto; height: auto;" />
					Automatically link documents to transcripts
				</label>
			</div>
		</section>

		<section>
			<h2>Transcription</h2>
			<div style="display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-md); margin-top: var(--space-md);">
				<label>
					Provider
					<select bind:value={settings.transcription.provider}>
						<option value="openrouter">OpenRouter</option>
						<option value="ollama">Ollama (Local)</option>
					</select>
				</label>
				<label>
					Model override
					<input type="text" bind:value={settings.transcription.model} placeholder="Defaults to recording model" />
				</label>
				<label>
					Audio chunk size (seconds)
					<input type="number" bind:value={settings.transcription.audio_chunk_length_seconds} />
				</label>
				<label>
					Refining batch size
					<input type="number" bind:value={settings.transcription.refining_batch_size} />
				</label>
			</div>
		</section>

		<section>
			<h2>Document Processing</h2>
			<div style="display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-md); margin-top: var(--space-md);">
				<label>
					Image resolution (DPI)
					<input type="number" bind:value={settings.documents.render_dots_per_inch} />
				</label>
				<label>
					Page limit per file
					<input type="number" bind:value={settings.documents.maximum_pages} />
				</label>
			</div>
		</section>

		<section>
			<h2>Safety and Limits</h2>
			<div style="display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-md); margin-top: var(--space-md);">
				<label>
					Maximum cost per job ($)
					<input type="number" step="0.5" bind:value={settings.safety.maximum_cost_per_job} />
				</label>
				<label>
					Maximum attempts
					<input type="number" bind:value={settings.safety.maximum_retries} />
				</label>
			</div>
		</section>

		<section>
			<h2>API Credentials</h2>
			<div style="display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-md); margin-top: var(--space-md);">
				<label>
					OpenRouter API
					<input type="password" bind:value={settings.providers.openrouter.api_key} placeholder="sk-or-v1-..." />
				</label>
				<label>
					Ollama URL
					<input type="text" bind:value={settings.providers.ollama.base_url} placeholder="http://localhost:11434" />
				</label>
			</div>
		</section>

		<section>
			<h2>Task-Specific Models</h2>
			<p style="font-size: 13px; color: #666; margin-bottom: var(--space-md);">You can override the primary model for specific AI tasks if needed.</p>
			
			<div style="display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-md);">
				<label>
					Transcribe audio and video
					<input type="text" 
						value={getModelValue(settings.llm.models.recording_transcription)} 
						oninput={(e) => setModelValue('llm.models.recording_transcription', e.currentTarget.value)}
						placeholder={settings.resolved_models.recording_transcription}
					/>
				</label>
				<label>
					Read and process documents
					<input type="text" 
						value={getModelValue(settings.llm.models.documents_ingestion)} 
						oninput={(e) => setModelValue('llm.models.documents_ingestion', e.currentTarget.value)}
						placeholder={settings.resolved_models.documents_ingestion}
					/>
				</label>
				<label>
					Link documents to transcript
					<input type="text" 
						value={getModelValue(settings.llm.models.documents_matching)} 
						oninput={(e) => setModelValue('llm.models.documents_matching', e.currentTarget.value)}
						placeholder={settings.resolved_models.documents_matching}
					/>
				</label>
				<label>
					Analyze lecture structure
					<input type="text" 
						value={getModelValue(settings.llm.models.outline_creation)} 
						oninput={(e) => setModelValue('llm.models.outline_creation', e.currentTarget.value)}
						placeholder={settings.resolved_models.outline_creation}
					/>
				</label>
				<label>
					Generate study material
					<input type="text" 
						value={getModelValue(settings.llm.models.content_generation)} 
						oninput={(e) => setModelValue('llm.models.content_generation', e.currentTarget.value)}
						placeholder={settings.resolved_models.content_generation}
					/>
				</label>
				<label>
					Refine titles and formatting
					<input type="text" 
						value={getModelValue(settings.llm.models.content_polishing)} 
						oninput={(e) => setModelValue('llm.models.content_polishing', e.currentTarget.value)}
						placeholder={settings.resolved_models.content_polishing}
					/>
				</label>
			</div>
		</section>

		<div style="padding-top: var(--space-lg); border-top: 1px solid var(--border-color); margin-top: var(--space-md);">
			<button onclick={saveSettings} disabled={saving} style="min-width: 200px;">
				{saving ? 'Saving Changes...' : 'Save Settings'}
			</button>
		</div>
	</div>
{/if}

<style>
	h2 { margin-bottom: 0; }
	label { display: block; font-size: 13px; font-weight: 600; color: #444; }
	input, select { margin-top: var(--space-xs); font-weight: normal; }
</style>
