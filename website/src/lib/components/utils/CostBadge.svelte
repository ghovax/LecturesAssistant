<script lang="ts">
  interface Props {
    amount: number;
    currency?: string;
    showSymbol?: boolean;
    class?: string;
  }

  let {
    amount,
    currency = "$",
    showSymbol = true,
    class: className = "",
  }: Props = $props();

  let formattedAmount = $derived(amount.toFixed(4));
</script>

<span class="cost-badge {className}">
  {#if showSymbol}{currency}
  {/if}{formattedAmount}
</span>

<style lang="scss">
  .cost-badge {
    font-size: calc(0.7rem * var(--font-mono-size-factor));
    color: var(--gray-400);
    font-family: var(--font-mono);
    white-space: nowrap;
  }
</style>
