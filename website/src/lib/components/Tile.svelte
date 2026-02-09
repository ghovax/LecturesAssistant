<script lang="ts">
    import type { Snippet } from 'svelte';

    interface Props {
        href: string;
        icon: string;
        title: string;
        monospaceTitle?: boolean;
        description?: Snippet;
        children?: Snippet;
        onclick?: (e: MouseEvent) => void;
    }

    let { href, icon, title, monospaceTitle = false, description, children, onclick }: Props = $props();
</script>

<a {href} {onclick}>
    <div lang="ja">{icon}</div>
    <p class:font-monospace={monospaceTitle}><strong>{title}</strong><br /></p>
    
    {#if description}
        <div class="tileContent">
            {@render description()}
        </div>
    {/if}

    {#if children}
        {@render children()}
    {/if}
</a>