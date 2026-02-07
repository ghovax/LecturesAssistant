<script lang="ts">
	import { onMount } from 'svelte';
	import { auth } from '$lib/api';
	import { goto } from '$app/navigation';

	let username = $state('');
	let password = $state('');
	let isSetup = $state(false);
	let error = $state(null);
	let loading = $state(false);

	async function handleSubmit() {
		loading = true;
		error = null;
		try {
			if (isSetup) {
				await auth.setup({ username, password });
				isSetup = false;
				alert('Setup successful. Please login.');
			} else {
				await auth.login({ username, password });
				window.location.href = '/';
			}
		} catch (e) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	onMount(async () => {
		try {
			const status = await auth.getStatus();
			if (status.authenticated) {
				goto('/');
			} else {
				isSetup = !status.initialized;
			}
		} catch (e) {
			// Ignore errors, default to login
		}
	});
</script>

<div style="max-width: 400px; margin: 10vh auto; padding: 0 var(--space-md);">
	<h1 style="text-align: center; margin-bottom: var(--space-xl);">
		{isSetup ? 'Create Your Account' : 'Welcome Back'}
	</h1>

	{#if error}
		<div class="error">{error}</div>
	{/if}

	<div class="card" style="padding: var(--space-xl);">
		<form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }}>
			<label>
				Username
				<input type="text" bind:value={username} required placeholder="Enter your username" />
			</label>
			<label>
				Password
				<input type="password" bind:value={password} required placeholder="Enter your password" />
			</label>
			<button type="submit" disabled={loading} style="width: 100%; margin-top: var(--space-sm);">
				{loading ? 'Please wait...' : (isSetup ? 'Get Started' : 'Sign In')}
			</button>
		</form>
	</div>

	<p style="text-align: center; margin-top: var(--space-lg); font-size: 13px; color: #666;">
		{#if isSetup}
			Already have an account? <button onclick={() => isSetup = false} style="min-width: auto; height: auto; padding: 0; background: transparent; border: none; color: var(--accent-color); font-weight: normal; text-decoration: underline;">Sign in here</button>
		{:else}
			First time here? <button onclick={() => isSetup = true} style="min-width: auto; height: auto; padding: 0; background: transparent; border: none; color: var(--accent-color); font-weight: normal; text-decoration: underline;">Create your first account</button>
		{/if}
	</p>
</div>

<style>
	label { display: block; margin-bottom: 16px; }
	input { margin-top: 4px; }
</style>
