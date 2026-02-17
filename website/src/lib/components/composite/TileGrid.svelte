<script lang="ts">
  import type { Snippet } from "svelte";

  interface Props {
    children: Snippet;
    columns?: "auto" | "2" | "3" | "4";
    gap?: "sm" | "md" | "lg";
    class?: string;
  }

  let {
    columns = "auto",
    gap = "md",
    children,
    class: className = "",
  }: Props = $props();

  const gapClasses = {
    sm: "gap-sm",
    md: "gap-md",
    lg: "gap-lg",
  };
</script>

<div class="tile-grid {className} {gapClasses[gap]} grid-{columns}">
  {@render children()}
</div>

<style lang="scss">
  .tile-grid {
    display: flex;
    flex-wrap: wrap;
    gap: 0;

    &.gap-sm {
      gap: 0;
    }

    &.gap-md {
      gap: 0;
    }

    &.gap-lg {
      gap: 0;
    }

    &.grid-2 {
      :global(.action-tile) {
        @media (min-width: 768px) {
          width: calc(50% - 0px);
        }
      }
    }

    &.grid-3 {
      :global(.action-tile) {
        @media (min-width: 768px) {
          width: calc(33.333% - 0px);
        }
      }
    }

    &.grid-4 {
      :global(.action-tile) {
        @media (min-width: 768px) {
          width: calc(25% - 0px);
        }
      }
    }
  }
</style>
