<script lang="ts">
    import { X, FileText } from 'lucide-svelte';

    interface Props {
        content: string;
        title?: string;
        sourceFile?: string;
        sourcePages?: number[];
        onClose: () => void;
        x: number;
        y: number;
    }

    let { content, title = 'Citation', sourceFile, sourcePages, onClose, x, y }: Props = $props();

    // Ensure the popup stays within viewport
    let popupElement: HTMLDivElement | null = $state(null);
    let adjustedX = $state(x + 10); // Offset from cursor
    let adjustedY = $state(y + 10);

    let formattedPages = $derived.by(() => {
        if (!sourcePages || sourcePages.length === 0) return '';
        // Basic range logic if contiguous
        if (sourcePages.length > 1 && sourcePages[sourcePages.length-1] - sourcePages[0] === sourcePages.length - 1) {
            return `pp. ${sourcePages[0]}-${sourcePages[sourcePages.length-1]}`;
        }
        return (sourcePages.length === 1 ? 'p. ' : 'pp. ') + sourcePages.join(', ');
    });

    $effect(() => {
        if (popupElement) {
            const rect = popupElement.getBoundingClientRect();
            const viewportWidth = window.innerWidth;
            const viewportHeight = window.innerHeight;

            if (adjustedX + rect.width > viewportWidth) {
                adjustedX = viewportWidth - rect.width - 20;
            }
            if (adjustedY + rect.height > viewportHeight) {
                adjustedY = viewportHeight - rect.height - 20;
            }
            
            if (adjustedX < 10) adjustedX = 10;
            if (adjustedY < 10) adjustedY = 10;
        }
    });
</script>

<!-- Backdrop to catch clicks outside -->
<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="citation-backdrop" onclick={onClose}></div>

<div 
    bind:this={popupElement}
    class="citation-popup well bg-white border p-0 overflow-hidden"
    style="left: {adjustedX}px; top: {adjustedY}px;"
>
    <div class="bg-light px-3 py-2 border-bottom d-flex justify-content-between align-items-center">
        <span class="fw-bold small" style="letter-spacing: 0.05em; font-size: 0.75rem; color: #568f27;">{title}</span>
        <button class="btn btn-link btn-sm p-0 text-muted" onclick={onClose}>
            <X size={14} />
        </button>
    </div>
    <div class="p-3 pb-2 citation-content">
        {@html content}
    </div>
    {#if sourceFile}
        <div class="px-3 pb-3 source-info">
            <div class="text-muted" style="font-size: 0.8rem; line-height: 1.4; word-break: break-all;">
                {sourceFile}
            </div>
            {#if formattedPages}
                <div class="text-muted fw-bold" style="font-size: 0.8rem; margin-top: 0.125rem;">
                    {formattedPages}
                </div>
            {/if}
        </div>
    {/if}
</div>

<style lang="scss">
    .citation-backdrop {
        position: fixed;
        top: 0;
        left: 0;
        width: 100vw;
        height: 100vh;
        z-index: 1000;
        background: transparent;
    }

    .citation-popup {
        position: fixed;
        z-index: 1001;
        width: 100%;
        max-width: 400px;
        min-width: 280px;
        margin: 0;
        pointer-events: auto;
        border-color: #e3e3e3 !important;
        // EXACT Kakimashou shadow from app.scss
        box-shadow: .3125rem .3125rem .625rem rgba(0, 0, 0, .5) !important;
    }

    .citation-content {
        font-size: 0.95rem;
        line-height: 1.5;
        color: #000;
        
        :global(p) {
            margin-bottom: 0;
        }

        :global(.footnote-back) {
            display: none;
        }
    }

    .source-info {
        line-height: 1.2;
    }
</style>
