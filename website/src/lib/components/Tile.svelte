<script lang="ts">
    import type { Snippet } from 'svelte';

    interface Props {
        href: string;
        icon: string;
        title: string;
        children?: Snippet;
    }

    let { href, icon, title, children }: Props = $props();
</script>

<a {href}>
    <div lang="ja">{icon}</div>
    <p><strong>{title}</strong><br /></p>
    {#if children}
        <div class="tileContent">
            {@render children()}
        </div>
    {/if}
</a>
