<script lang="ts">
  import { Loader2, Download, ExternalLink } from "lucide-svelte";
  import { onMount } from "svelte";

  interface Props {
    isCompleted: boolean;
    isExportingPDFWithImages: boolean;
    isExportingPDFNoImages: boolean;
    isExportingDocx: boolean;
    onExport: (format: string, includeImages: boolean) => void;
    onOpenPdf?: () => void;
    showImageOptions?: boolean;
  }

  let {
    isCompleted,
    isExportingPDFWithImages,
    isExportingPDFNoImages,
    isExportingDocx,
    onExport,
    onOpenPdf,
    showImageOptions = true,
  }: Props = $props();

  let isOpen = $state(false);
  let buttonElement: HTMLButtonElement | null = $state(null);
  let dropdownElement: HTMLUListElement | null = $state(null);
  let dropdownStyle = $state("");

  function openDropdown() {
    if (buttonElement) {
      const rect = buttonElement.getBoundingClientRect();
      dropdownStyle = `top: ${rect.bottom + 4}px; right: ${window.innerWidth - rect.right}px;`;
    }
    isOpen = true;
  }

  function toggleDropdown() {
    if (isOpen) {
      isOpen = false;
    } else {
      openDropdown();
    }
  }

  function handleClickOutside(event: MouseEvent) {
    if (
      buttonElement &&
      !buttonElement.contains(event.target as Node) &&
      dropdownElement &&
      !dropdownElement.contains(event.target as Node)
    ) {
      isOpen = false;
    }
  }

  onMount(() => {
    document.addEventListener("mousedown", handleClickOutside);
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  });
</script>

<div class="dropdown">
  <button
    bind:this={buttonElement}
    class="btn btn-link {isCompleted
      ? 'text-orange'
      : 'text-muted'} p-0 border-0 shadow-none dropdown-toggle no-caret"
    title="Export Options"
    aria-label="Export Options"
    disabled={isExportingPDFWithImages ||
      isExportingPDFNoImages ||
      isExportingDocx}
    onclick={toggleDropdown}
    aria-expanded={isOpen}
  >
    <Download size={16} />
  </button>
  {#if isOpen}
    <ul
      bind:this={dropdownElement}
      class="dropdown-menu dropdown-menu-end show"
      style={dropdownStyle}
    >
      {#if onOpenPdf}
        <li>
          <button
            class="dropdown-item d-flex align-items-center"
            onclick={() => {
              onOpenPdf?.();
              isOpen = false;
            }}
          >
            <ExternalLink size={14} class="me-2" />
            Open PDF
          </button>
        </li>
        <li><hr class="dropdown-divider" /></li>
      {/if}
      {#if showImageOptions}
        <li>
          <button
            class="dropdown-item d-flex justify-content-between align-items-center"
            onclick={() => {
              onExport("pdf", true);
              isOpen = false;
            }}
            disabled={isExportingPDFWithImages}
          >
            Export PDF (with images)
            {#if isExportingPDFWithImages}
              <Loader2 size={14} class="spinner-animation ms-2" />
            {/if}
          </button>
        </li>
        <li>
          <button
            class="dropdown-item d-flex justify-content-between align-items-center"
            onclick={() => {
              onExport("pdf", false);
              isOpen = false;
            }}
            disabled={isExportingPDFNoImages}
          >
            Export PDF (text only)
            {#if isExportingPDFNoImages}
              <Loader2 size={14} class="spinner-animation ms-2" />
            {/if}
          </button>
        </li>
      {:else}
        <li>
          <button
            class="dropdown-item d-flex justify-content-between align-items-center"
            onclick={() => {
              onExport("pdf", true);
              isOpen = false;
            }}
            disabled={isExportingPDFWithImages || isExportingPDFNoImages}
          >
            Export PDF
            {#if isExportingPDFWithImages || isExportingPDFNoImages}
              <Loader2 size={14} class="spinner-animation ms-2" />
            {/if}
          </button>
        </li>
      {/if}
      <li>
        <button
          class="dropdown-item d-flex justify-content-between align-items-center"
          onclick={() => {
            onExport("docx", true);
            isOpen = false;
          }}
          disabled={isExportingDocx}
        >
          Export Word
          {#if isExportingDocx}
            <Loader2 size={14} class="spinner-animation ms-2" />
          {/if}
        </button>
      </li>
    </ul>
  {/if}
</div>

<style lang="scss">
  .dropdown-menu.show {
    display: block;
    position: fixed;
    z-index: 9999;
  }
</style>
