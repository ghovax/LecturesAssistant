<script lang="ts">
    import { api } from '$lib/api/client';
    import { auth } from '$lib/auth.svelte';
    import { goto } from '$app/navigation';

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

<div class="row justify-content-center">
    <div class="col-md-6">
        <h1>Log In</h1>
        <div class="well">
            <form onsubmit={(e) => { e.preventDefault(); handleLogin(); }}>
                {#if error}
                    <div class="alert alert-danger">{error}</div>
                {/if}
                
                <div class="form-group mb-3">
                    <label for="username">Username</label>
                    <input type="text" id="username" class="form-control" bind:value={username} required />
                </div>

                <div class="form-group mb-4">
                    <label for="password">Password</label>
                    <input type="password" id="password" class="form-control" bind:value={password} required />
                </div>

                <div class="text-center">
                    <button type="submit" class="btn btn-primary btn-lg" disabled={loading}>
                        {#if loading}
                            <span class="spinner-border spinner-border-sm"></span>
                        {/if}
                        Log In
                    </button>
                </div>
            </form>
        </div>
    </div>
</div>
