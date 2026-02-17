<script lang="ts">
  import type { Snippet } from "svelte";

  interface Props {
    title?: string;
    children: Snippet;
    headerActions?: Snippet;
    class?: string;
    fitContent?: boolean;
  }

  let {
    title,
    children,
    headerActions,
    class: className = "",
    fitContent = false,
  }: Props = $props();
</script>

<div class="card-container {className}" class:fit-content={fitContent}>
  {#if title || headerActions}
    <div class="standard-header">
      <div class="header-title">
        {#if title}
          <span class="header-text">{title}</span>
        {/if}
      </div>
      {#if headerActions}
        <div class="header-actions">
          {@render headerActions()}
        </div>
      {/if}
    </div>
  {/if}
  <div class="card-content">
    {@render children()}
  </div>
</div>

<style lang="scss">
  .card-container {
    background: #fff;
    border: 1px solid var(--gray-300);
    margin-bottom: 1rem;
    width: 100%;

    &.fit-content {
      width: fit-content;
      max-width: 100%;
    }
  }

  .card-content {
    padding: 0;
  }
</style>
