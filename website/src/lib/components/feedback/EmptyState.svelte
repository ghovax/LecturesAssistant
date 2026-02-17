<script lang="ts">
    import type { Snippet } from 'svelte';
    import type { ComponentType, SvelteComponent } from 'svelte';
    import { FileText } from 'lucide-svelte';

    interface Props {
        icon?: ComponentType<any>;
        iconSize?: number;
        title: string;
        description?: string;
        action?: Snippet;
        class?: string;
    }

    let {
        icon: Icon = FileText,
        iconSize = 48,
        title,
        description,
        action,
        class: className = ''
    }: Props = $props();
</script>

<div class="empty-state {className}">
    <div class="empty-state-icon">
        <Icon size={iconSize} />
    </div>
    <h3 class="empty-state-title">{title}</h3>
    {#if description}
        <p class="empty-state-description">{description}</p>
    {/if}
    {#if action}
        <div class="empty-state-action">
            {@render action()}
        </div>
    {/if}
</div>

<style lang="scss">
    .empty-state {
        padding: 5rem 1.5rem;
        text-align: center;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        width: 100%;
    }

    .empty-state-icon {
        display: flex;
        align-items: center;
        justify-content: center;
        margin-bottom: 1.5rem;
        color: var(--gray-400);
        opacity: 0.25;
    }

    .empty-state-title {
        font-size: 1rem;
        font-weight: 600;
        color: var(--gray-900);
        margin: 0 0 0.5rem 0;
        line-height: 1.2;
    }

    .empty-state-description {
        font-size: 0.85rem;
        color: var(--gray-500);
        line-height: 1.5;
        margin: 0 0 1.5rem 0;
        max-width: 400px;
    }

    .empty-state-action {
        display: flex;
        gap: 0.75rem;
    }
</style>
