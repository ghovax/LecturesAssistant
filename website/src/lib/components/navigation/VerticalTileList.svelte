<script lang="ts">
    import type { Snippet } from 'svelte';
    import ActionTile from './ActionTile.svelte';

    interface Props {
        children: Snippet;
        direction?: 'horizontal' | 'vertical';
        class?: string;
    }

    let { direction = 'horizontal', children, class: className = '' }: Props = $props();
</script>

<div class="vertical-tile-list {className}" class:vertical={direction === 'vertical'}>
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

                .action-tile-link,
                .action-tile-button {
                    width: 100%;
                }
            }
        }

        :global(.action-tile) {
            width: 250px;
        }
    }
</style>
