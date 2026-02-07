<script lang="ts">
	import '../app.css';
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import { auth } from '$lib/api';
	import { goto } from '$app/navigation';

	let { children } = $props();
	let user = $state(null);
	let checking = $state(true);

	async function checkAuth() {
		if (page.url.pathname === '/login') {
			checking = false;
			return;
		}
		try {
			const status = await auth.getStatus();
			if (!status.authenticated) {
				goto('/login');
			} else {
				user = status.user;
			}
		} catch (e) {
			goto('/login');
		} finally {
			checking = false;
		}
	}

	async function handleLogout() {
		await auth.logout();
		goto('/login');
	}

	onMount(checkAuth);
</script>

{#if checking}
	<p style="padding: 20px;">Checking authentication...</p>
{:else}
	<div class="container">
		{#if user || page.url.pathname === '/login'}
			{#if user}
				<aside>
					<div style="margin-bottom: var(--space-xl);">
						<a href="/" style="color: inherit; text-decoration: none; font-size: 1.1rem;"><strong>LECTURES ASSISTANT</strong></a>
					</div>

					<nav style="flex: 1;">
						<a href="/" class="nav-item" class:active={page.url.pathname === '/'}>Overview</a>
						<a href="/exams" class="nav-item" class:active={page.url.pathname.startsWith('/exams')}>Courses</a>
						<a href="/jobs" class="nav-item" class:active={page.url.pathname.startsWith('/jobs')}>Activity</a>
						<a href="/settings" class="nav-item" class:active={page.url.pathname === '/settings'}>Settings</a>
					</nav>

					<div style="margin-top: auto; padding-top: var(--space-md); border-top: 1px solid var(--border-color);">
						<div style="font-weight: 600; color: #666; font-size: 13px; margin-bottom: var(--space-sm);">{user.username}</div>
						<button onclick={handleLogout} style="width: 100%; min-width: auto;">Logout</button>
					</div>
				</aside>
			{/if}

			<main>
				{@render children()}
			</main>
		{/if}
	</div>
{/if}
