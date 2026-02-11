<script lang="ts">
    import { Clock, FileText, Hash } from 'lucide-svelte';

    interface Props {
        type: 'page' | 'time' | 'count';
        current: number | string;
        total?: number | string;
        label?: string;
        class?: string;
    }

    let { type, current, total, label, class: className = '' }: Props = $props();
</script>

<div class="status-indicator {className}">
    <span class="indicator-icon">
        {#if type === 'page'}
            <FileText size={14} />
        {:else if type === 'time'}
            <Clock size={14} />
        {:else}
            <Hash size={14} />
        {/if}
    </span>
    
    <span class="indicator-text">
        {#if label}{label} {/if}
        <span class="current">{current}</span>
        {#if total}
            <span class="separator">of</span>
            <span class="total">{total}</span>
        {/if}
    </span>
</div>

<style lang="scss">
    .status-indicator {
        display: inline-flex;
        align-items: center;
        gap: 0.5rem;
        padding: 0.25rem 0.75rem;
        background-color: #f8f9fa;
        border: 1px solid #dee2e6;
        font-size: 0.75rem;
        font-weight: 700;
        color: #495057;
        letter-spacing: 0.02em;
        text-transform: uppercase;
        height: 1.75rem;
    }

    .indicator-icon {
        display: flex;
        align-items: center;
        color: #568f27;
    }

    .indicator-text {
        display: flex;
        align-items: center;
        gap: 0.25rem;
    }

    .current {
        color: #000;
    }

    .separator {
        color: #6c757d;
        font-weight: 400;
        text-transform: lowercase;
        font-style: italic;
        padding: 0 0.125rem;
    }

    .total {
        color: #6c757d;
    }

    :global(.transcript-nav) .status-indicator {
        background-color: #fff;
        border-color: #5cb85c;
    }

    :global(.document-nav) .status-indicator {
        background-color: #fff;
        border-color: #007bff;
    }
</style>
