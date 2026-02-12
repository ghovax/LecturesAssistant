<script lang="ts">
    import type { Snippet } from 'svelte';

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
</script>

<div class="tile-wrapper {className}">
    {#if href}
        <a {href} {onclick}>
            <div lang="ja" class="tile-glyph">{icon}</div>
            <p class="tile-title" class:font-monospace={monospaceTitle}><strong>{title}</strong><br /></p>
            
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
            <div lang="ja" class="tile-glyph">{icon}</div>
            <p class="tile-title" class:font-monospace={monospaceTitle}><strong>{title}</strong><br /></p>
            
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
        margin: .625rem;
        position: relative;
        vertical-align: top;
        background: #fff;
        box-shadow: .3125rem .3125rem .625rem rgba(0, 0, 0, .5);
        transition: transform 0.1s ease, box-shadow 0.1s ease;

        &:active {
            box-shadow: .1875rem .1875rem .5rem rgba(0, 0, 0, .5);
            transform: scale(.95);
        }
    }

    /* Common styles for both a and button */
    a, button.tile-button {
        appearance: none;
        border: none;
        background: #fff;
        color: #000;
        display: block;
        height: 7.5rem; /* 120px */
        width: 15.5rem; /* Default Md size */
        overflow: hidden;
        padding: .9375rem;
        text-decoration: none;
        text-align: left;
        transition: background-color 0.1s ease;
        position: relative;
        z-index: 1;

        &:hover:not(:disabled) {
            background: #c2c4a8;
        }

        &:disabled {
            opacity: 0.6;
            cursor: not-allowed;
        }
    }

    .tile-glyph {
        color: #dfe1d0;
        font-family: "ＭＳ Ｐゴシック", "MS PGothic", "メイリオ", Meiryo, sans-serif;
        font-size: 15rem;
        left: 6.25rem;
        line-height: 7.5rem;
        position: absolute;
        top: 2.8125rem;
        z-index: 1;
        pointer-events: none;
    }

    .tile-title {
        bottom: .9375rem !important;
        font-size: 1.2rem;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        position: absolute;
        left: .9375rem;
        right: .9375rem;
        z-index: 2;
        margin: 0;
        line-height: 1.2;

        &.font-monospace {
            font-family: monospace;
            font-size: 1rem;
        }
    }

    .tileContent {
        bottom: 2.6rem !important;
        font-size: 1rem;
        color: #444;
        line-height: 1.4;
        height: 2.8rem;
        overflow: hidden;
        display: -webkit-box;
        -webkit-line-clamp: 2;
        -webkit-box-orient: vertical;
        text-overflow: ellipsis;
        position: absolute;
        left: .9375rem;
        right: .9375rem;
        z-index: 2;
    }

    .tile-actions {
        position: absolute;
        top: 0.5rem;
        right: 0.5rem;
        z-index: 10;
        display: flex;
        gap: 0.25rem;
    }

    .tile-extra-children {
        position: relative;
        z-index: 2;
    }
</style>
