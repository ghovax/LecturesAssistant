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
        cost?: number;
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
        disabled = false,
        cost
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

            {#if cost && cost > 0}
                <div class="tile-cost">
                    ${cost.toFixed(4)}
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

            {#if cost && cost > 0}
                <div class="tile-cost">
                    ${cost.toFixed(4)}
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
        position: relative;
        vertical-align: top;
        background: #fff;
        transition: all 0.2s ease;
        
        &:hover {
            background: #fafaf9;
            z-index: 10;
        }

        &.tile-processing {
            background: #fafaf9;
            opacity: 0.8;
            
            .tile-title {
                color: var(--gray-500);
            }
        }

        &.tile-error {
            background: #fffafa;
            
            .tile-title {
                color: #b91c1c;
            }
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
        height: 150px;
        width: 250px;
        padding: 20px;
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
        font-size: 0.9rem;
        font-weight: 600;
        margin: 0 0 8px;
        line-height: 1.2;
        &.font-monospace {
            font-family: 'JetBrains Mono', monospace;
            font-size: 0.85rem;
        }
    }
    .tileContent {
        font-size: 0.85rem;
        color: var(--gray-500);
        line-height: 1.5;
        height: auto;
        overflow: hidden;
        display: -webkit-box;
        -webkit-line-clamp: 3;
        -webkit-box-orient: vertical;
        margin-bottom: 15px;
    }
    .tile-actions {
        position: absolute;
        bottom: 12px;
        right: 20px;
        z-index: 10;
        display: flex;
        gap: 8px;
    }
    .tile-extra-children {
        margin-top: auto;
        margin-bottom: 20px;
        position: relative;
        z-index: 2;
    }
    .tile-cost {
        position: absolute;
        bottom: 12px;
        left: 20px;
        font-size: 0.7rem;
        color: var(--gray-400);
        font-family: 'JetBrains Mono', monospace;
        pointer-events: none;
    }
</style>
