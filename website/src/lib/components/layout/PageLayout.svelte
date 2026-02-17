<script lang="ts">
    import type { Snippet } from 'svelte';

    interface Props {
        children: Snippet;
        class?: string;
    }

    let { children, class: className = '' }: Props = $props();
</script>

<div class="page-layout {className}">
    {@render children()}
</div>

<style lang="scss">
    .page-layout {
        padding-bottom: 80px;
    }
</style>
