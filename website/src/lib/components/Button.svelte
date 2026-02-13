<script lang="ts">
    import type { Snippet } from 'svelte';

    interface Props {
        type?: 'button' | 'submit' | 'reset';
        variant?: 'primary' | 'success' | 'danger' | 'white' | 'outline-primary' | 'outline-success' | 'link';
        size?: 'sm' | 'md' | 'lg';
        onclick?: (e: MouseEvent) => void;
        disabled?: boolean;
        class?: string;
        children: Snippet;
        href?: string;
    }

    let { 
        type = 'button', 
        variant = 'primary', 
        size = 'md', 
        onclick, 
        disabled = false, 
        class: className = '', 
        children,
        href
    }: Props = $props();

    const sizeClass = size === 'sm' ? 'btn-sm' : (size === 'lg' ? 'btn-lg' : '');
    const variantClass = `btn-${variant}`;
</script>

{#if href}
    <a 
        {href} 
        class="cozy-btn {variantClass} {sizeClass} {className}" 
        class:disabled
        onclick={(e) => !disabled && onclick?.(e)}
    >
        {@render children()}
    </a>
{:else}
    <button 
        {type} 
        class="cozy-btn {variantClass} {sizeClass} {className}" 
        {disabled} 
        {onclick}
    >
        {@render children()}
    </button>
{/if}

<style lang="scss">
    .cozy-btn {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
        font-family: 'Manrope', sans-serif;
        font-weight: 600;
        font-size: 0.75rem;
        padding: 0 1.25rem;
        height: 2.5rem;
        border-radius: 0;
        border: 1px solid transparent;
        transition: all 0.2s ease;
        cursor: pointer;
        text-decoration: none;
        line-height: 1;
        text-transform: uppercase;
        letter-spacing: 0.04em;

        &.disabled {
            opacity: 0.5;
            cursor: not-allowed;
            pointer-events: none;
        }

        &.btn-sm {
            height: 1.75rem;
            padding: 0 0.75rem;
            font-size: 0.65rem;
        }

        &.btn-lg {
            height: 3rem;
            padding: 0 2rem;
            font-size: 0.85rem;
        }

        &.btn-primary, &.btn-success {
            background: var(--gray-900);
            color: var(--cream);
            border-color: var(--gray-900);

            &:hover {
                background: var(--gray-800);
                border-color: var(--gray-800);
            }
        }

        &.btn-danger {
            background: #ef4444;
            color: white;
            border-color: #ef4444;
            &:hover {
                background: #dc2626;
            }
        }

        &.btn-white {
            background: white;
            color: var(--gray-800);
            border-color: var(--gray-300);
            &:hover {
                background: var(--gray-200);
                border-color: var(--gray-400);
            }
        }

        &.btn-outline-primary, &.btn-outline-success {
            background: transparent;
            color: var(--gray-800);
            border-color: var(--gray-300);
            &:hover {
                background: var(--gray-200);
                border-color: var(--gray-400);
            }
        }

        &.btn-link {
            background: transparent;
            color: var(--gray-600);
            padding: 0;
            border: none;
            &:hover {
                color: var(--orange);
                text-decoration: underline;
            }
        }
    }
</style>
