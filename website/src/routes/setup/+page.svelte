<script lang="ts">
    import { api } from '$lib/api/client';
    import { goto } from '$app/navigation';
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';

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
            goto('/');
        } catch (e: any) {
            error = e.message;
        } finally {
            loading = false;
        }
    }
</script>

<Breadcrumb items={[{ label: 'Get Started', active: true }]} />

<div class="row justify-content-center">
    <div class="col-md-8">
        <h1>Let's Get Started</h1>
        
        <div class="well bg-white shadow-sm border p-4">
            <p class="mb-4">Welcome! Create your account and provide your API credentials to begin your learning journey.</p>
            
            <form onsubmit={(e) => { e.preventDefault(); handleSetup(); }}>
                {#if error}
                    <div class="alert alert-danger mb-4">{error}</div>
                {/if}
                
                <div class="row">
                    <div class="col-md-6 mb-3">
                        <label for="username" class="form-label fw-bold small text-muted">Admin Username</label>
                        <input type="text" id="username" class="form-control" bind:value={username} required />
                    </div>

                    <div class="col-md-6 mb-3">
                        <label for="password" class="form-label fw-bold small text-muted">Admin Password (minimum 8 letters)</label>
                        <input type="password" id="password" class="form-control" bind:value={password} required minlength="8" />
                    </div>
                </div>

                <div class="mb-4">
                    <label for="apiKey" class="form-label fw-bold small text-muted">OpenRouter API Key</label>
                    <input type="password" id="apiKey" class="form-control" bind:value={apiKey} required />
                    <small class="form-text text-muted mt-2 d-block">This key is required for all AI-powered transcription and generation features.</small>
                </div>

                <div class="text-center">
                    <button type="submit" class="btn btn-success px-5" disabled={loading}>
                        {#if loading}
                            <span class="village-spinner d-inline-block me-2" style="width: 1rem; height: 1rem;"></span>
                        {/if}
                        Create My Account
                    </button>
                </div>
            </form>
        </div>
    </div>
</div>
