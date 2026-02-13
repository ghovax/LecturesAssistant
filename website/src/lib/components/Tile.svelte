<script lang="ts">
    import type { Snippet } from 'svelte';
    import Highlighter from './Highlighter.svelte';

    interface Props {
        href?: string;
        icon: string;
        title: string;
        monospaceTitle?: boolean;
        description?: Snippet;
        children?: Snippet;
        actions?: Snippet;
        onclick?: (e: MouseEvent) => void;
        class?: string;
        disabled?: boolean;
    }

    let { 
        href, 
        icon, 
        title, 
        monospaceTitle = false, 
        description, 
        children, 
        actions,
        onclick, 
        class: className = '', 
        disabled = false 
    }: Props = $props();

    let isHovered = $state(false);
</script>

<div 
    class="tile-wrapper {className}"
    onmouseenter={() => isHovered = true}
    onmouseleave={() => isHovered = false}
>
    {#if href}
        <a {href} {onclick}>
            <p class="tile-title" class:font-monospace={monospaceTitle}>
                <Highlighter show={isHovered} padding={0} offsetY={2}>
                    <strong>{title}</strong>
                </Highlighter>
            </p>
            
            {#if description}
                <div class="tileContent">
                    {@render description()}
                </div>
            {/if}

            {#if children}
                <div class="tile-extra-children">
                    {@render children()}
                </div>
            {/if}
        </a>
    {:else}
        <button type="button" class="tile-button" {onclick} {disabled}>
            <p class="tile-title" class:font-monospace={monospaceTitle}>
                <Highlighter show={isHovered} padding={0} offsetY={2}>
                    <strong>{title}</strong>
                </Highlighter>
            </p>
            
            {#if description}
                <div class="tileContent">
                    {@render description()}
                </div>
            {/if}

            {#if children}
                <div class="tile-extra-children">
                    {@render children()}
                </div>
            {/if}
        </button>
    {/if}

    {#if actions}
        <div class="tile-actions">
            {@render actions()}
        </div>
    {/if}
</div>
<style lang="scss">
    .tile-wrapper {
        display: inline-block;
        margin: 0;
        position: relative;
        vertical-align: top;
        background: #fff;
        border: 1px solid var(--gray-300);
        transition: all 0.2s ease;
        
        &:hover {
            border-color: var(--orange);
            background: #fafaf9;
        }
    }
    /* Common styles for both a and button */
    a, button.tile-button {
        appearance: none;
        border: none;
        background: transparent;
        color: var(--gray-900);
        display: flex;
        flex-direction: column;
        height: 140px;
        width: 240px;
        overflow: hidden;
        padding: 24px;
        text-decoration: none;
        text-align: left;
        position: relative;
        z-index: 1;
        font-family: 'Manrope', sans-serif;

        &:focus-visible {
            outline: 2px solid var(--orange);
            outline-offset: -2px;
        }
        &:disabled {
            opacity: 0.6;
            cursor: not-allowed;
        }
    }
    .tile-title {
        font-size: 15px;
        font-weight: 600;
        margin: 0 0 8px;
        line-height: 1.2;
        &.font-monospace {
            font-family: 'JetBrains Mono', monospace;
            font-size: 13px;
        }
    }
    .tileContent {
        font-size: 13px;
        color: var(--gray-500);
        line-height: 1.5;
        height: auto;
        overflow: hidden;
        display: -webkit-box;
        -webkit-line-clamp: 3;
        -webkit-box-orient: vertical;
    }
    .tile-actions {
        position: absolute;
        top: 12px;
        right: 12px;
        z-index: 10;
        display: flex;
        gap: 8px;
    }
    .tile-extra-children {
        margin-top: auto;
        position: relative;
        z-index: 2;
    }
</style>
