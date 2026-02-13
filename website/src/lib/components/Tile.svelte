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
        margin: .4rem;
        position: relative;
        vertical-align: top;
        background: #fff;
        border: 1px solid #eee;
        @extend .shadow-kakimashou;
        transition: transform 0.1s ease, box-shadow 0.1s ease, border-color 0.1s ease;
        &:hover {
            border-color: #568f27;
        }
        &:active {
            box-shadow: .1rem .1rem .3rem rgba(0, 0, 0, .2) !important;
            transform: translateY(1px);
        }
    }
    /* Common styles for both a and button */
    a, button.tile-button {
        appearance: none;
        border: none;
        background: #fff;
        color: #000;
        display: block;
        height: 7rem; /* Increased for better text fitting */
        width: 14.5rem; /* Increased for better text fitting */
        overflow: hidden;
        padding: .85rem;
        text-decoration: none;
        text-align: left;
        transition: background-color 0.1s ease;
        position: relative;
        z-index: 1;
        &:hover:not(:disabled) {
            background: #fcfcfc;
        }
        &:focus-visible {
            outline: 2px solid #568f27;
            outline-offset: -2px;
        }
        &:disabled {
            opacity: 0.6;
            cursor: not-allowed;
            background: #f5f5f5;
        }
    }
    .tile-glyph {
        color: #f0f1e8;
        font-family: "ＭＳ Ｐゴシック", "MS PGothic", "メイリオ", Meiryo, sans-serif;
        font-size: 12rem; /* Slightly larger glyph */
        left: 5.5rem;
        line-height: 7rem;
        position: absolute;
        top: 2.2rem;
        z-index: 1;
        pointer-events: none;
    }
    .tile-title {
        bottom: .85rem !important;
        font-size: 1.1rem;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        position: absolute;
        left: .85rem;
        right: .85rem;
        z-index: 2;
        margin: 0;
        line-height: 1.2;
        &.font-monospace {
            font-family: monospace;
            font-size: 0.95rem;
        }
    }
    .tileContent {
        bottom: 2.3rem !important;
        font-size: 0.9rem;
        color: #555;
        line-height: 1.35;
        height: 2.7rem;
        overflow: hidden;
        display: -webkit-box;
        -webkit-line-clamp: 2;
        -webkit-box-orient: vertical;
        text-overflow: ellipsis;
        position: absolute;
        left: .85rem;
        right: .85rem;
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
