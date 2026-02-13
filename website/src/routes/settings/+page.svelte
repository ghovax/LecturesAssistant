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
        <div class="bg-white border mb-3">
            <div class="standard-header">
                <div class="header-title">
                    <span class="header-glyph" lang="ja">脳</span>
                    <span class="header-text">AI Assistant Settings</span>
                </div>
            </div>
            <div class="p-3">
                <div class="row g-3">
                    <div class="col-md-6">
                        <label for="provider" class="form-label fw-bold small text-muted text-uppercase mb-1" style="font-size: 0.65rem; letter-spacing: 0.05em;">AI Service Provider</label>
                        <select id="provider" class="form-select form-select-sm rounded-0 border shadow-none" bind:value={settings.llm.provider}>
                            <option value="openrouter">OpenRouter (Cloud)</option>
                            <option value="ollama">Ollama (Local)</option>
                        </select>
                    </div>

                    <div class="col-md-6">
                        <label for="language" class="form-label fw-bold small text-muted text-uppercase mb-1" style="font-size: 0.65rem; letter-spacing: 0.05em;">Preferred Language</label>
                        <select id="language" class="form-select form-select-sm rounded-0 border shadow-none" bind:value={settings.llm.language}>
                            <option value="en-US">{getLanguageName('en-US')}</option>
                            <option value="it-IT">{getLanguageName('it-IT')}</option>
                            <option value="ja-JP">{getLanguageName('ja-JP')}</option>
                            <option value="es-ES">{getLanguageName('es-ES')}</option>
                            <option value="fr-FR">{getLanguageName('fr-FR')}</option>
                            <option value="de-DE">{getLanguageName('de-DE')}</option>
                        </select>
                    </div>

                    <div class="col-12">
                        <label for="model" class="form-label fw-bold small text-muted text-uppercase mb-1" style="font-size: 0.65rem; letter-spacing: 0.05em;">Global Default Model</label>
                        <input type="text" id="model" class="form-control form-control-sm rounded-0 border shadow-none" bind:value={settings.llm.model} />
                        <div class="form-text mt-1" style="font-size: 0.75rem;">Used for all tasks unless overridden below. Example: <code>anthropic/claude-3.5-sonnet</code></div>
                    </div>
                </div>
            </div>
        </div>

        <div class="bg-white border mb-3">
            <div class="standard-header">
                <div class="header-title">
                    <span class="header-glyph" lang="ja">路</span>
                    <span class="header-text">Advanced Task Model Routing</span>
                </div>
            </div>
            <div class="p-3">
                <p class="text-muted small mb-3">Optional: Specify different models for specific processing steps. Leave empty to use the global default.</p>
                <div class="row g-3">
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
                            <label for="model-{task.key}" class="form-label fw-bold small text-muted text-uppercase mb-1" style="font-size: 0.6rem; letter-spacing: 0.05em;">{task.label}</label>
                            <input 
                                type="text" 
                                id="model-{task.key}" 
                                class="form-control form-control-sm rounded-0 border shadow-none" 
                                placeholder="Default"
                                bind:value={settings.llm.models[task.key].model} 
                            />
                        </div>
                    {/each}
                </div>
            </div>
        </div>

        <div class="bg-white border mb-3">
            <div class="standard-header">
                <div class="header-title">
                    <span class="header-glyph" lang="ja">鍵</span>
                    <span class="header-text">Service Credentials</span>
                </div>
            </div>
            <div class="p-3">
                <div class="mb-0">
                    <label for="openrouterApiKey" class="form-label fw-bold small text-muted text-uppercase mb-1" style="font-size: 0.65rem; letter-spacing: 0.05em;">OpenRouter API Key</label>
                    <input type="password" id="openrouterApiKey" class="form-control form-control-sm rounded-0 border shadow-none" bind:value={settings.providers.openrouter.api_key} />
                </div>
            </div>
        </div>

        <div class="bg-white border mb-3">
            <div class="standard-header">
                <div class="header-title">
                    <span class="header-glyph" lang="ja">守</span>
                    <span class="header-text">Safety & Budget</span>
                </div>
            </div>
            <div class="p-3">
                <div class="row g-3">
                    <div class="col-md-6">
                        <label for="maxCost" class="form-label fw-bold small text-muted text-uppercase mb-1" style="font-size: 0.65rem; letter-spacing: 0.05em;">Max Cost Per Job (USD)</label>
                        <input type="number" id="maxCost" step="0.01" class="form-control form-control-sm rounded-0 border shadow-none" bind:value={settings.safety.maximum_cost_per_job} />
                    </div>
                    <div class="col-md-6">
                        <label for="maxRetries" class="form-label fw-bold small text-muted text-uppercase mb-1" style="font-size: 0.65rem; letter-spacing: 0.05em;">Maximum Retries</label>
                        <input type="number" id="maxRetries" class="form-control form-control-sm rounded-0 border shadow-none" bind:value={settings.safety.maximum_retries} />
                    </div>
                </div>
            </div>
        </div>

        <div class="text-center mt-4">
            <button type="submit" class="btn btn-success px-5 rounded-0" disabled={saving}>
                {#if saving}
                    <span class="spinner-border spinner-border-sm me-2"></span>
                {/if}
                Save Preferences
            </button>
        </div>
    </form>
{/if}
