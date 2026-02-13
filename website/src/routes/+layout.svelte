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

<div class="main-layout container-xl shadow-none">
    <Navbar />
    <NotificationBanner />
    <div class="content-wrapper">
        <main class="content">
            {@render children()}
        </main>
    </div>
</div>

<style lang="scss">
    :global(html, body) {
        height: 100%;
        margin: 0;
        background: #f5f3f0 !important;
        font-family: 'Manrope', sans-serif !important;
        color: #292524 !important;
    }

    .main-layout {
        background: transparent !important;
        box-shadow: none !important;
        border: none !important;
        max-width: 1000px !important; /* Increased width */
        margin: 0 auto;
        padding: 0 20px;
    }

    .content-wrapper {
        width: 100%;
    }

    .content {
        padding: 0;
    }
</style>