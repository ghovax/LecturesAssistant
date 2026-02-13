<script lang="ts">
    import { onMount } from 'svelte';
    import { page } from '$app/state';
    import { api } from '$lib/api/client';
    import { notifications } from '$lib/stores/notifications.svelte';
    import { getLanguageName, capitalize } from '$lib/utils';
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';
    import Tile from '$lib/components/Tile.svelte';
    import CitationPopup from '$lib/components/CitationPopup.svelte';
    import { Download, FileDown, ExternalLink, Clock, Settings2 } from 'lucide-svelte';

    let { id: examId, toolId } = $derived(page.params);
    let exam = $state<any>(null);
    let tool = $state<any>(null);
    let otherTools = $state<any[]>([]);
    let htmlContent = $state<any>(null);
    let citations = $state<any[]>([]);
    let loading = $state(true);

    // Citation Popup State
    let activeCitation = $state<{ content: string, x: number, y: number, sourceFile?: string, sourcePages?: number[] } | null>(null);
    let proseContainer: HTMLDivElement | null = $state(null);

    async function loadTool() {
        loading = true;
        try {
            const [examData, toolData, allTools, htmlRes] = await Promise.all([
                api.getExam(examId),
                api.request('GET', `/tools/details?tool_id=${toolId}&exam_id=${examId}`),
                api.listTools(examId),
                api.getToolHTML(toolId, examId)
            ]);
            
            exam = examData;
            tool = toolData;
            otherTools = (allTools ?? []).filter((t: any) => t.id !== toolId).slice(0, 3);
            
            if (tool.type === 'guide') {
                htmlContent = htmlRes.content_html;
                citations = htmlRes.citations ?? [];
            }
        } catch (e) {
            console.error(e);
        } finally {
            loading = false;
        }
    }

    function handleProseClick(event: MouseEvent) {
        const target = event.target as HTMLElement;
        const footnoteRef = target.closest('.footnote-ref');
        
        if (footnoteRef) {
            event.preventDefault();
            const href = footnoteRef.getAttribute('href');
            if (href && href.startsWith('#')) {
                const id = href.substring(1);
                // Extract number from id (usually fnN or similar)
                const numMatch = id.match(/\d+$/);
                const num = numMatch ? parseInt(numMatch[0]) : -1;
                const meta = citations.find(c => c.number === num);

                if (meta) {
                    activeCitation = {
                        content: meta.content_html,
                        x: event.clientX,
                        y: event.clientY,
                        sourceFile: meta.source_file,
                        sourcePages: meta.source_pages
                    };
                }
            }
        }
    }

    async function handleExport(format: string) {
        try {
            const res = await api.exportTool({ tool_id: toolId, exam_id: examId, format });
            notifications.success(`We are preparing your export. You can see the progress in the source lesson.`);
        } catch (e: any) {
            notifications.error(e.message || e);
        }
    }

    onMount(loadTool);
</script>

{#if tool && exam}
    <Breadcrumb items={[
        { label: 'My Studies', href: '/exams' }, 
        { label: exam.title, href: `/exams/${examId}` }, 
        { label: tool.title, active: true }
    ]} />

    <header class="page-header">
        <div class="d-flex justify-content-between align-items-center mb-2">
            <h1 class="page-title m-0">{tool.title}</h1>
            <div class="btn-group">
                <button class="btn btn-primary btn-sm dropdown-toggle rounded-0" data-bs-toggle="dropdown">
                    Export
                </button>
                <ul class="dropdown-menu dropdown-menu-end rounded-0 shadow-none border">
                    <li><button class="dropdown-item" onclick={() => handleExport('pdf')}>PDF Document</button></li>
                    <li><button class="dropdown-item" onclick={() => handleExport('docx')}>Word Document</button></li>
                    <li><button class="dropdown-item" onclick={() => handleExport('md')}>Markdown Source</button></li>
                </ul>
            </div>
        </div>
        <div class="d-flex align-items-center gap-2">
            <span class="badge bg-dark rounded-0">{capitalize(tool.type)} Material</span>
        </div>
    </header>

    <div class="container-fluid p-0">
        <div class="row g-4">
            <!-- Main Content: Tool Content -->
            <div class="col-12">
                {#if tool.type === 'guide'}
                    <div class="bg-white border mb-3">
                        <div class="standard-header">
                            <div class="header-title">
                                <span class="header-text">Study Guide</span>
                            </div>
                        </div>
                        <!-- svelte-ignore a11y_click_events_have_key_events -->
                        <!-- svelte-ignore a11y_no_static_element_interactions -->
                        <div 
                            class="p-4 prose" 
                            bind:this={proseContainer}
                            onclick={handleProseClick}
                        >
                            {@html htmlContent}
                        </div>
                    </div>
                {/if}
            </div>
        </div>
    </div>
{:else if loading}
    <div class="text-center p-5">
        <div class="village-spinner mx-auto"></div>
    </div>
{/if}

{#if activeCitation}
    <CitationPopup 
        content={activeCitation.content} 
        sourceFile={activeCitation.sourceFile}
        sourcePages={activeCitation.sourcePages}
        x={activeCitation.x} 
        y={activeCitation.y} 
        onClose={() => activeCitation = null} 
    />
{/if}

<style lang="scss">
    .prose :global(h2) { font-size: 1.25rem; margin-top: 2rem; border-bottom: 1px solid var(--gray-300); padding-bottom: 0.5rem; color: var(--gray-900); }
    .prose :global(h3) { font-size: 1.1rem; margin-top: 1.5rem; color: var(--gray-700); }
    .prose :global(p) { line-height: 1.6; margin-bottom: 1rem; font-size: 0.85rem; }
    .prose :global(ul) { margin-bottom: 1rem; }
    .prose :global(li) { margin-bottom: 0.5rem; }

    /* Hide default footnotes section since we use popups */
    .prose :global(.footnotes) {
        display: none;
    }

    .prose :global(.footnote-ref) {
        text-decoration: none;
        font-weight: bold;
        color: var(--orange);
        padding: 0 0.125rem;
        transition: all 0.15s ease;
    }

    .prose :global(.footnote-ref:hover) {
        background-color: var(--orange);
        color: #fff !important;
        text-decoration: none;
    }

    .linkTiles {
        display: grid;
        grid-template-columns: 1fr;
        gap: 0;
        background: transparent;
        
        :global(.tile-wrapper) {
            margin: 0;
            border: 1px solid var(--gray-300);
            width: 100%;
            
            :global(a), :global(button) {
                width: 100%;
            }
        }
    }

    .bg-orange {
        background-color: var(--orange) !important;
    }
    .border-orange {
        border-color: var(--orange) !important;
    }
</style>
        