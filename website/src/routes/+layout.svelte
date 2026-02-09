<script lang="ts">
    import { onMount } from 'svelte';
    import "../app.scss";
    import Navbar from '$lib/components/Navbar.svelte';
    
    let { children } = $props();
</script>

<div class="container-xl main-page">
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