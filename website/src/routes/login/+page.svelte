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
				goto('/');
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
			if (status.authenticated) goto('/');
			// If status check fails or returns configured: false, we might want to show setup
			// But the server handler for setup checks if users exist.
		} catch (e) {
			// Ignore
		}
	});
</script>

<div style="max-width: 400px; margin: 100px auto;">
	<h1>{isSetup ? 'Initial Setup' : 'Login'}</h1>

	{#if error}
		<div class="error">{error}</div>
	{/if}

	<div class="card">
		<form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }}>
			<label>
				Username
				<input type="text" bind:value={username} required />
			</label>
			<label>
				Password
				<input type="password" bind:value={password} required />
			</label>
			<button type="submit" disabled={loading}>
				{loading ? 'Processing...' : (isSetup ? 'Create Admin' : 'Login')}
			</button>
		</form>
	</div>

	<p style="text-align: center;">
		<button onclick={() => isSetup = !isSetup}>
			Switch to {isSetup ? 'Login' : 'Setup (Admin)'}
		</button>
	</p>
</div>

<style>
	label { display: block; margin-bottom: 16px; }
	input { margin-top: 4px; }
</style>
