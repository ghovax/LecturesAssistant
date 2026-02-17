<script lang="ts">
  import type { Snippet } from "svelte";
  import Highlighter from "../Highlighter.svelte";

  interface Props {
    text: string;
    monospace?: boolean;
    highlightOnHover?: boolean;
    highlightColor?: string;
    class?: string;
  }

  let {
    text,
    monospace = false,
    highlightOnHover = false,
    highlightColor,
    class: className = "",
  }: Props = $props();

  let isHovered = $state(false);
</script>

<span
  class="text-with-highlight {className}"
  class:font-monospace={monospace}
  onmouseenter={() => (isHovered = true)}
  onmouseleave={() => (isHovered = false)}
>
  {#if highlightOnHover}
    <Highlighter
      show={isHovered}
      padding={0}
      offsetY={2}
      color={highlightColor}
    >
      <strong>{text}</strong>
    </Highlighter>
  {:else}
    {text}
  {/if}
</span>

<style lang="scss">
  .text-with-highlight {
    &.font-monospace {
      font-family: var(--font-mono);
      font-size-adjust: var(--font-mono-adjust);
    }
  }
</style>
