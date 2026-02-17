<script lang="ts">
  import type { Snippet } from "svelte";
  import type { ComponentType } from "svelte";
  import { ChevronRight } from "lucide-svelte";

  interface Props {
    href?: string;
    icon?: ComponentType<any>;
    iconSize?: number;
    title: string;
    description?: string;
    metadata?: Snippet;
    onclick?: (e: MouseEvent) => void;
    external?: boolean;
    disabled?: boolean;
    class?: string;
  }

  let {
    href,
    icon: Icon,
    iconSize = 20,
    title,
    description,
    metadata,
    onclick,
    external = false,
    disabled = false,
    class: className = "",
  }: Props = $props();

  let isHovered = $state(false);

  function handleClick(e: MouseEvent) {
    if (disabled) {
      e.preventDefault();
      return;
    }
    onclick?.(e);
  }
</script>

<div
  class="nav-card {className}"
  class:disabled
  onmouseenter={() => (isHovered = true)}
  onmouseleave={() => (isHovered = false)}
>
  {#if href}
    <a
      {href}
      class="nav-card-link"
      target={external ? "_blank" : undefined}
      rel={external ? "noopener noreferrer" : undefined}
      onclick={handleClick}
    >
      <div class="nav-card-content">
        {#if Icon}
          <div class="nav-card-icon">
            <Icon size={iconSize} />
          </div>
        {/if}
        <div class="nav-card-text">
          <h3 class="nav-card-title">{title}</h3>
          {#if description}
            <p class="nav-card-description">{description}</p>
          {/if}
          {#if metadata}
            <div class="nav-card-metadata">
              {@render metadata()}
            </div>
          {/if}
        </div>
        <div class="nav-card-chevron">
          <ChevronRight size={18} />
        </div>
      </div>
    </a>
  {:else}
    <button
      class="nav-card-button"
      type="button"
      onclick={handleClick}
      {disabled}
    >
      <div class="nav-card-content">
        {#if Icon}
          <div class="nav-card-icon">
            <Icon size={iconSize} />
          </div>
        {/if}
        <div class="nav-card-text">
          <h3 class="nav-card-title">{title}</h3>
          {#if description}
            <p class="nav-card-description">{description}</p>
          {/if}
          {#if metadata}
            <div class="nav-card-metadata">
              {@render metadata()}
            </div>
          {/if}
        </div>
        <div class="nav-card-chevron">
          <ChevronRight size={18} />
        </div>
      </div>
    </button>
  {/if}
</div>

<style lang="scss">
  .nav-card {
    background: #fff;
    border: 1px solid var(--gray-300);
    transition: all 0.2s ease;
    width: 100%;

    &:hover:not(.disabled) {
      background: #fafaf9;
      border-color: var(--gray-400);
    }

    &.disabled {
      opacity: 0.6;
      cursor: not-allowed;
    }
  }

  .nav-card-link,
  .nav-card-button {
    display: block;
    width: 100%;
    padding: 1.25rem;
    text-decoration: none;
    background: transparent;
    border: none;
    text-align: left;
    cursor: pointer;

    &:focus-visible {
      outline: 2px solid var(--orange);
      outline-offset: -2px;
    }
  }

  .nav-card-button {
    &:disabled {
      cursor: not-allowed;
    }
  }

  .nav-card-content {
    display: flex;
    align-items: flex-start;
    gap: 1rem;
  }

  .nav-card-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--orange);
    flex-shrink: 0;
    width: 40px;
    height: 40px;
    background: var(--cream);
    border: 1px solid var(--gray-300);
  }

  .nav-card-text {
    flex: 1;
    min-width: 0;
  }

  .nav-card-title {
    font-size: 0.95rem;
    font-weight: 600;
    color: var(--gray-900);
    margin: 0 0 0.25rem 0;
    line-height: 1.3;
  }

  .nav-card-description {
    font-size: 0.8rem;
    color: var(--gray-500);
    line-height: 1.4;
    margin: 0;
  }

  .nav-card-metadata {
    margin-top: 0.5rem;
    font-size: 0.75rem;
    color: var(--gray-400);
  }

  .nav-card-chevron {
    display: flex;
    align-items: center;
    color: var(--gray-400);
    flex-shrink: 0;
    transition: transform 0.2s ease;
  }

  .nav-card:hover:not(.disabled) .nav-card-chevron {
    transform: translateX(4px);
    color: var(--orange);
  }
</style>
