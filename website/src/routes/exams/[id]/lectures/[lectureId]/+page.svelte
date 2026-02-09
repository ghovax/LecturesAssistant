<script lang="ts">
    import { onMount } from 'svelte';
    import { page } from '$app/state';
    import { api } from '$lib/api/client';
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';
    import Tile from '$lib/components/Tile.svelte';
    import { FileText, Clock, PlayCircle, Settings2 } from 'lucide-svelte';

    let { id: examId, lectureId } = $derived(page.params);
    let exam = $state<any>(null);
    let lecture = $state<any>(null);
    let transcript = $state<any>(null);
    let documents = $state<any[]>([]);
    let loading = $state(true);

    function formatTime(ms: number) {
        const totalSeconds = Math.floor(ms / 1000);
        const minutes = Math.floor(totalSeconds / 60);
        const seconds = totalSeconds % 60;
        return `${minutes}:${seconds.toString().padStart(2, '0')}`;
    }

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

    <div class="d-flex justify-content-between align-items-center mb-3">
        <h2>{lecture.title}</h2>
        <div class="btn-group">
            <button class="btn btn-primary dropdown-toggle" data-bs-toggle="dropdown">
                <span class="glyphicon me-1"><Settings2 size={16} /></span> Create Study Kit
            </button>
            <ul class="dropdown-menu dropdown-menu-end">
                <li><button class="dropdown-item" onclick={() => createTool('guide')}>Study Guide</button></li>
                <li><button class="dropdown-item" onclick={() => createTool('flashcard')}>Flashcards</button></li>
                <li><button class="dropdown-item" onclick={() => createTool('quiz')}>Practice Quiz</button></li>
            </ul>
        </div>
    </div>

    <div class="container-fluid p-0">
        <div class="row">
            <!-- Sidebar / Meta & Materials -->
            <div class="col-lg-3 col-md-4 order-md-2">
                <h3>Study Materials</h3>
                {#if documents.length === 0}
                    <div class="well text-center p-3 mb-4">
                        <p class="small text-muted m-0">No reference files linked.</p>
                    </div>
                {:else}
                    <div class="linkTiles tileSizeMd mb-4">
                        {#each documents as doc}
                            <Tile href="/exams/{examId}/lectures/{lectureId}/documents/{doc.id}" 
                                  icon="資" 
                                  title={doc.title}>
                                {#snippet description()}
                                    {doc.page_count} pages • 
                                    {#if doc.extraction_status === 'completed'}
                                        Ready to study
                                    {:else if doc.extraction_status === 'failed'}
                                        Processing error
                                    {:else}
                                        Preparing...
                                    {/if}
                                {/snippet}
                            </Tile>
                        {/each}
                    </div>
                {/if}

                <h3>Lecture Metadata</h3>
                <div class="well">
                    <table class="table table-sm table-borderless m-0">
                        <tbody>
                            <tr>
                                <td style="width: 40%"><strong>Status</strong></td>
                                <td>
                                    <span class="badge {lecture.status === 'ready' ? 'bg-success' : (lecture.status === 'failed' ? 'bg-danger' : 'bg-primary')}">
                                        {#if lecture.status === 'ready'}
                                            Ready to study
                                        {:else if lecture.status === 'failed'}
                                            Error
                                        {:else}
                                            Preparing...
                                        {/if}
                                    </span>
                                </td>
                            </tr>
                            <tr>
                                <td><strong>Date</strong></td>
                                <td>{new Date(lecture.created_at).toLocaleDateString()}</td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </div>

            <!-- Main Content / Transcript & Notes -->
            <div class="col-lg-9 col-md-8 order-md-1">
                <h3>Lesson Notes</h3>
                <p class="text-muted mb-3">{lecture.description || 'Comprehensive learning materials from this lecture recording.'}</p>

                <div class="well bg-white transcript-view mb-5">
                    {#if transcript && transcript.segments}
                        {#each transcript.segments as seg}
                            <div class="segment mb-3 d-flex">
                                <div class="time small text-muted me-3 mt-1 fw-bold" style="min-width: 85px; white-space: nowrap; font-family: monospace;">
                                    {formatTime(seg.start_millisecond)} - {formatTime(seg.end_millisecond)}
                                </div>
                                <div class="text">{@html seg.text_html}</div>
                            </div>
                        {/each}
                    {:else}
                        <div class="text-center p-5">
                            <div class="village-spinner mx-auto mb-3"></div>
                            <p>Our AI is meticulously preparing your lecture notes. This may take a few minutes...</p>
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

<style>
    .transcript-view {
        max-height: 600px;
        overflow-y: auto;
        border: 1px solid #dee2e6;
    }
    .segment .text {
        line-height: 1.5;
    }
</style>
