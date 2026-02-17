<script lang="ts">
  import type { Snippet } from "svelte";

  interface Props {
    children: Snippet;
    variant?: "default" | "muted" | "emphasized";
    class?: string;
  }

  let {
    variant = "default",
    children,
    class: className = "",
  }: Props = $props();
</script>

<div class="content-block {className} {variant}">
  {@render children()}
</div>

<style lang="scss">
  .content-block {
    padding: 1.5rem;
    background: #fff;
    border: 1px solid var(--gray-300);

    &.muted {
      background: var(--cream);
      border-color: var(--gray-200);
    }

    &.emphasized {
      border-width: 2px;
      border-color: var(--orange);
    }
  }
</style>
