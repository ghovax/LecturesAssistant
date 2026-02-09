<script lang="ts">
    import { onMount } from 'svelte';
    import { api } from '$lib/api/client';
    import { notifications } from '$lib/stores/notifications.svelte';
    import { getLanguageName } from '$lib/utils';
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';

    let settings = $state<any>(null);
    let loading = $state(true);
    let saving = $state(false);

    async function loadSettings() {
        loading = true;
        try {
            settings = await api.getSettings();
        } catch (e) {
            console.error(e);
        } finally {
            loading = false;
        }
    }

    async function saveSettings() {
        saving = true;
        try {
            await api.updateSettings({
                llm: settings.llm,
                safety: settings.safety,
                providers: settings.providers
            });
            notifications.success('Your preferences have been saved.');
        } catch (e: any) {
            notifications.error(e.message || e);
        } finally {
            saving = false;
        }
    }

    onMount(loadSettings);
</script>

<Breadcrumb items={[{ label: 'Preferences', active: true }]} />

<h1>Preferences</h1>

{#if loading}
    <div class="text-center p-5">
        <div class="village-spinner mx-auto"></div>
    </div>
{:else if settings}
    <form onsubmit={(e) => { e.preventDefault(); saveSettings(); }} class="well">
        <h4>LLM Configuration</h4>
        <div class="mb-4">
            <label for="provider" class="form-label">Provider</label>
            <select id="provider" class="form-select" bind:value={settings.llm.provider}>
                <option value="openrouter">OpenRouter (Cloud)</option>
                <option value="ollama">Ollama (Local)</option>
            </select>
        </div>

        <div class="mb-4">
            <label for="language" class="form-label">Study Language</label>
            <select id="language" class="form-select" bind:value={settings.llm.language}>
                <option value="en-US">{getLanguageName('en-US')}</option>
                <option value="it-IT">{getLanguageName('it-IT')}</option>
                <option value="ja-JP">{getLanguageName('ja-JP')}</option>
                <option value="es-ES">{getLanguageName('es-ES')}</option>
                <option value="fr-FR">{getLanguageName('fr-FR')}</option>
                <option value="de-DE">{getLanguageName('de-DE')}</option>
            </select>
            <small class="text-muted d-block mt-1">Primary language for study kits and assistant responses.</small>
        </div>

        <div class="mb-4">
            <label for="model" class="form-label">Primary Model</label>
            <input type="text" id="model" class="form-control" bind:value={settings.llm.model} />
            <small class="text-muted">Default model used for generation tasks.</small>
        </div>

        <h4>Provider Credentials</h4>
        <div class="mb-4">
            <label for="openrouterApiKey" class="form-label">OpenRouter API Key</label>
            <input type="password" id="openrouterApiKey" class="form-control" bind:value={settings.providers.openrouter.api_key} />
        </div>

        <h4>Safety & Budget</h4>
        <div class="row mb-4">
            <div class="col-md-6">
                <label for="maxCost" class="form-label">Maximum Cost Per Job (USD)</label>
                <input type="number" id="maxCost" step="0.01" class="form-control" bind:value={settings.safety.maximum_cost_per_job} />
            </div>
            <div class="col-md-6">
                <label for="maxRetries" class="form-label">Maximum Retries</label>
                <input type="number" id="maxRetries" class="form-control" bind:value={settings.safety.maximum_retries} />
            </div>
        </div>

        <div class="text-center mt-4">
            <button type="submit" class="btn btn-primary btn-lg" disabled={saving}>
                {#if saving}
                    <span class="spinner-border spinner-border-sm me-2"></span>
                {/if}
                Save Settings
            </button>
        </div>
    </form>
{/if}
