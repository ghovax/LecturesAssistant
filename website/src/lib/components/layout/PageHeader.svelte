<script lang="ts">
  import type { Snippet } from "svelte";

  interface Props {
    title: string;
    description?: string;
    actions?: Snippet;
    children?: Snippet;
    class?: string;
  }

  let {
    title,
    description,
    actions,
    children,
    class: className = "",
  }: Props = $props();
</script>

<header class="page-header {className}">
  <div class="page-header-content">
    <div class="page-title-group">
      <h1 class="page-title">{title}</h1>
      {#if description}
        <p class="page-description">{description}</p>
      {/if}
    </div>
    {#if actions}
      <div class="page-actions">
        {@render actions()}
      </div>
    {:else if children}
      <div class="page-actions">
        {@render children()}
      </div>
    {/if}
  </div>
</header>

<style lang="scss">
  .page-header {
    margin-bottom: 2.5rem;
  }

  .page-header-content {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 1rem;
  }

  .page-title-group {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .page-title {
    font-size: 1.75rem;
    font-weight: 500;
    color: var(--gray-900);
    letter-spacing: -0.02em;
    line-height: 1;
    margin: 0;
    display: inline-flex;
    align-items: center;
  }

  .page-description {
    font-family: var(--font-primary);
    font-size: 0.95rem;
    line-height: 1.6;
    color: var(--gray-600);
    margin: 0;
    max-width: 600px;
  }

  .page-actions {
    display: flex;
    gap: 0.75rem;
    align-items: center;
    flex-shrink: 0;
  }
</style>
