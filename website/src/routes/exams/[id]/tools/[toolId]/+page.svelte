<script lang="ts">
    import { onMount } from 'svelte';
    import { page } from '$app/state';
    import { api } from '$lib/api/client';
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';
    import { Download, FileDown, ExternalLink } from 'lucide-svelte';

    let { id: examId, toolId } = $derived(page.params);
    let exam = $state<any>(null);
    let tool = $state<any>(null);
    let htmlContent = $state('');
    let loading = $state(true);

    async function loadTool() {
        loading = true;
        try {
            exam = await api.getExam(examId);
            tool = await api.request('GET', `/tools/details?tool_id=${toolId}&exam_id=${examId}`);
            
            const htmlRes = await api.getToolHTML(toolId, examId);
            if (tool.type === 'guide') {
                htmlContent = htmlRes.content_html;
            } else {
                htmlContent = htmlRes.content; // Array for flash/quiz
            }
        } catch (e) {
            console.error(e);
        } finally {
            loading = false;
        }
    }

    async function handleExport(format: string) {
        try {
            const res = await api.exportTool({ tool_id: toolId, exam_id: examId, format });
            alert(`Export job created: ${res.job_id}. You can monitor progress in the Jobs section.`);
        } catch (e) {
            alert(e);
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

    <div class="container-fluid p-0">
        <div class="row">
            <div class="col-xl-8 col-lg-7 col-md-6">
                <h1 class="characterHeading mb-1">{tool.title}</h1>
                <div class="mb-4">
                    <span class="badge bg-secondary text-uppercase">{tool.type}</span>
                    <span class="ms-2 text-muted small">Generated on {new Date(tool.created_at).toLocaleDateString()}</span>
                </div>

                <div class="well bg-white p-4 shadow-sm">
                    {#if tool.type === 'guide'}
                        <div class="prose">
                            {@html htmlContent}
                        </div>
                    {:else if tool.type === 'flashcard'}
                        <div class="row g-4">
                            {#each htmlContent as card}
                                <div class="col-md-12">
                                    <div class="well bg-light border-start border-4 border-primary p-0 overflow-hidden">
                                        <div class="px-3 py-2 bg-dark text-white small fw-bold text-uppercase">Front</div>
                                        <div class="p-3 bg-white">{@html card.front_html}</div>
                                        <div class="px-3 py-2 bg-secondary text-white small fw-bold text-uppercase border-top">Back</div>
                                        <div class="p-3 bg-white">{@html card.back_html}</div>
                                    </div>
                                </div>
                            {/each}
                        </div>
                    {:else if tool.type === 'quiz'}
                        <div class="quiz-list">
                            {#each htmlContent as item, i}
                                <div class="well bg-white mb-4 p-4 border shadow-sm">
                                    <h4 class="border-bottom pb-2 mb-3">Question {i + 1}</h4>
                                    <div class="mb-4 fs-5">{@html item.question_html}</div>
                                    
                                    <div class="list-group mb-4">
                                        {#each item.options_html as opt}
                                            <div class="list-group-item py-3">{@html opt}</div>
                                        {/each}
                                    </div>
                                    
                                    <div class="well bg-success bg-opacity-10 border-success mb-3">
                                        <strong class="text-success text-uppercase small d-block mb-1">Correct Answer</strong>
                                        <div class="fs-6">{@html item.correct_answer_html}</div>
                                    </div>
                                    
                                    <div class="well bg-light border-0 m-0 p-3 small">
                                        <strong class="text-muted text-uppercase d-block mb-1">Explanation</strong>
                                        <div class="text-muted">{@html item.explanation_html}</div>
                                    </div>
                                </div>
                            {/each}
                        </div>
                    {/if}
                </div>
            </div>

            <div class="col-xl-4 col-lg-5 col-md-6">
                <h3>Export Tool</h3>
                <div class="well bg-light mb-4">
                    <p class="small text-muted">Generate a high-quality document for offline study or printing.</p>
                    <div class="d-grid gap-2">
                        <button class="btn btn-primary" onclick={() => handleExport('pdf')}>
                            <span class="glyphicon me-2"><FileDown size={16} /></span> Download PDF
                        </button>
                        <button class="btn btn-outline-secondary" onclick={() => handleExport('docx')}>
                            <span class="glyphicon me-2"><FileDown size={16} /></span> Word Document
                        </button>
                        <button class="btn btn-outline-secondary" onclick={() => handleExport('md')}>
                            <span class="glyphicon me-2"><FileDown size={16} /></span> Markdown Source
                        </button>
                    </div>
                </div>

                <h3>Study Details</h3>
                <div class="well bg-light">
                    <table class="table table-sm table-borderless m-0 small">
                        <tbody>
                            <tr>
                                <td><strong>Subject</strong></td>
                                <td>{exam.title}</td>
                            </tr>
                            <tr>
                                <td><strong>Type</strong></td>
                                <td class="text-uppercase">{tool.type}</td>
                            </tr>
                            <tr>
                                <td><strong>Language</strong></td>
                                <td>{tool.language_code || 'en-US'}</td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    </div>
{:else if loading}
    <div class="text-center p-5">
        <div class="village-spinner mx-auto"></div>
    </div>
{/if}

<style>
    .prose :global(h2) { font-size: 1.75rem; margin-top: 2rem; border-bottom: 1px solid #eee; }
    .prose :global(h3) { font-size: 1.25rem; margin-top: 1.5rem; color: #555; }
    .prose :global(p) { line-height: 1.6; margin-bottom: 1rem; }
    .prose :global(ul) { margin-bottom: 1rem; }
    .prose :global(li) { margin-bottom: 0.5rem; }
    
    .characterHeading {
        font-size: 2.5rem;
        font-weight: 300;
        border: none;
    }
</style>
