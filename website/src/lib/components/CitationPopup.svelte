<script lang="ts">
    import { X } from 'lucide-svelte';

    interface Props {
        content: string;
        title?: string;
        onClose: () => void;
        x: number;
        y: number;
    }

    let { content, title = 'Citation', onClose, x, y }: Props = $props();

    // Ensure the popup stays within viewport
    let popupElement: HTMLDivElement | null = $state(null);
    let adjustedX = $state(x + 10); // Offset from cursor
    let adjustedY = $state(y + 10);

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
        <span class="fw-bold small text-uppercase" style="letter-spacing: 0.05em; font-size: 0.75rem; color: #568f27;">{title}</span>
        <button class="btn btn-link btn-sm p-0 text-muted" onclick={onClose}>
            <X size={14} />
        </button>
    </div>
    <div class="p-3 citation-content">
        {@html content}
    </div>
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
</style>
