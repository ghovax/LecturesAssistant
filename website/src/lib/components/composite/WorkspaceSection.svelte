<script lang="ts">
  import type { Snippet } from "svelte";

  interface Props {
    title?: string;
    overline?: string;
    children: Snippet;
    actions?: Snippet;
    variant?: "card" | "section" | "page";
    class?: string;
  }

  let {
    title,
    overline,
    children,
    actions,
    variant = "card",
    class: className = "",
  }: Props = $props();
</script>

<div class="workspace-section {className} {variant}">
  {#if overline || title || actions}
    <div class="workspace-header">
      <div class="workspace-title-group">
        {#if overline}
          <span class="workspace-overline">{overline}</span>
        {/if}
        {#if title}
          <h2 class="workspace-title">{title}</h2>
        {/if}
      </div>
      {#if actions}
        <div class="workspace-actions">
          {@render actions()}
        </div>
      {/if}
    </div>
  {/if}
  <div class="workspace-content">
    {@render children()}
  </div>
</div>

<style lang="scss">
  .workspace-section {
    margin-bottom: 60px;

    &:last-child {
      margin-bottom: 0;
    }

    &.card {
      background: #fff;
      border: 1px solid var(--gray-300);
      border-radius: var(--border-radius);
      padding: 20px;

      .workspace-header {
        margin-bottom: 1.5rem;
      }
    }

    &.section {
      .workspace-header {
        margin-bottom: 1.5rem;
      }
    }

    &.page {
      margin-bottom: 2.5rem;
    }
  }

  .workspace-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 1rem;
  }

  .workspace-title-group {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .workspace-overline {
    font-size: 10px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.12em;
    color: var(--gray-500);
    display: block;
  }

  .workspace-title {
    font-size: 18px;
    font-weight: 500;
    color: var(--gray-900);
    margin: 0;
    line-height: 1.2;
  }

  .workspace-actions {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  .workspace-content {
    /* Content styling handled by children components */
  }
</style>
