<script lang="ts">
  import type { Snippet } from "svelte";

  interface Props {
    type?: "button" | "submit" | "reset";
    variant?:
      | "primary"
      | "success"
      | "danger"
      | "white"
      | "outline-primary"
      | "outline-success"
      | "link";
    size?: "sm" | "md" | "lg";
    onclick?: (e: MouseEvent) => void;
    disabled?: boolean;
    class?: string;
    children: Snippet;
    href?: string;
  }

  let {
    type = "button",
    variant = "primary",
    size = "md",
    onclick,
    disabled = false,
    class: className = "",
    children,
    href,
  }: Props = $props();

  const sizeClass = size === "sm" ? "btn-sm" : size === "lg" ? "btn-lg" : "";
  const variantClass = `btn-${variant}`;
</script>

{#if href}
  <a
    {href}
    class="cozy-btn {variantClass} {sizeClass} {className}"
    class:disabled
    onclick={(e) => !disabled && onclick?.(e)}
  >
    {@render children()}
  </a>
{:else}
  <button
    {type}
    class="cozy-btn {variantClass} {sizeClass} {className}"
    {disabled}
    {onclick}
  >
    {@render children()}
  </button>
{/if}

<style lang="scss">
  .cozy-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    font-family: var(--font-primary);
    font-weight: 600;
    font-size: 0.8rem;
    padding: 0 1.25rem;
    height: 2.5rem;
    border-radius: var(--border-radius);
    border: 1px solid rgba(0, 0, 0, 0.15);
    transition: all 0.15s ease;
    cursor: pointer;
    text-decoration: none;
    line-height: 1;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    box-shadow:
      0 1px 2px rgba(0, 0, 0, 0.05),
      0 2px 0 rgba(0, 0, 0, 0.1);
    transform: translateY(0);

    &:active {
      transform: translateY(2px);
      box-shadow:
        0 0 0 rgba(0, 0, 0, 0.05),
        0 0 0 rgba(0, 0, 0, 0.1);
    }

    &.disabled {
      opacity: 0.5;
      cursor: not-allowed;
      pointer-events: none;
    }

    &.btn-sm {
      height: 1.75rem;
      padding: 0 0.75rem;
      font-size: 0.7rem;
    }

    &.btn-lg {
      height: 3rem;
      padding: 0 2rem;
      font-size: 0.85rem;
    }

    &.btn-primary,
    &.btn-success {
      background: var(--gray-900);
      color: var(--cream);
      border-color: rgba(0, 0, 0, 0.3);

      &:hover {
        background: var(--gray-800);
        border-color: rgba(0, 0, 0, 0.35);
      }
    }

    &.btn-danger {
      background: #b91c1c;
      color: white;
      border-color: rgba(0, 0, 0, 0.3);
      &:hover {
        background: #991b1b;
        border-color: rgba(0, 0, 0, 0.35);
      }
    }

    &.btn-white {
      background: white;
      color: var(--gray-800);
      border-color: rgba(0, 0, 0, 0.2);
      &:hover {
        background: var(--gray-200);
        border-color: rgba(0, 0, 0, 0.25);
      }
    }

    &.btn-outline-primary,
    &.btn-outline-success {
      background: transparent;
      color: var(--gray-800);
      border-color: rgba(0, 0, 0, 0.2);
      &:hover {
        background: var(--gray-200);
        border-color: rgba(0, 0, 0, 0.3);
      }
    }

    &.btn-link {
      background: transparent;
      color: var(--gray-600);
      padding: 0;
      border: none;
      box-shadow: none;
      transform: none;
      &:hover {
        color: var(--orange);
        text-decoration: underline;
      }
      &:active {
        transform: none;
      }
    }
  }
</style>
