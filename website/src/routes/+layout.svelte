<script lang="ts">
    import { onMount } from 'svelte';
    import "../app.scss";
    import Navbar from '$lib/components/Navbar.svelte';
    import NotificationBanner from '$lib/components/NotificationBanner.svelte';
    
    let { children } = $props();

    onMount(async () => {
        // @ts-ignore
        await import('bootstrap');
    });
</script>

<div class="container-xl main-page">
    <NotificationBanner />
    <Navbar />
    <main class="content">
        {@render children()}
    </main>
</div>

<style>
    :global(html, body) {
        height: 100%;
        margin: 0;
    }
</style>