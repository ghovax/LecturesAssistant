<script lang="ts">
    import { api } from '$lib/api/client';
    import { notifications } from '$lib/stores/notifications.svelte';
    import { goto } from '$app/navigation';
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';

    let username = $state('');
    let password = $state('');
    let confirmPassword = $state('');
    let error = $state('');
    let loading = $state(false);

    async function handleRegister() {
        if (password !== confirmPassword) {
            error = 'Passwords do not match';
            return;
        }

        loading = true;
        error = '';
        try {
            await api.request('POST', '/auth/register', { username, password });
            notifications.success('Your account has been created. You can now sign in.');
            goto('/login');
        } catch (e: any) {
            error = e.message;
        } finally {
            loading = false;
        }
    }
</script>

<Breadcrumb items={[{ label: 'Sign Up', active: true }]} />

<div class="row justify-content-center">
    <div class="col-md-6">
        <h1>Sign Up</h1>
        
        <div class="well bg-white shadow-sm border p-4">
            <form onsubmit={(e) => { e.preventDefault(); handleRegister(); }}>
                {#if error}
                    <div class="alert alert-danger mb-4">{error}</div>
                {/if}
                
                <div class="mb-3">
                    <label for="username" class="form-label fw-bold small text-muted">Desired Username</label>
                    <input type="text" id="username" class="form-control" bind:value={username} required />
                </div>

                <div class="mb-3">
                    <label for="password" class="form-label fw-bold small text-muted">Password (minimum 8 letters)</label>
                    <input type="password" id="password" class="form-control" bind:value={password} required minlength="8" />
                </div>

                <div class="mb-4">
                    <label for="confirmPassword" class="form-label fw-bold small text-muted">Confirm Password</label>
                    <input type="password" id="confirmPassword" class="form-control" bind:value={confirmPassword} required />
                </div>

                <div class="text-center">
                    <button type="submit" class="btn btn-success px-4" disabled={loading}>
                        {#if loading}
                            <div class="spinner-border spinner-border-sm me-2" role="status">
                                <span class="visually-hidden">Loading...</span>
                            </div>
                        {/if}
                        Create Account
                    </button>
                </div>
            </form>
        </div>
        
        <div class="mt-4 text-center">
            <p class="text-muted small">Already have an account? <a href="/login">Log in here</a>.</p>
        </div>
    </div>
</div>
