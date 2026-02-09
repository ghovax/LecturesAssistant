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

<div class="container-fluid p-0">
    <div class="row justify-content-center">
        <div class="col-xl-6 col-lg-8 col-md-10">
            <h1 class="characterHeading text-center mb-4">Log In</h1>
            
            <div class="well bg-white p-5 shadow-sm border">
                <form onsubmit={(e) => { e.preventDefault(); handleLogin(); }}>
                    {#if error}
                        <div class="alert alert-danger border-0 mb-4">{error}</div>
                    {/if}
                    
                    <div class="mb-4">
                        <label for="username" class="form-label fw-bold small text-uppercase text-muted">Username</label>
                        <input type="text" id="username" class="form-control form-control-lg bg-light border-0" bind:value={username} required />
                    </div>

                    <div class="mb-5">
                        <label for="password" class="form-label fw-bold small text-uppercase text-muted">Password</label>
                        <input type="password" id="password" class="form-control form-control-lg bg-light border-0" bind:value={password} required />
                    </div>

                    <div class="text-center">
                        <button type="submit" class="btn btn-primary btn-lg px-5" disabled={loading}>
                            {#if loading}
                                <span class="village-spinner d-inline-block me-2" style="width: 1.2rem; height: 1.2rem;"></span>
                            {/if}
                            Sign In
                        </button>
                    </div>
                </form>
            </div>
            
            <div class="mt-4 text-center">
                <p class="text-muted">Access your personalized learning materials and study tools.</p>
            </div>
        </div>
    </div>
</div>
