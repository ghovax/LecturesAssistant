<script lang="ts">
  import type { Snippet } from "svelte";

  interface Props {
    title?: string;
    overline?: string;
    children: Snippet;
    actions?: Snippet;
    class?: string;
  }

  let {
    title,
    overline,
    children,
    actions,
    class: className = "",
  }: Props = $props();
</script>

<section class="custom-section {className}">
  {#if overline || title || actions}
    <div class="section-header">
      <div class="section-title-group">
        {#if overline}
          <span class="section-overline">{overline}</span>
        {/if}
        {#if title}
          <h2 class="section-title">{title}</h2>
        {/if}
      </div>
      {#if actions}
        <div class="section-actions">
          {@render actions()}
        </div>
      {/if}
    </div>
  {/if}
  <div class="section-content">
    {@render children()}
  </div>
</section>

<style lang="scss">
  .custom-section {
    margin-bottom: 60px;

    &:last-child {
      margin-bottom: 0;
    }
  }

  .section-header {
    margin-bottom: 24px;
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 1rem;
  }

  .section-title-group {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .section-overline {
    font-size: 0.625rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.12em;
    color: var(--gray-500);
    display: block;
  }

  .section-title {
    font-size: 1.125rem;
    font-weight: 500;
    color: var(--gray-900);
    margin: 0;
    line-height: 1.2;
  }

  .section-actions {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  .section-content {
    /* Content styling handled by children components */
  }
</style>
