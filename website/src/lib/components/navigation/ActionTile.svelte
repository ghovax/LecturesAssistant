<script lang="ts">
  import type { Snippet } from "svelte";
  import Highlighter from "../Highlighter.svelte";

  interface Props {
    href?: string;
    title: string;
    monospaceTitle?: boolean;
    description?: Snippet;
    children?: Snippet;
    actions?: Snippet;
    onclick?: (e: MouseEvent) => void;
    class?: string;
    disabled?: boolean;
    cost?: number;
    width?: string;
    height?: string;
  }

  let {
    href,
    title,
    monospaceTitle = false,
    description,
    children,
    actions,
    onclick,
    class: className = "",
    disabled = false,
    cost,
    width = "250px",
    height = "150px",
  }: Props = $props();

  let isHovered = $state(false);
</script>

<div
  class="action-tile {className}"
  class:disabled
  onmouseenter={() => (isHovered = true)}
  onmouseleave={() => (isHovered = false)}
>
  {#if href}
    <a {href} {onclick} class="action-tile-link">
      <p class="action-tile-title" class:font-monospace={monospaceTitle}>
        <Highlighter show={isHovered} padding={0} offsetY={2}>
          <strong>{title}</strong>
        </Highlighter>
      </p>

      {#if description}
        <div class="action-tile-content">
          {@render description()}
        </div>
      {/if}

      {#if children}
        <div class="action-tile-extra">
          {@render children()}
        </div>
      {/if}

      {#if cost && cost > 0}
        <div class="action-tile-cost">
          ${cost.toFixed(4)}
        </div>
      {/if}
    </a>
  {:else}
    <button type="button" class="action-tile-button" {onclick} {disabled}>
      <p class="action-tile-title" class:font-monospace={monospaceTitle}>
        <Highlighter show={isHovered} padding={0} offsetY={2}>
          <strong>{title}</strong>
        </Highlighter>
      </p>

      {#if description}
        <div class="action-tile-content">
          {@render description()}
        </div>
      {/if}

      {#if children}
        <div class="action-tile-extra">
          {@render children()}
        </div>
      {/if}

      {#if cost && cost > 0}
        <div class="action-tile-cost">
          ${cost.toFixed(4)}
        </div>
      {/if}
    </button>
  {/if}

  {#if actions}
    <div class="action-tile-actions">
      {@render actions()}
    </div>
  {/if}
</div>

<style lang="scss">
  .action-tile {
    display: inline-block;
    position: relative;
    vertical-align: top;
    background: #fff;
    transition: all 0.2s ease;

    &:hover:not(.disabled) {
      background: #fafaf9;
      z-index: 10;
    }

    &.disabled {
      opacity: 0.6;
      cursor: not-allowed;
    }

    &.processing {
      background: #fafaf9;
      opacity: 0.8;

      .action-tile-title {
        color: var(--gray-500);
      }
    }

    &.error {
      background: #fffafa;

      .action-tile-title {
        color: #b91c1c;
      }
    }
  }

  .action-tile-link,
  .action-tile-button {
    display: flex;
    flex-direction: column;
    height: v-bind(height);
    width: v-bind(width);
    padding: 20px;
    text-decoration: none;
    text-align: left;
    position: relative;
    z-index: 1;
    font-family: var(--font-primary);
    color: var(--gray-900);
    background: transparent;
    border: none;
    cursor: pointer;

    &:focus-visible {
      outline: 2px solid var(--orange);
      outline-offset: -2px;
    }

    &:disabled {
      cursor: not-allowed;
    }
  }

  .action-tile-button:disabled {
    cursor: not-allowed;
  }

  .action-tile-title {
    font-size: 0.9rem;
    font-weight: 600;
    margin: 0 0 8px;
    line-height: 1.2;

    &.font-monospace {
      font-family: var(--font-mono);
      font-size: 0.85rem;
      font-size-adjust: var(--font-mono-adjust);
    }
  }

  .action-tile-content {
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

  .action-tile-actions {
    position: absolute;
    bottom: 12px;
    right: 20px;
    z-index: 10;
    display: flex;
    gap: 8px;
  }

  .action-tile-extra {
    margin-top: auto;
    margin-bottom: 20px;
    position: relative;
    z-index: 2;
  }

  .action-tile-cost {
    position: absolute;
    bottom: 12px;
    left: 20px;
    font-size: 0.7rem;
    color: var(--gray-400);
    font-family: var(--font-mono);
    font-size-adjust: var(--font-mono-adjust);
    pointer-events: none;
  }
</style>
