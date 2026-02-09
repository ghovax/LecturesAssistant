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
            } else {
                htmlContent = htmlRes.content; // Array for flash/quiz
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
            notifications.success(`We are preparing your export. You can see the progress in the source lecture.`);
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

    <div class="d-flex justify-content-between align-items-start mb-3">
        <div>
            <h2 class="mb-1">{tool.title}</h2>
            <span class="badge bg-dark">{capitalize(tool.type)} Kit</span>
        </div>
        <div class="btn-group">
            <button class="btn btn-primary dropdown-toggle" data-bs-toggle="dropdown">
                <span class="glyphicon me-1"><Download size={16} /></span> Export
            </button>
            <ul class="dropdown-menu dropdown-menu-end">
                <li><button class="dropdown-item" onclick={() => handleExport('pdf')}>PDF Document</button></li>
                <li><button class="dropdown-item" onclick={() => handleExport('docx')}>Word Document</button></li>
                <li><button class="dropdown-item" onclick={() => handleExport('md')}>Markdown Source</button></li>
            </ul>
        </div>
    </div>

    <div class="container-fluid p-0">
        <div class="row">
            <!-- Sidebar: Details & Navigation -->
            <div class="col-lg-3 col-md-4 order-md-2">
                <h3>Source Lecture</h3>
                <div class="well small mb-4">
                    <p>This study kit was generated from your lecture materials. You can find related tools and documents in the source lecture page.</p>
                    {#if tool.lecture_id}
                        <a href="/exams/{examId}/lectures/{tool.lecture_id}" class="btn btn-outline-primary btn-sm w-100">
                            Back to Lecture
                        </a>
                    {/if}
                </div>
            </div>

            <!-- Main Content: Tool Content -->
            <div class="col-lg-9 col-md-8 order-md-1">
                <div class="well bg-white p-4 shadow-sm border mb-5">
                    {#if tool.type === 'guide'}
                        <!-- svelte-ignore a11y_click_events_have_key_events -->
                        <!-- svelte-ignore a11y_no_static_element_interactions -->
                        <div 
                            class="prose" 
                            bind:this={proseContainer}
                            onclick={handleProseClick}
                        >
                            {@html htmlContent}
                        </div>
                    {:else if tool.type === 'flashcard'}
                        <div class="row g-4">
                            {#each htmlContent as card}
                                <div class="col-12">
                                    <div class="well bg-light border-start border-4 border-primary p-0 overflow-hidden shadow-none mb-3">
                                        <div class="px-3 py-2 bg-dark text-white small fw-bold">Front</div>
                                        <div class="p-3 bg-white wordBriefTitle" style="font-size: 1.1rem;">{@html card.front_html}</div>
                                        <div class="px-3 py-2 bg-secondary text-white small fw-bold border-top">Back</div>
                                        <div class="p-3 bg-white wordBriefContent">{@html card.back_html}</div>
                                    </div>
                                </div>
                            {/each}
                        </div>
                    {:else if tool.type === 'quiz'}
                        <div class="quiz-list">
                            {#each htmlContent as item, i}
                                <div class="well bg-white mb-4 p-4 border shadow-none">
                                    <h4 class="border-bottom pb-2 mb-3">Question {i + 1}</h4>
                                    <div class="mb-4 fs-5 fw-bold">{@html item.question_html}</div>
                                    
                                    <div class="list-group mb-4 shadow-sm">
                                        {#each item.options_html as opt}
                                            <div class="list-group-item py-3">{@html opt}</div>
                                        {/each}
                                    </div>
                                    
                                    <div class="well bg-success bg-opacity-10 border-success mb-3 p-3">
                                        <strong class="text-success small d-block mb-1">Correct Answer</strong>
                                        <div class="fs-6 fw-bold">{@html item.correct_answer_html}</div>
                                    </div>
                                    
                                    <div class="well bg-light border-0 m-0 p-3 small">
                                        <strong class="text-muted d-block mb-1">Explanation</strong>
                                        <div class="text-muted" style="line-height: 1.5;">{@html item.explanation_html}</div>
                                    </div>
                                </div>
                            {/each}
                        </div>
                    {/if}
                </div>
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
        <style>
        .prose :global(h2) { font-size: 1.5rem; margin-top: 2rem; border-bottom: 1px solid #eee; padding-bottom: 0.5rem; color: #2c4529; }
        .prose :global(h3) { font-size: 1.2rem; margin-top: 1.5rem; color: #555; }
        .prose :global(p) { line-height: 1.6; margin-bottom: 1rem; font-size: 1rem; }
        .prose :global(ul) { margin-bottom: 1rem; }
        .prose :global(li) { margin-bottom: 0.5rem; }
    
        /* Hide default footnotes section since we use popups */
        .prose :global(.footnotes) {
            display: none;
        }
    
            .prose :global(.footnote-ref) {
                text-decoration: none;
                font-weight: bold;
                color: #568f27;
                padding: 0 0.125rem;
                transition: all 0.15s ease;
            }
        
            .prose :global(.footnote-ref:hover) {
                background-color: #568f27;
                color: #fff !important;
                text-decoration: none;
            }
        </style>
        