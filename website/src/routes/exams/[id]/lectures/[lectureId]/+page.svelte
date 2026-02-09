<script lang="ts">
    import { onMount, onDestroy } from 'svelte';
    import { page } from '$app/state';
    import { browser } from '$app/environment';
    import { api } from '$lib/api/client';
    import { notifications } from '$lib/stores/notifications.svelte';
    import { capitalize, formatJobType } from '$lib/utils';
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';
    import Tile from '$lib/components/Tile.svelte';
    import CitationPopup from '$lib/components/CitationPopup.svelte';
    import { FileText, Clock, PlayCircle, Settings2, ChevronLeft, ChevronRight, List, Volume2, Activity, Loader2, CheckCircle2, XCircle, Play, AlertCircle } from 'lucide-svelte';

    let { id: examId, lectureId } = $derived(page.params);
    let exam = $state<any>(null);
    let lecture = $state<any>(null);
    let transcript = $state<any>(null);
    let documents = $state<any[]>([]);
    let tools = $state<any[]>([]);
    let guideTool = $derived(tools.find(t => t.type === 'guide'));
    let guideHTML = $state('');
    let activeJobs = $state<any[]>([]);
    let loading = $state(true);
    let currentSegmentIndex = $state(0);
    let audioElement: HTMLAudioElement | null = $state(null);
    let pollInterval: any;

    // Citation Popup State
    let activeCitation = $state<{ content: string, x: number, y: number, sourceFile?: string, sourcePages?: number[] } | null>(null);

    function formatTime(ms: number) {
        const totalSeconds = Math.floor(ms / 1000);
        const minutes = Math.floor(totalSeconds / 60);
        const seconds = totalSeconds % 60;
        return `${minutes}:${seconds.toString().padStart(2, '0')}`;
    }

    async function loadLecture() {
        loading = true;
        try {
            const [examR, lectureR, transcriptR, docsR, toolsR] = await Promise.all([
                api.getExam(examId),
                api.getLecture(lectureId, examId),
                api.request('GET', `/transcripts/html?lecture_id=${lectureId}`),
                api.listDocuments(lectureId),
                api.request('GET', `/tools?lecture_id=${lectureId}&exam_id=${examId}`)
            ]);
            exam = examR;
            lecture = lectureR;
            transcript = transcriptR;
            documents = docsR ?? [];
            tools = toolsR ?? [];
            
            if (guideTool) {
                const htmlRes = await api.getToolHTML(guideTool.id, examId);
                guideHTML = htmlRes.content_html;
            }

            await loadJobs();
        } catch (e) {
            console.error(e);
        } finally {
            loading = false;
        }
    }

    async function loadJobs() {
        try {
            const jobsData = await api.request('GET', `/jobs?lecture_id=${lectureId}`);
            activeJobs = jobsData ?? [];
        } catch (e) {
            console.error(e);
        }
    }

    async function handleCitationClick(event: MouseEvent) {
        const target = event.target as HTMLElement;
        const footnoteRef = target.closest('.footnote-ref');
        
        if (footnoteRef && guideTool) {
            event.preventDefault();
            const href = footnoteRef.getAttribute('href');
            if (href && href.startsWith('#')) {
                const id = href.substring(1);
                // Extract number from id
                const numMatch = id.match(/\d+$/);
                const num = numMatch ? parseInt(numMatch[0]) : -1;
                
                try {
                    // Fetch citations for this tool if not already loaded or just get from detail
                    const htmlRes = await api.getToolHTML(guideTool.id, examId);
                    const meta = htmlRes.citations?.find((c: any) => c.number === num);

                    if (meta) {
                        activeCitation = {
                            content: meta.content_html,
                            x: event.clientX,
                            y: event.clientY,
                            sourceFile: meta.source_file,
                            sourcePages: meta.source_pages
                        };
                    }
                } catch (e) {
                    console.error('Failed to load citation metadata', e);
                }
            }
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
            notifications.success('We are building your study kit. You can see the progress in the sidebar.');
        } catch (e: any) {
            notifications.error(e.message || e);
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
        pollInterval = setInterval(loadJobs, 3000);
        if (browser) {
            window.addEventListener('keydown', handleKeyDown);
        }
    });

    onDestroy(() => {
        clearInterval(pollInterval);
        if (browser) {
            window.removeEventListener('keydown', handleKeyDown);
        }
    });
</script>

{#if lecture && exam}
    <Breadcrumb items={[
        { label: exam.title, href: `/exams/${examId}` }, 
        { label: lecture.title, active: true }
    ]} />

    <div class="d-flex justify-content-between align-items-center mb-4">
        <h2 class="m-0">{lecture.title}</h2>
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
            <!-- Sidebar: Navigation & Materials -->
            <div class="col-lg-3 col-md-4 order-md-2">
                <h3>Dashboard</h3>
                <div class="linkTiles tileSizeMd mb-4">
                    {#if guideTool}
                        <Tile href="#study-guide" icon="案" title="Study Guide">
                            {#snippet description()}
                                Comprehensive summary of this lecture.
                            {/snippet}
                        </Tile>
                    {/if}

                    <Tile href="#lesson-notes" icon="講" title="Transcript">
                        {#snippet description()}
                            The complete lecture dialogue and notes.
                        {/snippet}
                    </Tile>

                    {#each tools.filter(t => t.type !== 'guide') as tool}
                        <Tile href="/exams/{examId}/tools/{tool.id}" 
                            icon={tool.type === 'flashcard' ? '札' : '問'} 
                            title={capitalize(tool.type)}>
                            {#snippet description()}
                                Practice your knowledge.
                            {/snippet}
                        </Tile>
                    {/each}
                </div>

                {#if activeJobs.some(j => j.status === 'RUNNING' || j.status === 'PENDING')}
                    <h3>Activity Progress</h3>
                    <div class="well bg-white p-0 mb-4 border shadow-sm overflow-hidden">
                        {#each activeJobs.filter(j => j.status === 'RUNNING' || j.status === 'PENDING') as job}
                            <div class="p-3 border-bottom last-child-border-0">
                                <div class="d-flex justify-content-between align-items-center mb-2">
                                    <span class="fw-bold small">
                                        {formatJobType(job.type)}
                                    </span>
                                    {#if job.status === 'RUNNING'}
                                        <Loader2 size={14} class="spin text-primary" />
                                    {:else}
                                        <Play size={14} class="text-muted" />
                                    {/if}
                                </div>
                                <div class="progress mb-2" style="height: 4px;">
                                    <div class="progress-bar" style="width: {job.progress}%"></div>
                                </div>
                                <div class="small text-muted text-truncate" style="font-size: 0.7rem;">
                                    {#if job.metadata?.document_title}
                                        Reading: {job.metadata.document_title}
                                    {:else if job.metadata?.media_index}
                                        Audio part {job.metadata.media_index} of {job.metadata.total_media}
                                    {:else}
                                        {job.progress_message_text || 'Waiting...'}
                                    {/if}
                                </div>
                            </div>
                        {/each}
                    </div>
                {/if}

                {#if documents.length > 0}
                    <h3>Study Materials</h3>
                    <div class="linkTiles tileSizeMd mb-4">
                        {#each documents as doc}
                            <Tile href="/exams/{examId}/lectures/{lectureId}/documents/{doc.id}" 
                                    icon="資" 
                                    title={doc.title}>
                                {#snippet description()}
                                    {doc.page_count} pages • {doc.extraction_status === 'completed' ? 'Ready' : 'Reading...'}
                                {/snippet}
                            </Tile>
                        {/each}
                    </div>
                {/if}

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
            </div>

            <!-- Main Content: Single Segment Transcript -->
            <div class="col-lg-9 col-md-8 order-md-1">
                {#if guideHTML}
                    <div class="mb-5" id="study-guide">
                        <div class="border-bottom pb-2 mb-4">
                            <h3 class="m-0 border-0">Study Guide</h3>
                        </div>
                        <div class="well bg-white p-4 shadow-sm border prose" onclick={handleCitationClick}>
                            {@html guideHTML}
                        </div>
                    </div>
                {/if}

                <div class="mb-3" id="lesson-notes">
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

    .spin {
        animation: spin 2s linear infinite;
    }

    @keyframes spin {
        from { transform: rotate(0deg); }
        to { transform: rotate(360deg); }
    }

    .last-child-border-0:last-child {
        border-bottom: 0 !important;
    }

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
