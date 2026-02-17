<script lang="ts">
    import { onMount, tick, type Snippet } from 'svelte';
    import { fade } from 'svelte/transition';
    import { annotate } from 'rough-notation';

    interface Props {
        children: Snippet;
        show?: boolean;
        color?: string;
        strokeWidth?: number;
        animationDuration?: number;
        iterations?: number;
        padding?: number;
        multiline?: boolean;
        offsetY?: number;
        class?: string;
    }

    let { 
        children, 
        show = false, 
        color = '#fd7e1433', // Light semi-transparent version of the project's orange
        strokeWidth = 1.5,
        animationDuration = 600,
        iterations = 1,
        padding = 2,
        multiline = true,
        offsetY = 0,
        class: className = ''
    }: Props = $props();

    let containerElement: HTMLSpanElement | null = $state(null);
    let contentElement: HTMLSpanElement | null = $state(null);
    let annotation: any = null;
    let svgContent = $state('');
    let individualSvgs = $state<string[]>([]);
    let isShowing = $state(false);

    function createAnnotation() {
        if (!contentElement) return;
        
        // Cleanup previous
        if (annotation) {
            annotation.remove();
        }
        const existingSvg = contentElement.querySelector('svg');
        if (existingSvg) existingSvg.remove();

        annotation = annotate(contentElement, {
            type: 'highlight',
            color,
            strokeWidth,
            animationDuration,
            iterations,
            padding,
            multiline: true
        });

        // We show it temporarily to capture the SVG then hide/remove standard overlay
        annotation.show();

        // Small delay to ensure rough-notation has injected the SVG
        setTimeout(() => {
            const svg = contentElement?.querySelector('svg');
            if (svg) {
                let cloned = svg.cloneNode(true) as SVGElement;
                
                if (!multiline) {
                    const paths = Array.from(svg.querySelectorAll('path'));
                    if (paths.length > 0) {
                        const individualStrings: string[] = [];
                        paths.forEach(path => {
                            const indSvg = svg.cloneNode(false) as SVGElement;
                            indSvg.appendChild(path.cloneNode(true));
                            if (offsetY !== 0) indSvg.style.transform = `translateY(${offsetY}px)`;
                            individualStrings.push(indSvg.outerHTML);
                        });
                        individualSvgs = individualStrings;
                        svgContent = '';
                        svg.remove();
                        return;
                    }
                }

                if (offsetY !== 0) cloned.style.transform = `translateY(${offsetY}px)`;
                svgContent = cloned.outerHTML;
                individualSvgs = [];
                svg.remove();
            }
        }, 50);
    }

    $effect(() => {
        // Re-create when showing or when core props change
        if (show && contentElement) {
            isShowing = true;
            createAnnotation();
        } else if (!show) {
            // Delay clearing SVG to allow fade-out transition
            setTimeout(() => {
                svgContent = '';
                individualSvgs = [];
                isShowing = false;
            }, 200);
        }
    });

    onMount(() => {
        const resizeObserver = new ResizeObserver(() => {
            if (show) createAnnotation();
        });
        if (contentElement) resizeObserver.observe(contentElement);
        
        return () => {
            resizeObserver.disconnect();
            if (annotation) annotation.remove();
        };
    });
</script>

<span bind:this={containerElement} class="highlighter-container {className}">
    {#if isShowing}
        {#if multiline}
            {#if svgContent}
                <span class="highlight-svg-overlay" transition:fade={{ duration: 200 }}>
                    {@html svgContent}
                </span>
            {/if}
        {:else}
            {#each individualSvgs as svg, i}
                <span
                    class="highlight-svg-overlay individual"
                    style="animation-delay: {i * 150}ms; animation-duration: {animationDuration}ms"
                >
                    {@html svg}
                </span>
            {/each}
        {/if}
    {/if}
    <span bind:this={contentElement} class="highlighter-content">
        {@render children()}
    </span>
</span>

<style lang="scss">
    .highlighter-container {
        position: relative;
        display: inline-block;
    }

    .highlighter-content {
        position: relative;
        z-index: 1;
    }

    .highlight-svg-overlay {
        position: absolute;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        pointer-events: none;
        z-index: 0;
        display: block;

        :global(svg) {
            width: 100%;
            height: 100%;
            display: block;
        }

        &.individual {
            animation: highlight-fade-in ease-out both;
        }
    }

    @keyframes highlight-fade-in {
        0% {
            opacity: 0;
            transform: scale(0.95);
        }
        50% {
            opacity: 1;
            transform: scale(1.02);
        }
        100% {
            opacity: 1;
            transform: scale(1);
        }
    }
</style>
