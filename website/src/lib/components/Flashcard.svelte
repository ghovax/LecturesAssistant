<script lang="ts">
    interface Props {
        frontHTML: string;
        backHTML: string;
    }

    let { frontHTML, backHTML }: Props = $props();
    let isFlipped = $state(false);
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="flashcard-container" onclick={() => isFlipped = !isFlipped}>
    <div class="flashcard {isFlipped ? 'flipped' : ''}">
        <!-- Front -->
        <div class="flashcard-face front">
            <div class="tile-like bg-white p-4 d-flex align-items-center justify-content-center text-center h-100">
                <div class="content">{@html frontHTML}</div>
            </div>
        </div>

        <!-- Back -->
        <div class="flashcard-face back">
            <div class="tile-like bg-white p-4 d-flex align-items-center justify-content-center text-center h-100 border-top-orange">
                <div class="content">{@html backHTML}</div>
            </div>
        </div>
    </div>
</div>

<style lang="scss">
    .flashcard-container {
        perspective: 1000px;
        height: 200px; /* Slightly taller */
        cursor: pointer;
        margin-bottom: 1.5rem;
    }

    .flashcard {
        position: relative;
        width: 100%;
        height: 100%;
        transition: transform 0.3s;
        transform-style: preserve-3d;
        
        &.flipped {
            transform: rotateY(180deg);
        }
    }

    .flashcard-face {
        position: absolute;
        width: 100%;
        height: 100%;
        backface-visibility: hidden;
        -webkit-backface-visibility: hidden;
        
        &.back {
            transform: rotateY(180deg);
        }
    }

    .tile-like {
        background: #fff;
        box-shadow: 0 2px 8px rgba(0,0,0,0.05);
        color: var(--gray-900);
        border: 1px solid var(--gray-300);
        font-family: var(--font-primary);
        font-size: 1.05rem;
        line-height: 1.4;
        overflow-y: auto; /* Handle overflow */
        
        /* Hide scrollbar but allow scrolling */
        -ms-overflow-style: none;
        scrollbar-width: none;
        &::-webkit-scrollbar {
            display: none;
        }
    }

    .border-top-orange {
        border-top: 4px solid var(--orange) !important;
    }

    .content {
        padding: 5px;
        :global(p) {
            margin-bottom: 0;
        }
    }
</style>
