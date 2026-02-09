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

<div class="row justify-content-center">
    <div class="col-md-6">
        <h1>System Setup</h1>
        <div class="well">
            <p class="mb-4">Create the initial administrator account and provide your OpenRouter API key to begin.</p>
            <form onsubmit={(e) => { e.preventDefault(); handleSetup(); }}>
                {#if error}
                    <div class="alert alert-danger">{error}</div>
                {/if}
                
                <div class="form-group mb-3">
                    <label for="username">Admin Username</label>
                    <input type="text" id="username" class="form-control" bind:value={username} required />
                </div>

                <div class="form-group mb-3">
                    <label for="password">Admin Password (min 8 chars)</label>
                    <input type="password" id="password" class="form-control" bind:value={password} required minlength="8" />
                </div>

                <div class="form-group mb-4">
                    <label for="apiKey">OpenRouter API Key</label>
                    <input type="password" id="apiKey" class="form-control" bind:value={apiKey} required />
                    <small class="form-text text-muted">This key is required for AI processing features.</small>
                </div>

                <div class="text-center">
                    <button type="submit" class="btn btn-success btn-lg" disabled={loading}>
                        {#if loading}
                            <span class="spinner-border spinner-border-sm"></span>
                        {/if}
                        Create Admin Account
                    </button>
                </div>
            </form>
        </div>
    </div>
</div>
