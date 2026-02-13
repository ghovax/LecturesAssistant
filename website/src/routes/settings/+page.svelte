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
            
            // Ensure nested models structure exists for binding
            if (!settings.llm.models) settings.llm.models = {};
            const tasks = [
                'recording_transcription', 'documents_ingestion', 'documents_matching',
                'outline_creation', 'content_generation', 'content_verification', 'content_polishing'
            ];
            for (const t of tasks) {
                if (!settings.llm.models[t]) settings.llm.models[t] = { model: '' };
                else if (settings.llm.models[t].model === undefined) settings.llm.models[t].model = '';
            }
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

{#if loading}
    <div class="text-center p-5">
        <div class="village-spinner mx-auto" role="status"></div>
    </div>
{:else if settings}
    <form onsubmit={(e) => { e.preventDefault(); saveSettings(); }}>
        <div class="bg-white border mb-4">
            <div class="standard-header">
                <div class="header-title">
                    <span class="header-text">AI Assistant Settings</span>
                </div>
            </div>
            <div class="p-4">
                <div class="row g-4">
                    <div class="col-md-6">
                        <label for="provider" class="cozy-label">AI Service Provider</label>
                        <select id="provider" class="form-select cozy-input" bind:value={settings.llm.provider}>
                            <option value="openrouter">OpenRouter (Cloud)</option>
                            <option value="ollama">Ollama (Local)</option>
                        </select>
                    </div>

                    <div class="col-md-6">
                        <label for="language" class="cozy-label">Preferred Language</label>
                        <select id="language" class="form-select cozy-input" bind:value={settings.llm.language}>
                            <option value="en-US">{getLanguageName('en-US')}</option>
                            <option value="it-IT">{getLanguageName('it-IT')}</option>
                            <option value="ja-JP">{getLanguageName('ja-JP')}</option>
                            <option value="es-ES">{getLanguageName('es-ES')}</option>
                            <option value="fr-FR">{getLanguageName('fr-FR')}</option>
                            <option value="de-DE">{getLanguageName('de-DE')}</option>
                        </select>
                    </div>

                    <div class="col-12">
                        <label for="model" class="cozy-label">Global Default Model</label>
                        <input type="text" id="model" class="form-control cozy-input" bind:value={settings.llm.model} />
                        <div class="form-text mt-2" style="font-size: 12px;">Used for all tasks unless overridden below. Example: <code>anthropic/claude-3.5-sonnet</code></div>
                    </div>
                </div>
            </div>
        </div>

        <div class="bg-white border mb-4">
            <div class="standard-header">
                <div class="header-title">
                    <span class="header-text">Advanced Task Model Routing</span>
                </div>
            </div>
            <div class="p-4">
                <p class="text-muted small mb-4">Optional: Specify different models for specific processing steps. Leave empty to use the global default.</p>
                <div class="row g-4">
                    {#each [
                        { key: 'recording_transcription', label: 'Transcription Cleanup' },
                        { key: 'documents_ingestion', label: 'Document Analysis (OCR)' },
                        { key: 'documents_matching', label: 'Reference Triangulation' },
                        { key: 'outline_creation', label: 'Study Guide Outlining' },
                        { key: 'content_generation', label: 'Study Guide Writing' },
                        { key: 'content_verification', label: 'Accuracy Verification' },
                        { key: 'content_polishing', label: 'Footnote Polishing' }
                    ] as task}
                        <div class="col-md-6">
                            <label for="model-{task.key}" class="cozy-label">{task.label}</label>
                            <input 
                                type="text" 
                                id="model-{task.key}" 
                                class="form-control cozy-input" 
                                placeholder="Default"
                                bind:value={settings.llm.models[task.key].model} 
                            />
                        </div>
                    {/each}
                </div>
            </div>
        </div>

        <div class="bg-white border mb-4">
            <div class="standard-header">
                <div class="header-title">
                    <span class="header-text">Service Credentials</span>
                </div>
            </div>
            <div class="p-4">
                <div class="mb-0">
                    <label for="openrouterApiKey" class="cozy-label">OpenRouter API Key</label>
                    <input type="password" id="openrouterApiKey" class="form-control cozy-input" bind:value={settings.providers.openrouter.api_key} />
                </div>
            </div>
        </div>

        <div class="bg-white border mb-4">
            <div class="standard-header">
                <div class="header-title">
                    <span class="header-text">Safety & Budget</span>
                </div>
            </div>
            <div class="p-4">
                <div class="row g-4">
                    <div class="col-md-6">
                        <label for="maxCost" class="cozy-label">Max Cost Per Job (USD)</label>
                        <input type="number" id="maxCost" step="0.01" class="form-control cozy-input" bind:value={settings.safety.maximum_cost_per_job} />
                    </div>
                    <div class="col-md-6">
                        <label for="maxRetries" class="cozy-label">Maximum Retries</label>
                        <input type="number" id="maxRetries" class="form-control cozy-input" bind:value={settings.safety.maximum_retries} />
                    </div>
                </div>
            </div>
        </div>

        <div class="text-center mt-5 pb-5">
            <button type="submit" class="btn btn-success px-5 rounded-0 btn-lg" disabled={saving}>
                {#if saving}
                    <span class="spinner-border spinner-border-sm me-2"></span>
                {/if}
                Save Preferences
            </button>
        </div>
    </form>
{/if}

<style lang="scss">
</style>
