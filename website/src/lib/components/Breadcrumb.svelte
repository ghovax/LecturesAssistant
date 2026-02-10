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
    </ol>
</nav>