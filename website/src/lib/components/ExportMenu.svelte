<script lang="ts">
    import { Loader2, Download } from 'lucide-svelte';
    import { onMount } from 'svelte';

    interface Props {
        isCompleted: boolean;
        isExportingPDFWithImages: boolean;
        isExportingPDFNoImages: boolean;
        isExportingDocx: boolean;
        onExport: (format: string, includeImages: boolean) => void;
    }

    let {
        isCompleted,
        isExportingPDFWithImages,
        isExportingPDFNoImages,
        isExportingDocx,
        onExport
    }: Props = $props();

    let isOpen = $state(false);
    let dropdownElement: HTMLDivElement | null = $state(null);

    function handleClickOutside(event: MouseEvent) {
        if (dropdownElement && !dropdownElement.contains(event.target as Node)) {
            isOpen = false;
        }
    }

    onMount(() => {
        document.addEventListener('mousedown', handleClickOutside);
        return () => {
            document.removeEventListener('mousedown', handleClickOutside);
        };
    });
</script>

<div class="dropdown" bind:this={dropdownElement}>
    <button
        class="btn btn-link {isCompleted ? 'text-orange' : 'text-muted'} p-0 border-0 shadow-none dropdown-toggle no-caret"
        title="Export Options"
        aria-label="Export Options"
        disabled={isExportingPDFWithImages || isExportingPDFNoImages || isExportingDocx}
        onclick={() => isOpen = !isOpen}
        aria-expanded={isOpen}
    >
        <Download size={16} />
    </button>
    {#if isOpen}
        <ul class="dropdown-menu dropdown-menu-end show">
            <li>
                <button
                    class="dropdown-item d-flex justify-content-between align-items-center"
                    onclick={() => { onExport('pdf', true); isOpen = false; }}
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
                    onclick={() => { onExport('pdf', false); isOpen = false; }}
                    disabled={isExportingPDFNoImages}
                >
                    Export PDF (text only)
                    {#if isExportingPDFNoImages}
                        <Loader2 size={14} class="spinner-animation ms-2" />
                    {/if}
                </button>
            </li>
            <li>
                <button
                    class="dropdown-item d-flex justify-content-between align-items-center"
                    onclick={() => { onExport('docx', true); isOpen = false; }}
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
        position: absolute;
        top: 125%;
        right: 0;
        z-index: 9999;
    }
</style>
