<script lang="ts">
    import { onMount } from 'svelte';
    import { goto } from '$app/navigation';
    import { auth } from '$lib/auth.svelte';

    let { children } = $props();
    let checking = $state(true);

    onMount(async () => {
        await auth.check();

        if (!auth.initialized) {
            goto('/setup');
            return;
        }

        if (!auth.user) {
            goto('/login');
            return;
        }

        checking = false;
    });
</script>

{#if checking}
    <div class="text-center p-5">
        <div class="village-spinner mx-auto"></div>
    </div>
{:else}
    {@render children()}
{/if}
