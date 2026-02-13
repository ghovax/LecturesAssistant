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
        class="btn {variantClass} {sizeClass} {className}" 
        class:disabled
        onclick={(e) => !disabled && onclick?.(e)}
    >
        {@render children()}
    </a>
{:else}
    <button 
        {type} 
        class="btn {variantClass} {sizeClass} {className}" 
        {disabled} 
        {onclick}
    >
        {@render children()}
    </button>
{/if}
