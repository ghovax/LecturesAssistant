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

<style lang="scss">
    .tile-button {
        appearance: none;
        border: none;
        background: #fff;
        box-shadow: .3125rem .3125rem .625rem rgba(0, 0, 0, .5);
        color: #000;
        display: inline-block;
        height: 7.5rem; /* 120px */
        margin: .625rem;
        overflow: hidden;
        padding: .9375rem;
        position: relative;
        text-decoration: none;
        vertical-align: top;
        text-align: left;
        width: 15.5rem;
        transition: transform 0.1s ease, background-color 0.1s ease, box-shadow 0.1s ease;

        &:hover:not(:disabled) {
            background: #c2c4a8;
        }

        &:active:not(:disabled) {
            background: #c2c4a8;
            box-shadow: .1875rem .1875rem .5rem rgba(0, 0, 0, .5);
            transform: scale(.95);
        }

        &:disabled {
            opacity: 0.6;
            cursor: not-allowed;
        }
    }
</style>
