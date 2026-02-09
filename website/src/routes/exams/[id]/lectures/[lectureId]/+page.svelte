<script lang="ts">
    import { onMount, onDestroy } from 'svelte';
    import { page } from '$app/state';
    import { browser } from '$app/environment';
    import { api } from '$lib/api/client';
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';
    import Tile from '$lib/components/Tile.svelte';
    import { FileText, Clock, PlayCircle, Settings2, ChevronLeft, ChevronRight, List, Volume2 } from 'lucide-svelte';

    let { id: examId, lectureId } = $derived(page.params);
    let exam = $state<any>(null);
    let lecture = $state<any>(null);
    let transcript = $state<any>(null);
    let documents = $state<any[]>([]);
    let loading = $state(true);
    let currentSegmentIndex = $state(0);
    let audioElement: HTMLAudioElement | null = $state(null);

    function formatTime(ms: number) {
        const totalSeconds = Math.floor(ms / 1000);
        const minutes = Math.floor(totalSeconds / 60);
        const seconds = totalSeconds % 60;
        return `${minutes}:${seconds.toString().padStart(2, '0')}`;
    }

    async function loadLecture() {
        loading = true;
        try {
            const [examR, lectureR, transcriptR, docsR] = await Promise.all([
                api.getExam(examId),
                api.getLecture(lectureId, examId),
                api.request('GET', `/transcripts/html?lecture_id=${lectureId}`),
                api.listDocuments(lectureId)
            ]);
            exam = examR;
            lecture = lectureR;
            transcript = transcriptR;
            documents = docsR ?? [];
        } catch (e) {
            console.error(e);
        } finally {
            loading = false;
        }
    }

    function nextSegment() {
        if (transcript?.segments && currentSegmentIndex < transcript.segments.length - 1) {
            currentSegmentIndex++;
        }
    }

    function prevSegment() {
        if (currentSegmentIndex > 0) {
            currentSegmentIndex--;
        }
    }

    function handleKeyDown(event: KeyboardEvent) {
        if (event.target instanceof HTMLInputElement || event.target instanceof HTMLTextAreaElement) return;
        
        if (event.key === 'ArrowRight') {
            nextSegment();
        } else if (event.key === 'ArrowLeft') {
            prevSegment();
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

    $effect(() => {
        // When segment changes, reload audio if needed
        if (audioElement && transcript?.segments[currentSegmentIndex]) {
            audioElement.load();
        }
    });

    onMount(() => {
        loadLecture();
        if (browser) {
            window.addEventListener('keydown', handleKeyDown);
        }
    });

    onDestroy(() => {
        if (browser) {
            window.removeEventListener('keydown', handleKeyDown);
        }
    });
</script>

{#if lecture && exam}
    <Breadcrumb items={[
        { label: 'My Studies', href: '/exams' }, 
        { label: exam.title, href: `/exams/${examId}` }, 
        { label: lecture.title, active: true }
    ]} />

    <div class="d-flex justify-content-between align-items-start mb-3">
        <div>
            <h2 class="mb-1">{lecture.title}</h2>
            {#if transcript && transcript.segments}
                <span class="badge bg-dark">Segment {currentSegmentIndex + 1} of {transcript.segments.length}</span>
            {/if}
        </div>
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
            <!-- Sidebar: Study Materials & Transcript Nav -->
            <div class="col-lg-3 col-md-4 order-md-2">
                <h3>Study Materials</h3>
                <div class="linkTiles tileSizeMd mb-4">
                    {#each documents as doc}
                        <Tile href="/exams/{examId}/lectures/{lectureId}/documents/{doc.id}" 
                                icon="資" 
                                title={doc.title}>
                            {#snippet description()}
                                {doc.page_count} pages • {doc.extraction_status === 'completed' ? 'Ready to study' : 'Preparing...'}
                            {/snippet}
                        </Tile>
                    {/each}
                    {#if documents.length === 0}
                        <div class="well text-center p-3 small text-muted">No materials yet.</div>
                    {/if}
                </div>

                {#if transcript && transcript.segments}
                    <h3>Transcript Index</h3>
                    <div class="list-group shadow-sm small overflow-auto mb-4" style="max-height: 40vh;">
                        {#each transcript.segments as seg, i}
                            <button 
                                onclick={() => currentSegmentIndex = i} 
                                class="list-group-item list-group-item-action d-flex justify-content-between align-items-center text-start {currentSegmentIndex === i ? 'active' : ''}"
                            >
                                {formatTime(seg.start_millisecond)}
                                <span class={currentSegmentIndex === i ? 'text-white' : 'text-muted'}><Clock size={12} /></span>
                            </button>
                        {/each}
                    </div>
                {/if}

                <h3>Lecture Metadata</h3>
                <div class="well small">
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

            <!-- Main Content: Single Segment Transcript -->
            <div class="col-lg-9 col-md-8 order-md-1">
                <div class="mb-3">
                    <h3>Lesson Notes</h3>
                    <p class="text-muted mb-0">{lecture.description || 'Comprehensive learning materials from this lecture recording.'}</p>
                </div>

                {#if transcript && transcript.segments}
                    {@const seg = transcript.segments[currentSegmentIndex]}
                    <div class="well bg-white p-0 overflow-hidden mb-5 border shadow-sm">
                        <div class="bg-light px-4 py-2 border-bottom d-flex justify-content-between align-items-center">
                            <span class="fw-bold small text-uppercase">
                                {formatTime(seg.start_millisecond)} &ndash; {formatTime(seg.end_millisecond)}
                            </span>
                            <div class="btn-group">
                                <button 
                                    class="btn btn-link btn-sm text-dark p-0 me-2 shadow-none" 
                                    disabled={currentSegmentIndex === 0}
                                    onclick={prevSegment}
                                    title="Previous Segment (Left Arrow)"
                                >
                                    <ChevronLeft size={18} />
                                </button>
                                <button 
                                    class="btn btn-link btn-sm text-dark p-0 shadow-none" 
                                    disabled={currentSegmentIndex === transcript.segments.length - 1}
                                    onclick={nextSegment}
                                    title="Next Segment (Right Arrow)"
                                >
                                    <ChevronRight size={18} />
                                </button>
                            </div>
                        </div>
                        
                        <div class="p-4">
                            {#if seg.media_id}
                                <div class="mb-4 bg-light p-3 border d-flex align-items-center gap-3">
                                    <Volume2 size={24} class="text-primary flex-shrink-0" />
                                    <audio 
                                        bind:this={audioElement}
                                        controls 
                                        class="w-100" 
                                        style="height: 32px;"
                                    >
                                        <source 
                                            src="http://localhost:3000/api/media/content?media_id={seg.media_id}&session_token={browser ? localStorage.getItem('session_token') : ''}#t={seg.original_start_milliseconds / 1000},{seg.original_end_milliseconds / 1000}" 
                                            type="audio/mpeg"
                                        />
                                        Your browser does not support the audio element.
                                    </audio>
                                </div>
                            {/if}

                            <div class="transcript-text" style="font-size: 1rem; line-height: 1.6;">
                                {@html seg.text_html}
                            </div>
                        </div>
                    </div>
                    <div class="text-center text-muted small mt-n4 mb-5">
                        <kbd>←</kbd> <kbd>→</kbd> Use arrow keys to navigate segments
                    </div>
                {:else}
                    <div class="well bg-white text-center p-5 mb-5 border shadow-sm">
                        <div class="village-spinner mx-auto mb-3"></div>
                        <p>Our AI is meticulously preparing your lecture notes. This may take a few minutes...</p>
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

<style>
    .transcript-text {
        color: #333;
    }

    .list-group-item.active {
        background-color: #568f27;
        border-color: #568f27;
    }

    kbd {
        background-color: #eee;
        border-radius: 3px;
        border: 1px solid #b4b4b4;
        box-shadow: 0 1px 1px rgba(0,0,0,.2),0 2px 0 0 rgba(255,255,255,.7) inset;
        color: #333;
        display: inline-block;
        font-size: .85em;
        font-weight: 700;
        line-height: 1;
        padding: 2px 4px;
        white-space: nowrap;
    }

    audio::-webkit-media-controls-enclosure {
        border-radius: 0;
        background-color: #f8f9fa;
    }
</style>
