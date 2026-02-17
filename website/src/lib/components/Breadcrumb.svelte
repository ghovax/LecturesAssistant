<script lang="ts">
    interface Item {
        label: string;
        href?: string;
        active?: boolean;
        onclick?: () => void;
    }

    interface Props {
        items: Item[];
    }

    let { items }: Props = $props();
</script>

<nav aria-label="breadcrumb">
    <ol class="breadcrumb">
        {#if items.length === 0}
            <li class="breadcrumb-item active">Home</li>
        {:else}
            <li class="breadcrumb-item"><a href="/">Home</a></li>
            {#each items as item}
                {#if item.active}
                    <li class="breadcrumb-item active" aria-current="page">{item.label}</li>
                {:else}
                    <li class="breadcrumb-item">
                        <a href={item.href} onclick={item.onclick}>{item.label}</a>
                    </li>
                {/if}
            {/each}
        {/if}
    </ol>
</nav>

<style lang="scss">
    .breadcrumb {
        font-family: var(--font-primary);
        font-size: 0.7rem;
        font-weight: 500;
        text-transform: uppercase;
        letter-spacing: 0.08em;
        padding: 0;
        margin-top: 1.5rem;
        margin-bottom: 1.5rem;
        background: transparent;
        
        .breadcrumb-item {
            color: var(--gray-500);
            display: flex;
            align-items: center;

            & + .breadcrumb-item::before {
                content: "/";
                color: var(--gray-300);
                padding: 0 10px;
                font-size: 0.85rem;
            }

            a {
                color: var(--gray-600);
                text-decoration: none;
                transition: color 0.2s ease;

                &:hover {
                    color: var(--orange);
                }
            }

            &.active {
                color: var(--gray-900);
                font-weight: 600;
            }
        }
    }
</style>
