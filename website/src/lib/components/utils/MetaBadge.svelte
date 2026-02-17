<script lang="ts">
  interface Props {
    type: "page" | "time" | "count" | "item";
    current: number | string;
    total?: number | string;
    label?: string;
    compact?: boolean;
    class?: string;
  }

  let {
    type,
    current,
    total,
    label,
    compact = false,
    class: className = "",
  }: Props = $props();

  let icon = $derived(() => {
    switch (type) {
      case "page":
        return "📄";
      case "time":
        return "⏱";
      case "count":
      case "item":
        return "#";
      default:
        return "•";
    }
  });
</script>

<div class="meta-badge {className}" class:compact>
  {#if label}
    <span class="meta-label">{label}</span>
  {/if}
  <span class="meta-value">
    {#if !compact}{icon}
    {/if}{current}
  </span>
  {#if total}
    <span class="meta-separator">of</span>
    <span class="meta-total">{total}</span>
  {/if}
</div>

<style lang="scss">
  .meta-badge {
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

    &.compact {
      padding: 0.125rem 0.5rem;
      font-size: 10px;
      height: 1.5rem;
      gap: 0.375rem;
    }
  }

  .meta-label {
    color: var(--gray-500);
  }

  .meta-value {
    color: var(--gray-900);
    display: flex;
    align-items: center;
    gap: 0.25rem;
  }

  .meta-separator {
    color: var(--gray-400);
    font-weight: 400;
    text-transform: lowercase;
    font-style: italic;
    padding: 0 0.125rem;
  }

  .meta-total {
    color: var(--gray-500);
  }
</style>
