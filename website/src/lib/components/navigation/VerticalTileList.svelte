<script lang="ts">
  import type { Snippet } from "svelte";
  import ActionTile from "./ActionTile.svelte";

  interface Props {
    children: Snippet;
    direction?: "horizontal" | "vertical";
    class?: string;
  }

  let {
    direction = "horizontal",
    children,
    class: className = "",
  }: Props = $props();
</script>

<div
  class="vertical-tile-list {className}"
  class:vertical={direction === "vertical"}
>
  {@render children()}
</div>

<style lang="scss">
  .vertical-tile-list {
    display: flex;
    flex-wrap: wrap;
    gap: 0;
    background: transparent;
    overflow: visible;

    &.vertical {
      flex-direction: column;
      flex-wrap: nowrap;
      gap: 0;

      :global(.action-tile) {
        margin: 0;
        width: 250px;
        border-right: none;
        border-bottom: 1px solid var(--gray-300);
        border-radius: 0 !important;

        &:last-child {
          border-bottom: none;
          border-radius: 0 0 var(--border-radius) var(--border-radius) !important;
        }

        &:first-child {
          border-radius: var(--border-radius) var(--border-radius) 0 0 !important;
        }

        &.tile-locked {
          background-color: #fafaf9;
          opacity: 0.8;
        }

        .action-tile-link,
        .action-tile-button {
          width: 100%;
          padding-right: 100px;
        }

        .action-tile-actions {
          position: absolute;
          top: 50%;
          right: 16px;
          transform: translateY(-50%);
        }
      }

      &.adaptive-width {
        border: 1px solid var(--gray-300);
        border-radius: var(--border-radius);
        overflow: hidden;

        :global(.action-tile) {
          width: 100%;
          border-bottom: 1px solid var(--gray-300);
          border-radius: 0 !important;

          &:first-child {
            border-top: none;
          }

          &:last-child {
            border-bottom: none;
          }
        }
      }
    }

    :global(.action-tile) {
      width: 250px;
      border-right: 1px solid var(--gray-300);
      border-radius: 0 !important;

      &:last-child {
        border-right: none;
        border-radius: 0 var(--border-radius) var(--border-radius) 0 !important;
      }

      &:first-child {
        border-radius: var(--border-radius) 0 0 var(--border-radius) !important;
      }
    }
  }
</style>
