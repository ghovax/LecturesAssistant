<script lang="ts">
    import { onMount } from 'svelte';
    import { page } from '$app/state';
    import { api } from '$lib/api/client';
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';
    import { FileText, Clock, PlayCircle, Settings2 } from 'lucide-svelte';

    let { id: examId, lectureId } = $derived(page.params);
    let exam = $state<any>(null);
    let lecture = $state<any>(null);
    let transcript = $state<any>(null);
    let documents = $state<any[]>([]);
    let loading = $state(true);

    async function loadLecture() {
        loading = true;
        try {
            [exam, lecture, transcript, documents] = await Promise.all([
                api.getExam(examId),
                api.getLecture(lectureId, examId),
                api.request('GET', `/transcripts/html?lecture_id=${lectureId}`),
                api.listDocuments(lectureId)
            ]);
        } catch (e) {
            console.error(e);
        } finally {
            loading = false;
        }
    }

    async function createTool(type: string) {
        try {
            await api.createTool({
                exam_id: examId,
                lecture_id: lectureId,
                type,
                length: 'medium'
            });
            alert('Generation job created! Check the Jobs section.');
        } catch (e) {
            alert(e);
        }
    }

    onMount(loadLecture);
</script>

{#if lecture && exam}
    <Breadcrumb items={[
        { label: 'My Studies', href: '/exams' }, 
        { label: exam.title, href: `/exams/${examId}` }, 
        { label: lecture.title, active: true }
    ]} />

    <div class="row mb-4">
        <div class="col-md-8">
            <h1 class="characterHeading mb-1">{lecture.title}</h1>
            <p class="text-muted">{lecture.description || 'Learn from this lesson below.'}</p>
        </div>
        <div class="col-md-4 text-end">
            <div class="btn-group">
                <button class="btn btn-success dropdown-toggle" data-bs-toggle="dropdown">
                    <Settings2 size={18} class="me-1" /> Create Study Kit
                </button>
                <ul class="dropdown-menu dropdown-menu-end">
                    <li><button class="dropdown-item" onclick={() => createTool('guide')}>Study Guide</button></li>
                    <li><button class="dropdown-item" onclick={() => createTool('flashcard')}>Flashcards</button></li>
                    <li><button class="dropdown-item" onclick={() => createTool('quiz')}>Practice Quiz</button></li>
                </ul>
            </div>
        </div>
    </div>

    <div class="row">
        <div class="col-md-8">
            <h3>Lesson Notes</h3>
            <div class="well bg-white transcript-view">
                {#if transcript && transcript.segments}
                    {#each transcript.segments as seg}
                        <div class="segment mb-3 d-flex">
                            <div class="time small text-muted me-3 mt-1">
                                {Math.floor(seg.start_millisecond / 60000)}:{(Math.floor(seg.start_millisecond / 1000) % 60).toString().padStart(2, '0')}
                            </div>
                            <div class="text">{@html seg.text_html}</div>
                        </div>
                    {/each}
                {:else}
                    <p class="text-center p-4">Your lesson notes are being prepared...</p>
                {/if}
            </div>
        </div>

        <div class="col-md-4">
            <h3>Study Materials</h3>
            {#if documents.length === 0}
                <div class="well small text-center">No slides or PDFs linked.</div>
            {:else}
                <div class="list-group shadow-sm">
                    {#each documents as doc}
                        <div class="list-group-item">
                            <div class="d-flex align-items-center">
                                <FileText size={20} class="text-primary me-2" />
                                <div class="text-truncate flex-grow-1">
                                    <strong>{doc.title}</strong>
                                    <div class="small text-muted">{doc.page_count} pages • {doc.extraction_status === 'completed' ? 'Ready' : 'Preparing...'}</div>
                                </div>
                            </div>
                        </div>
                    {/each}
                </div>
            {/if}
        </div>
    </div>
{:else if loading}
    <div class="text-center p-5">
        <div class="village-spinner mx-auto"></div>
    </div>
{/if}

<style>
    .transcript-view {
        max-height: 600px;
        overflow-y: auto;
        border: 1px solid #dee2e6;
    }
    .segment .time {
        min-width: 45px;
        font-family: monospace;
    }
    .segment .text {
        line-height: 1.5;
    }
</style>
