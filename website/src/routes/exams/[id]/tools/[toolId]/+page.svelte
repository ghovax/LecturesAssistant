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

    <div class="d-flex justify-content-between align-items-start mb-4">
        <div>
            <h1 class="characterHeading mb-1">{tool.title}</h1>
            <span class="badge bg-secondary text-uppercase">{tool.type}</span>
        </div>
        <div class="btn-group">
            <button class="btn btn-outline-primary dropdown-toggle" data-bs-toggle="dropdown">
                <FileDown size={18} class="me-1" /> Export
            </button>
            <ul class="dropdown-menu dropdown-menu-end">
                <li><button class="dropdown-item" onclick={() => handleExport('pdf')}>PDF Document</button></li>
                <li><button class="dropdown-item" onclick={() => handleExport('docx')}>Word Document</button></li>
                <li><button class="dropdown-item" onclick={() => handleExport('md')}>Markdown</button></li>
            </ul>
        </div>
    </div>

    <div class="well bg-white">
        {#if tool.type === 'guide'}
            <div class="prose">
                {@html htmlContent}
            </div>
        {:else if tool.type === 'flashcard'}
            <div class="row g-4">
                {#each htmlContent as card}
                    <div class="col-md-6">
                        <div class="card h-100 shadow-sm">
                            <div class="card-header bg-light small fw-bold">FRONT</div>
                            <div class="card-body">{@html card.front_html}</div>
                            <div class="card-header bg-light border-top small fw-bold">BACK</div>
                            <div class="card-body">{@html card.back_html}</div>
                        </div>
                    </div>
                {/each}
            </div>
        {:else if tool.type === 'quiz'}
            <div class="quiz-list">
                {#each htmlContent as item, i}
                    <div class="mb-5 pb-4 border-bottom">
                        <h5>Question {i + 1}</h5>
                        <div class="mb-3">{@html item.question_html}</div>
                        <div class="list-group mb-3">
                            {#each item.options_html as opt}
                                <div class="list-group-item">{@html opt}</div>
                            {/each}
                        </div>
                        <div class="alert alert-success py-2">
                            <strong>Correct:</strong> {@html item.correct_answer_html}
                        </div>
                        <div class="small text-muted">
                            <strong>Explanation:</strong> {@html item.explanation_html}
                        </div>
                    </div>
                {/each}
            </div>
        {/if}
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
