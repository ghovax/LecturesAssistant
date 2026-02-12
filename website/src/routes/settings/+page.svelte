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

<div class="bg-white border mb-5">
    <div class="standard-header">
        <div class="header-title">
            <span class="header-glyph" lang="ja">設</span>
            <span class="header-text">Preferences</span>
        </div>
    </div>

    {#if loading}
        <div class="text-center p-5">
            <div class="village-spinner mx-auto" role="status"></div>
        </div>
    {:else if settings}
        <div class="p-4">
            <form onsubmit={(e) => { e.preventDefault(); saveSettings(); }}>
                <h4 class="mt-0">AI Assistant Settings</h4>
                <div class="mb-4">
                    <label for="provider" class="form-label fw-bold">AI Service</label>
                    <select id="provider" class="form-select shadow-none" bind:value={settings.llm.provider}>
                        <option value="openrouter">OpenRouter (Cloud)</option>
                        <option value="ollama">Ollama (Local)</option>
                    </select>
                </div>

                <div class="mb-4">
                    <label for="language" class="form-label fw-bold">Preferred Language</label>
                    <select id="language" class="form-select shadow-none" bind:value={settings.llm.language}>
                        <option value="en-US">{getLanguageName('en-US')}</option>
                        <option value="it-IT">{getLanguageName('it-IT')}</option>
                        <option value="ja-JP">{getLanguageName('ja-JP')}</option>
                        <option value="es-ES">{getLanguageName('es-ES')}</option>
                        <option value="fr-FR">{getLanguageName('fr-FR')}</option>
                        <option value="de-DE">{getLanguageName('de-DE')}</option>
                    </select>
                    <small class="text-muted d-block mt-1">Used for study materials and assistant responses.</small>
                </div>

                <div class="mb-4">
                    <label for="model" class="form-label fw-bold">AI Model</label>
                    <input type="text" id="model" class="form-control shadow-none" bind:value={settings.llm.model} />
                    <small class="text-muted">The specific model used for analysis and generation.</small>
                </div>

                <h4>Service Credentials</h4>
                <div class="mb-4">
                    <label for="openrouterApiKey" class="form-label fw-bold">API Key</label>
                    <input type="password" id="openrouterApiKey" class="form-control shadow-none" bind:value={settings.providers.openrouter.api_key} />
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
                    <button type="submit" class="btn btn-primary btn-lg px-5" disabled={saving}>
                        {#if saving}
                            <span class="spinner-border spinner-border-sm me-2"></span>
                        {/if}
                        Save Settings
                    </button>
                </div>
            </form>
        </div>
    {/if}
</div>
