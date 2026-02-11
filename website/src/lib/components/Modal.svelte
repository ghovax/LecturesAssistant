<script lang="ts">
    import { X } from 'lucide-svelte';
    import type { Snippet } from 'svelte';

    interface Props {
        title: string;
        glyph: string;
        isOpen: boolean;
        onClose: () => void;
        children: Snippet;
        footer?: Snippet;
        maxWidth?: string;
    }

    let { title, glyph, isOpen, onClose, children, footer, maxWidth = '500px' }: Props = $props();

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
        <div class="modal-dialog modal-dialog-centered" style="max-width: {maxWidth};">
            <div class="modal-content border-0 rounded-0 shadow-kakimashou">
                <!-- Header -->
                <div class="px-4 py-3 border-bottom d-flex justify-content-between align-items-center bg-white">
                    <div class="d-flex align-items-center gap-2">
                        <span class="glyph-icon" lang="ja">{glyph}</span>
                        <span class="fw-bold modal-title-text">{title}</span>
                    </div>
                    <button class="btn btn-link btn-sm text-muted p-0 d-flex align-items-center shadow-none border-0" onclick={onClose}>
                        <X size={20} />
                    </button>
                </div>

                <!-- Body -->
                <div class="modal-body p-4 bg-light">
                    {@render children()}
                </div>

                <!-- Footer -->
                {#if footer}
                    <div class="px-4 py-3 bg-white border-top">
                        {@render footer()}
                    </div>
                {/if}
            </div>
        </div>
    </div>
{/if}

<style lang="scss">
    .shadow-kakimashou {
        box-shadow: .3125rem .3125rem .625rem rgba(0, 0, 0, .5) !important;
    }

    .glyph-icon {
        font-size: 1.25rem;
        color: #568f27;
        font-family: "ＭＳ Ｐゴシック", "MS PGothic", "メイリオ", Meiryo, sans-serif;
    }

    .modal-title-text {
        letter-spacing: 0.02em;
        font-size: 1rem;
        text-transform: uppercase;
        color: #333;
    }

    .modal-content {
        border-radius: 0 !important;
    }

    :global(.modal-body) {
        .form-label {
            font-size: 0.75rem;
            font-weight: 700;
            color: #6c757d;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            margin-bottom: 0.5rem;
            display: block;
        }
    }
</style>
