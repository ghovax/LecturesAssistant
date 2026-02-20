<script lang="ts">
  import { X } from "lucide-svelte";
  import type { Snippet } from "svelte";

  interface Props {
    title: string;
    isOpen: boolean;
    onClose: () => void;
    children: Snippet;
    footer?: Snippet;
    maxWidth?: string;
  }

  let {
    title,
    isOpen,
    onClose,
    children,
    footer,
    maxWidth = "500px",
  }: Props = $props();

  function handleBackdropClick(e: MouseEvent) {
    if (e.target === e.currentTarget) {
      onClose();
    }
  }
</script>

{#if isOpen}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="modal fade show d-block"
    tabindex="-1"
    style="background: rgba(0,0,0,0.4); backdrop-filter: blur(2px); z-index: 1050;"
    onclick={handleBackdropClick}
  >
    <div
      class="modal-dialog modal-dialog-centered"
      style="max-width: {maxWidth};"
    >
      <div class="modal-content shadow-lg">
        <!-- Header -->
        <div class="standard-header px-3">
          <div class="header-title">
            <span class="header-text">{title}</span>
          </div>
          <button
            class="btn btn-link btn-sm text-muted p-0 d-flex align-items-center shadow-none border-0"
            onclick={onClose}
          >
            <X size={20} />
          </button>
        </div>

        <!-- Body -->
        <div class="modal-body p-3 bg-white">
          {@render children()}
        </div>

        <!-- Footer -->
        {#if footer}
          <div class="px-3 py-3 bg-white border-top">
            {@render footer()}
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}

<style lang="scss">
  .modal-content {
    border-radius: var(--border-radius) !important;
    border: 1px solid var(--gray-300) !important;
  }
</style>
