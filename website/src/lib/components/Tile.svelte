<script lang="ts">
    import type { Snippet } from 'svelte';

    interface Props {
        href?: string;
        icon: string;
        title: string;
        monospaceTitle?: boolean;
        description?: Snippet;
        children?: Snippet;
        onclick?: (e: MouseEvent) => void;
        class?: string;
        disabled?: boolean;
    }

    let { href, icon, title, monospaceTitle = false, description, children, onclick, class: className = '', disabled = false }: Props = $props();
</script>

{#if href}
    <a {href} {onclick} class={className}>
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
{:else}
    <button type="button" {onclick} class="tile-button {className}" {disabled}>
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
    </button>
{/if}
