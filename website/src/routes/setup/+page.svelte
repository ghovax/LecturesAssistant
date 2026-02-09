<script lang="ts">
    import { api } from '$lib/api/client';
    import { goto } from '$app/navigation';

    let username = $state('');
    let password = $state('');
    let apiKey = $state('');
    let error = $state('');
    let loading = $state(false);

    async function handleSetup() {
        loading = true;
        error = '';
        try {
            await api.setup({ username, password, openrouter_api_key: apiKey });
            goto('/login');
        } catch (e: any) {
            error = e.message;
        } finally {
            loading = false;
        }
    }
</script>

<div class="container-fluid p-0">
    <div class="row justify-content-center">
        <div class="col-xl-8 col-lg-10">
            <h1 class="characterHeading text-center mb-4">System Setup</h1>
            
            <div class="well bg-white p-5 shadow-sm border">
                <p class="lead mb-5 text-center">Create the initial administrator account and provide your API credentials to begin.</p>
                
                <form onsubmit={(e) => { e.preventDefault(); handleSetup(); }}>
                    {#if error}
                        <div class="alert alert-danger border-0 mb-4">{error}</div>
                    {/if}
                    
                    <div class="row">
                        <div class="col-md-6 mb-4">
                            <label for="username" class="form-label fw-bold small text-uppercase text-muted">Admin Username</label>
                            <input type="text" id="username" class="form-control form-control-lg bg-light border-0" bind:value={username} required />
                        </div>

                        <div class="col-md-6 mb-4">
                            <label for="password" class="form-label fw-bold small text-uppercase text-muted">Admin Password (min 8 chars)</label>
                            <input type="password" id="password" class="form-control form-control-lg bg-light border-0" bind:value={password} required minlength="8" />
                        </div>
                    </div>

                    <div class="mb-5">
                        <label for="apiKey" class="form-label fw-bold small text-uppercase text-muted">OpenRouter API Key</label>
                        <input type="password" id="apiKey" class="form-control form-control-lg bg-light border-0" bind:value={apiKey} required />
                        <small class="form-text text-muted mt-2 d-block">This key is required for all AI-powered transcription and generation features.</small>
                    </div>

                    <div class="text-center">
                        <button type="submit" class="btn btn-success btn-lg px-5" disabled={loading}>
                            {#if loading}
                                <span class="village-spinner d-inline-block me-2" style="width: 1.2rem; height: 1.2rem;"></span>
                            {/if}
                            Initialize System
                        </button>
                    </div>
                </form>
            </div>
        </div>
    </div>
</div>
