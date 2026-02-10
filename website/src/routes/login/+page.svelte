<script lang="ts">
    import { api } from '$lib/api/client';
    import { auth } from '$lib/auth.svelte';
    import { goto } from '$app/navigation';
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';

    let username = $state('');
    let password = $state('');
    let error = $state('');
    let loading = $state(false);

    async function handleLogin() {
        loading = true;
        error = '';
        try {
            await api.login({ username, password });
            await auth.check();
            goto('/');
        } catch (e: any) {
            error = e.message;
        } finally {
            loading = false;
        }
    }
</script>

<Breadcrumb items={[{ label: 'Login', active: true }]} />

<div class="row justify-content-center">
    <div class="col-md-6">
        <h1>Login</h1>
        
        <div class="well bg-white shadow-sm border p-4">
            <form onsubmit={(e) => { e.preventDefault(); handleLogin(); }}>
                {#if error}
                    <div class="alert alert-danger mb-4">{error}</div>
                {/if}
                
                <div class="mb-3">
                    <label for="username" class="form-label fw-bold small text-muted">Username</label>
                    <input type="text" id="username" class="form-control" bind:value={username} required />
                </div>

                <div class="mb-4">
                    <label for="password" class="form-label fw-bold small text-muted">Password</label>
                    <input type="password" id="password" class="form-control" bind:value={password} required />
                </div>

                <div class="text-center">
                    <button type="submit" class="btn btn-primary px-4" disabled={loading}>
                        {#if loading}
                            <span class="village-spinner d-inline-block me-2" style="width: 1rem; height: 1rem;"></span>
                        {/if}
                        Sign In
                    </button>
                </div>
            </form>
        </div>
        
        <div class="mt-4 text-center">
            <p class="text-muted small">Don't have an account? <a href="/register">Sign up here</a>.</p>
            <p class="text-muted small">Access your personalized learning materials and study aids.</p>
        </div>
    </div>
</div>
