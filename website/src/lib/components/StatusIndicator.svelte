<script lang="ts">
  import { Clock, FileText, Hash } from "lucide-svelte";

  interface Props {
    type: "page" | "time" | "count";
    current: number | string;
    total?: number | string;
    label?: string;
    class?: string;
  }

  let { type, current, total, label, class: className = "" }: Props = $props();
</script>

<div class="status-indicator {className}">
  <span class="indicator-icon">
    {#if type === "page"}
      <FileText size={14} />
    {:else if type === "time"}
      <Clock size={14} />
    {:else}
      <Hash size={14} />
    {/if}
  </span>

  <span class="indicator-text">
    {#if label}{label}
    {/if}
    <span class="current">{current}</span>
    {#if total}
      <span class="separator">–</span>
      <span class="total">{total}</span>
    {/if}
  </span>
</div>

<style lang="scss">
  .status-indicator {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.25rem 0.75rem;
    background-color: var(--cream);
    border: 1px solid var(--gray-300);
    font-family: var(--font-primary);
    font-size: 11px;
    font-weight: 600;
    color: var(--gray-600);
    letter-spacing: 0.02em;
    text-transform: uppercase;
    height: 1.75rem;
  }

  .indicator-icon {
    display: flex;
    align-items: center;
    color: var(--orange);
  }

  .indicator-text {
    display: flex;
    align-items: center;
    gap: 0.25rem;
  }

  .current {
    color: var(--gray-900);
  }

  .separator {
    color: var(--gray-400);
    font-weight: 400;
    text-transform: lowercase;
    font-style: italic;
    padding: 0 0.125rem;
  }

  .total {
    color: var(--gray-500);
  }

  :global(.transcript-nav) .status-indicator,
  :global(.document-nav) .status-indicator {
    background-color: #fff;
    border-color: var(--gray-300);
  }
</style>
