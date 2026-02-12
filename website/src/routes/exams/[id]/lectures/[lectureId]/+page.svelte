<script lang="ts">
    import { onMount, onDestroy, tick } from 'svelte';
    import { page } from '$app/state';
    import { browser } from '$app/environment';
    import { api } from '$lib/api/client';
    import { notifications } from '$lib/stores/notifications.svelte';
    import { capitalize } from '$lib/utils';
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';
    import Tile from '$lib/components/Tile.svelte';
    import CitationPopup from '$lib/components/CitationPopup.svelte';
    import Flashcard from '$lib/components/Flashcard.svelte';
    import EditModal from '$lib/components/EditModal.svelte';
    import Modal from '$lib/components/Modal.svelte';
    import ConfirmModal from '$lib/components/ConfirmModal.svelte';
    import StatusIndicator from '$lib/components/StatusIndicator.svelte';
    import { FileText, Clock, ChevronLeft, ChevronRight, Volume2, Plus, X, Edit3, Loader2, Trash2, RotateCcw } from 'lucide-svelte';

    let { id: examId, lectureId } = $derived(page.params);
    let exam = $state<any>(null);
    let lecture = $state<any>(null);
    let transcript = $state<any>(null);
    let documents = $state<any[]>([]);
    let tools = $state<any[]>([]);
    let jobs = $state<any[]>([]);
    let guideTool = $derived(tools.find(t => t.type === 'guide'));
    let guideHTML = $state('');
    let guideCitations = $state<any[]>([]);
    let loading = $state(true);
    let currentSegmentIndex = $state(0);
    let audioElement: HTMLAudioElement | null = $state(null);
    let jobPollingInterval: number | null = null;

    // Derived state for job status
    let transcriptJobRunning = $derived(jobs.some(j => j.type === 'TRANSCRIBE_MEDIA' && (j.status === 'PENDING' || j.status === 'RUNNING')));
    let documentsJobRunning = $derived(jobs.some(j => j.type === 'INGEST_DOCUMENTS' && (j.status === 'PENDING' || j.status === 'RUNNING')));
    let transcriptJob = $derived(jobs.find(j => j.type === 'TRANSCRIBE_MEDIA'));
    let documentsJob = $derived(jobs.find(j => j.type === 'INGEST_DOCUMENTS'));
    
    // Derived tools being built
    let activeToolsJobs = $derived(jobs.filter(j => j.type === 'BUILD_MATERIAL' && (j.status === 'PENDING' || j.status === 'RUNNING' || j.status === 'FAILED')));

    let hasGuide = $derived(tools.some(t => t.type === 'guide') || activeToolsJobs.some(j => j.payload?.type === 'guide'));
    let hasFlashcards = $derived(tools.some(t => t.type === 'flashcard') || activeToolsJobs.some(j => j.payload?.type === 'flashcard'));
    let hasQuiz = $derived(tools.some(t => t.type === 'quiz') || activeToolsJobs.some(j => j.payload?.type === 'quiz'));

    // View State
    let activeView = $state<'dashboard' | 'guide' | 'transcript' | 'doc' | 'tool'>('dashboard');

    let selectedDocId = $state<string | null>(null);
    let selectedDocPages = $state<any[]>([]);
    let selectedDocPageIndex = $state(0);
    let selectedToolId = $state<string | null>(null);

    // Tool Creation State
    let showCreateModal = $state(false);
    let showEditModal = $state(false);
    let pendingToolType = $state<string>('guide');
    let toolOptions = $state({
        length: 'medium',
        language_code: 'en-US'
    });

    // Confirmation Modal State
    let confirmModal = $state({
        isOpen: false,
        title: '',
        message: '',
        confirmText: 'Confirm',
        onConfirm: () => {},
        isDanger: false
    });

    function showConfirm(options: { title: string, message: string, confirmText?: string, onConfirm: () => void, isDanger?: boolean }) {
        confirmModal = {
            isOpen: true,
            title: options.title,
            message: options.message,
            confirmText: options.confirmText ?? 'Confirm',
            onConfirm: () => {
                options.onConfirm();
                confirmModal.isOpen = false;
            },
            isDanger: options.isDanger ?? false
        };
    }

    // Citation Popup State
    let activeCitation = $state<{ content: string, x: number, y: number, sourceFile?: string, sourcePages?: number[] } | null>(null);

    function formatTime(ms: number) {
        const totalSeconds = Math.floor(ms / 1000);
        const minutes = Math.floor(totalSeconds / 60);
        const seconds = totalSeconds % 60;
        return `${minutes}:${seconds.toString().padStart(2, '0')}`;
    }

    async function loadJobs() {
        try {
            const jobsR = await api.request('GET', `/jobs?lecture_id=${lectureId}`);
            const rawJobs = jobsR ?? [];
            const newJobs = rawJobs.map((j: any) => {
                if (typeof j.payload === 'string') {
                    try {
                        j.payload = JSON.parse(j.payload);
                    } catch (e) {
                        console.error('Failed to parse job payload', e);
                    }
                }
                return j;
            });

            // Check for newly failed jobs to notify user
            for (const newJob of newJobs) {
                const oldJob = jobs.find(j => j.id === newJob.id);
                if (newJob.status === 'FAILED' && (!oldJob || oldJob.status !== 'FAILED')) {
                    notifications.error(`${newJob.error || 'Unknown error'}`);
                }
            }

            jobs = newJobs;

            // If there are active jobs, start polling
            const hasActiveJobs = jobs.some(j => j.status === 'PENDING' || j.status === 'RUNNING');
            if (hasActiveJobs && !jobPollingInterval && browser) {
                startJobPolling();
            } else if (!hasActiveJobs && jobPollingInterval) {
                stopJobPolling();
            }
        } catch (e) {
            console.error('Failed to load jobs:', e);
        }
    }

    async function retryJob(job: any) {
        if (!job.payload) return;
        try {
            // Remove the failed job record first to clean up UI
            await api.request('POST', '/jobs/cancel', { job_id: job.id, delete: true });
            
            // Re-trigger the tool creation with the same payload
            await api.createTool(job.payload);
            
            notifications.success(`Retrying ${job.payload.type} generation...`);
            await loadJobs();
        } catch (e: any) {
            notifications.error('Failed to retry: ' + e.message);
        }
    }

    async function removeJob(jobId: string) {
        try {
            await api.request('POST', '/jobs/cancel', { job_id: jobId, delete: true });
            await loadJobs();
        } catch (e: any) {
            notifications.error('Failed to remove job record: ' + e.message);
        }
    }

    function startJobPolling() {
        if (jobPollingInterval) return;
        jobPollingInterval = window.setInterval(() => {
            loadJobs();
            // Reload lecture data to get updated transcript/documents
            loadLectureData();
        }, 3000); // Poll every 3 seconds
    }

    function stopJobPolling() {
        if (jobPollingInterval) {
            clearInterval(jobPollingInterval);
            jobPollingInterval = null;
        }
    }

    async function loadLectureData() {
        try {
            const [lectureR, transcriptR, docsR, toolsR] = await Promise.all([
                api.getLecture(lectureId!, examId!),
                api.request('GET', `/transcripts/html?lecture_id=${lectureId}`),
                api.listDocuments(lectureId!),
                api.request('GET', `/tools?lecture_id=${lectureId}&exam_id=${examId}`)
            ]);
            lecture = lectureR;
            transcript = transcriptR;
            documents = docsR ?? [];
            tools = toolsR ?? [];

            if (guideTool) {
                const htmlRes = await api.getToolHTML(guideTool.id, examId!);
                guideHTML = htmlRes.content_html;
                guideCitations = htmlRes.citations ?? [];
            }
        } catch (e) {
            console.error(e);
        }
    }

    async function loadLecture() {
        loading = true;
        try {
            const [examR, settingsR] = await Promise.all([
                api.getExam(examId!),
                api.getSettings()
            ]);
            exam = examR;

            if (settingsR?.llm?.language) {
                toolOptions.language_code = settingsR.llm.language;
            }

            await Promise.all([
                loadLectureData(),
                loadJobs()
            ]);
        } catch (e) {
            console.error(e);
        } finally {
            loading = false;
        }
    }

    async function openDocument(id: string) {
        selectedDocId = id;
        activeView = 'doc';
        selectedDocPageIndex = 0;
        try {
            selectedDocPages = await api.getDocumentPages(id, lectureId!);
        } catch (e) {
            console.error('Failed to load document pages', e);
        }
    }

    function nextDocPage() {
        if (selectedDocPageIndex < selectedDocPages.length - 1) {
            selectedDocPageIndex++;
        }
    }

    function prevDocPage() {
        if (selectedDocPageIndex > 0) {
            selectedDocPageIndex--;
        }
    }

    function openTool(id: string) {
        const tool = tools.find(t => t.id === id);
        if (tool?.type === 'guide') {
            activeView = 'guide';
        } else {
            selectedToolId = id;
            activeView = 'tool';
        }
    }

    async function deleteTool(id: string) {
        showConfirm({
            title: 'Delete Material',
            message: 'Are you sure you want to remove this study material? This cannot be undone.',
            isDanger: true,
            confirmText: 'Remove',
            onConfirm: async () => {
                try {
                    await api.request('DELETE', '/tools', { tool_id: id, exam_id: examId });
                    notifications.success('Material removed.');
                    activeView = 'dashboard';
                    await loadLectureData();
                } catch (e: any) {
                    notifications.error(e.message || e);
                }
            }
        });
    }

    async function handleCitationClick(event: MouseEvent) {
        const target = event.target as HTMLElement;
        const footnoteRef = target.closest('.footnote-ref');
        
        if (footnoteRef && guideTool) {
            event.preventDefault();
            const href = footnoteRef.getAttribute('href');
            if (href && href.startsWith('#')) {
                const id = href.substring(1);
                const numMatch = id.match(/\d+$/);
                const num = numMatch ? parseInt(numMatch[0]) : -1;
                
                const meta = guideCitations.find((c: any) => c.number === num);

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
            if (activeView === 'transcript') nextSegment();
            if (activeView === 'doc') nextDocPage();
        } else if (event.key === 'ArrowLeft') {
            if (activeView === 'transcript') prevSegment();
            if (activeView === 'doc') prevDocPage();
        }
    }

    async function createTool(type: string) {
        // Check if tool already exists or is being built
        const existing = tools.find(t => t.type === type);
        if (existing) {
            showConfirm({
                title: `Recreate ${capitalize(type)}`,
                message: `A ${type} already exists. Do you want to delete it and create a new one?`,
                confirmText: 'Recreate',
                onConfirm: async () => {
                    try {
                        await api.request('DELETE', '/tools', { tool_id: existing.id, exam_id: examId });
                        await loadLectureData();
                        // Proceed to show creation modal
                        pendingToolType = type;
                        if (lecture?.language) {
                            toolOptions.language_code = lecture.language;
                        }
                        showCreateModal = true;
                    } catch (e: any) {
                        notifications.error('Failed to remove old material: ' + e.message);
                    }
                }
            });
            return;
        }

        pendingToolType = type;
        if (lecture?.language) {
            toolOptions.language_code = lecture.language;
        }
        showCreateModal = true;
    }

    async function confirmCreateTool() {
        showCreateModal = false;
        try {
            await api.createTool({
                exam_id: examId,
                lecture_id: lectureId,
                type: pendingToolType,
                length: toolOptions.length,
                language_code: toolOptions.language_code
            });
            notifications.success(`We are preparing your ${pendingToolType}. It will appear in the study aids list once ready.`);
        } catch (e: any) {
            notifications.error(e.message || e);
        }
    }

    async function handleEditConfirm(newTitle: string, newDesc: string) {
        if (!newTitle) return;
        try {
            await api.request('PATCH', '/lectures', {
                exam_id: examId,
                lecture_id: lectureId,
                title: newTitle,
                description: newDesc
            });
            lecture.title = newTitle;
            lecture.description = newDesc;
            showEditModal = false;
            notifications.success('Lesson updated.');
        } catch (e: any) {
            notifications.error('Failed to update: ' + (e.message || e));
        }
    }

    $effect(() => {
        if (audioElement && transcript?.segments[currentSegmentIndex]) {
            audioElement.load();
        }
    });

    $effect(() => {
        // Reload data when route parameters change
        if (examId && lectureId) {
            loadLecture();
        }
    });

    onMount(() => {
        if (browser) {
            window.addEventListener('keydown', handleKeyDown);
        }
    });

    onDestroy(() => {
        if (browser) {
            window.removeEventListener('keydown', handleKeyDown);
        }
        stopJobPolling();
    });
</script>

{#if showEditModal && lecture}
    <EditModal 
        title="Edit Lesson" 
        initialTitle={lecture.title} 
        initialDescription={lecture.description || ''} 
        onConfirm={handleEditConfirm} 
        onCancel={() => showEditModal = false} 
    />
{/if}

<ConfirmModal 
    isOpen={confirmModal.isOpen}
    title={confirmModal.title}
    message={confirmModal.message}
    confirmText={confirmModal.confirmText}
    isDanger={confirmModal.isDanger}
    onConfirm={confirmModal.onConfirm}
    onCancel={() => confirmModal.isOpen = false}
/>

{#if lecture && exam}
    <Breadcrumb items={[
        { label: 'My Studies', href: '/exams' },
        { label: exam.title, href: `/exams/${examId}` },
        {
            label: lecture.title,
            href: activeView === 'dashboard' ? undefined : 'javascript:void(0)',
            active: activeView === 'dashboard',
            onclick: activeView === 'dashboard' ? undefined : () => activeView = 'dashboard'
        },
        ...(activeView !== 'dashboard' ? [{
            label: activeView === 'guide' ? 'Study Guide' :
                   activeView === 'transcript' ? 'Dialogue' :
                   activeView === 'doc' ? (documents.find(d => d.id === selectedDocId)?.title || 'Reference') :
                   activeView === 'tool' ? (tools.find(t => t.id === selectedToolId)?.title || 'Study Aid') :
                   'Resource',
            active: true
        }] : [])
    ]} />

    <div class="d-flex justify-content-between align-items-center mb-4">
        <div class="d-flex align-items-center gap-3">
            <h2 class="m-0">{lecture.title}</h2>
            <button class="btn btn-link btn-sm text-muted p-0" onclick={() => showEditModal = true} title="Edit Lesson">
                <Edit3 size={18} />
            </button>
        </div>
        <div class="btn-group">
            <button class="btn btn-primary dropdown-toggle" data-bs-toggle="dropdown">
                <span class="glyphicon me-1"><Plus size={16} /></span> Prepare Material
            </button>
            <ul class="dropdown-menu dropdown-menu-end">
                <li><button class="dropdown-item" onclick={() => createTool('guide')}>{hasGuide ? 'Recreate' : 'Create'} Study Guide</button></li>
                <li><button class="dropdown-item" onclick={() => createTool('flashcard')}>{hasFlashcards ? 'Recreate' : 'Create'} Flashcards</button></li>
                <li><button class="dropdown-item" onclick={() => createTool('quiz')}>{hasQuiz ? 'Recreate' : 'Create'} Practice Quiz</button></li>
            </ul>
        </div>
    </div>

    <div class="container-fluid p-0">
        <div class="row">
            <!-- Main Content Area (Left on Desktop) -->
            <div class="col-lg-9 col-md-8 order-md-1">
                {#if activeView === 'dashboard'}
                    <div class="mb-4">
                        <div class="bg-white border mb-5">
                            <div class="standard-header">
                                <div class="header-title">
                                    <span class="header-glyph" lang="ja">解</span>
                                    <span class="header-text">Description</span>
                                </div>
                            </div>
                            <div class="p-4 prose" style="font-size: 1.1rem;">
                                <p>{lecture.description || 'No summary available for this lesson.'}</p>
                            </div>
                        </div>

                        <div class="linkTiles tileSizeMd">
                            <Tile
                                icon="講"
                                title="Dialogue"
                                onclick={() => activeView = 'transcript'}
                                disabled={transcriptJobRunning || !transcript || !transcript.segments}
                                class={transcriptJobRunning ? 'tile-processing' : ''}
                            >
                                {#snippet description()}
                                    {#if transcriptJobRunning}
                                        <div class="d-flex align-items-center gap-2">
                                            <div class="spinner-border spinner-border-sm text-success" role="status">
                                                <span class="visually-hidden">Processing...</span>
                                            </div>
                                            <span>{transcriptJob?.progress || 0}%</span>
                                        </div>
                                    {:else if !transcript || !transcript.segments}
                                        <span class="text-muted">Not yet available.</span>
                                    {:else}
                                        Full lesson recording and text.
                                    {/if}
                                {/snippet}
                            </Tile>

                            {#each documents as doc}
                                <Tile href="javascript:void(0)" icon="資" title={doc.title} onclick={() => openDocument(doc.id)}>
                                </Tile>
                            {/each}

                            {#if documentsJobRunning}
                                <Tile icon="資" title="Reference Materials" class="tile-processing" disabled={true}>
                                    {#snippet description()}
                                        <div class="d-flex align-items-center gap-2">
                                            <div class="spinner-border spinner-border-sm text-success" role="status">
                                                <span class="visually-hidden">Processing...</span>
                                            </div>
                                            <span>{documentsJob?.progress || 0}%</span>
                                        </div>
                                    {/snippet}
                                </Tile>
                            {/if}
                        </div>
                    </div>
                {:else if activeView === 'guide'}
                    <div 
                        class="well bg-white p-0 overflow-hidden mb-5 border" 
                        onclick={handleCitationClick}
                        onkeydown={(e) => e.key === 'Enter' && handleCitationClick(e as any)}
                        role="article"
                        tabindex="0"
                    >
                        <div class="standard-header">
                            <div class="header-title">
                                <span class="header-glyph" lang="ja">案</span>
                                <span class="header-text">Study Guide</span>
                            </div>
                            <div class="d-flex align-items-center gap-2">
                                <button class="btn btn-link btn-sm text-danger p-0 d-flex align-items-center shadow-none border-0" title="Delete Guide" onclick={() => deleteTool(guideTool.id)}>
                                    <Trash2 size={18} />
                                </button>
                                <button class="btn btn-link btn-sm text-muted p-0 d-flex align-items-center shadow-none border-0" onclick={() => activeView = 'dashboard'}><X size={20} /></button>
                            </div>
                        </div>
                        <div class="p-4 prose">
                            {@html guideHTML}
                        </div>
                    </div>
                {:else if activeView === 'transcript'}
                    <div class="well bg-white p-0 overflow-hidden mb-5 border">
                        <div class="standard-header">
                            <div class="header-title">
                                <span class="header-glyph" lang="ja">講</span>
                                <span class="header-text">Dialogue</span>
                            </div>
                            <button class="btn btn-link btn-sm text-muted p-0 d-flex align-items-center shadow-none border-0" onclick={() => activeView = 'dashboard'}><X size={20} /></button>
                        </div>
                        
                        {#if transcript && transcript.segments}
                            {@const seg = transcript.segments[currentSegmentIndex]}
                            <div class="p-4">
                                <div class="transcript-nav mb-4 d-flex justify-content-between align-items-center p-2 border">
                                    <div class="d-flex align-items-center gap-3">
                                        <StatusIndicator type="count" label="Segment" current={currentSegmentIndex + 1} total={transcript.segments.length} />
                                        <StatusIndicator type="time" current={formatTime(seg.start_millisecond)} total={formatTime(seg.end_millisecond)} />
                                        {#if seg.media_filename}
                                            <span class="text-muted small border-start ps-3 d-none d-lg-inline" style="font-size: 0.75rem;">{seg.media_filename}</span>
                                        {/if}
                                    </div>
                                    <div class="btn-group">

                                        <button class="btn btn-outline-success btn-sm p-1 d-flex align-items-center me-2" disabled={currentSegmentIndex === 0} onclick={prevSegment} title="Previous Segment"><ChevronLeft size={18} /></button>
                                        <button class="btn btn-outline-success btn-sm p-1 d-flex align-items-center" disabled={currentSegmentIndex === transcript.segments.length - 1} onclick={nextSegment} title="Next Segment"><ChevronRight size={18} /></button>
                                    </div>
                                </div>

                                {#if seg.media_id}
                                    <div class="mb-4 bg-white p-0 border">
                                        <audio 
                                            bind:this={audioElement}
                                            controls 
                                            class="w-100" 
                                            style="height: 40px; display: block; background: #fff;" 
                                            src={api.getAuthenticatedMediaUrl(`/media/content?media_id=${seg.media_id}`) + `#t=${seg.original_start_milliseconds / 1000},${seg.original_end_milliseconds / 1000}`}
                                        ></audio>
                                    </div>
                                {/if}

                                <div class="prose">{@html seg.text_html}</div>
                            </div>
                        {:else}
                            <div class="p-5 text-center">
                                {#if transcriptJobRunning}
                                    <div class="d-flex flex-column align-items-center gap-3">
                                        <div class="spinner-border text-success" role="status">
                                            <span class="visually-hidden">Processing...</span>
                                        </div>
                                        <p class="text-muted mb-0">Transcribing audio... {transcriptJob?.progress || 0}%</p>
                                        {#if transcriptJob?.progress_message_text}
                                            <p class="text-muted small mb-0">{transcriptJob.progress_message_text}</p>
                                        {/if}
                                    </div>
                                {:else}
                                    <p class="text-muted mb-0">Dialogue is not available yet.</p>
                                {/if}
                            </div>
                        {/if}
                    </div>
                {:else if activeView === 'doc'}
                    {@const doc = documents.find(d => d.id === selectedDocId)}
                    <div class="well bg-white p-0 overflow-hidden mb-5 border">
                        <div class="standard-header">
                            <div class="header-title">
                                <span class="header-glyph" lang="ja">資</span>
                                <span class="header-text">{doc?.title || 'Study Resource'}</span>
                            </div>
                            <button class="btn btn-link btn-sm text-muted p-0 d-flex align-items-center shadow-none border-0" onclick={() => activeView = 'dashboard'}><X size={20} /></button>
                        </div>
                        
                        {#if selectedDocPages.length > 0}
                            {@const p = selectedDocPages[selectedDocPageIndex]}
                            <div class="p-4">
                                <div class="document-nav mb-4 d-flex justify-content-between align-items-center p-2 border">
                                    <StatusIndicator type="page" label="Page" current={p.page_number} total={selectedDocPages.length} />
                                    <div class="btn-group">

                                        <button class="btn btn-outline-primary btn-sm p-1 d-flex align-items-center me-2" disabled={selectedDocPageIndex === 0} onclick={prevDocPage} title="Previous Page"><ChevronLeft size={18} /></button>
                                        <button class="btn btn-outline-primary btn-sm p-1 d-flex align-items-center" disabled={selectedDocPageIndex === selectedDocPages.length - 1} onclick={nextDocPage} title="Next Page"><ChevronRight size={18} /></button>
                                    </div>
                                </div>

                                <div class="bg-light d-flex justify-content-center p-3 mb-4 border text-center">
                                    <img 
                                        src={api.getAuthenticatedMediaUrl(`/documents/pages/image?document_id=${selectedDocId}&lecture_id=${lectureId}&page_number=${p.page_number}`)} 
                                        alt="Page {p.page_number}"
                                        class="img-fluid shadow-sm border"
                                        style="width: 100%; height: auto;"
                                    />
                                </div>
                                
                                <div class="prose">
                                    {#if p.extracted_html}
                                        {@html p.extracted_html}
                                    {:else}
                                        <p>{p.extracted_text || 'No content analyzed for this page.'}</p>
                                    {/if}
                                </div>
                            </div>
                        {:else}
                            <div class="p-5 text-center text-muted">
                                <div class="d-flex flex-column align-items-center gap-3">
                                    <div class="spinner-border text-success" role="status">
                                        <span class="visually-hidden">Loading...</span>
                                    </div>
                                </div>
                            </div>
                        {/if}
                    </div>
                {:else if activeView === 'tool'}
                    {@const tool = tools.find(t => t.id === selectedToolId)}
                    <div class="well bg-white p-0 overflow-hidden mb-5 border">
                        <div class="standard-header">
                            <div class="header-title">
                                <span class="header-glyph" lang="ja">{tool?.type === 'flashcard' ? '札' : '問'}</span>
                                <span class="header-text">{tool?.title || 'Practice Mode'}</span>
                            </div>
                            <div class="d-flex align-items-center gap-2">
                                {#if tool}
                                    <button class="btn btn-link btn-sm text-danger p-0 d-flex align-items-center shadow-none border-0" title="Delete {capitalize(tool.type)}" onclick={() => deleteTool(tool.id)}>
                                        <Trash2 size={18} />
                                    </button>
                                {/if}
                                <button class="btn btn-link btn-sm text-muted p-0 d-flex align-items-center shadow-none border-0" onclick={() => activeView = 'dashboard'}><X size={20} /></button>
                            </div>
                        </div>
                        
                        <div class="p-4">
                            {#if tool?.type === 'flashcard'}
                                {#await api.getToolHTML(tool.id, examId!)}
                                    <div class="text-center p-5"><div class="village-spinner mx-auto"></div></div>
                                {:then toolHTML}
                                    <div class="row g-4">
                                        {#each toolHTML.content as card}
                                            <div class="col-xl-4 col-lg-6 col-md-12">
                                                <Flashcard frontHTML={card.front_html} backHTML={card.back_html} />
                                            </div>
                                        {/each}
                                    </div>
                                {/await}
                            {:else if tool?.type === 'quiz'}
                                {#await api.getToolHTML(tool.id, examId!)}
                                    <div class="text-center p-5"><div class="village-spinner mx-auto"></div></div>
                                {:then toolHTML}
                                    <div class="quiz-list">
                                        {#each toolHTML.content as item, i}
                                            <div class="bg-white mb-5 border">
                                                <div class="px-4 py-2 border-bottom bg-light d-flex justify-content-between align-items-center">
                                                    <span class="fw-bold small text-muted">Question {i + 1}</span>
                                                </div>
                                                <div class="p-4">
                                                    <div class="mb-4 fs-5 fw-bold" style="line-height: 1.4;">{@html item.question_html}</div>
                                                    
                                                    <div class="list-group mb-4 rounded-0 shadow-none">
                                                        {#each item.options_html as opt}
                                                            <div class="list-group-item py-3 border-start-0 border-end-0">{@html opt}</div>
                                                        {/each}
                                                    </div>
                                                    
                                                    <div class="bg-success bg-opacity-10 border-start border-4 border-success mb-4 p-3">
                                                        <strong class="text-success small d-block mb-1">Correct Answer</strong>
                                                        <div class="fs-6 fw-bold">{@html item.correct_answer_html}</div>
                                                    </div>
                                                    
                                                    <div class="bg-light border-start border-4 border-secondary p-3 small">
                                                        <strong class="text-muted d-block mb-1">Explanation</strong>
                                                        <div class="text-muted" style="line-height: 1.5;">{@html item.explanation_html}</div>
                                                    </div>
                                                </div>
                                            </div>
                                        {/each}
                                    </div>
                                {/await}
                            {/if}
                        </div>
                    </div>
                {/if}
            </div>

            <!-- Sidebar: Navigation Tiles ONLY (Right Side on Desktop) -->
            <div class="col-lg-3 col-md-4 order-md-2">
                {#if activeView !== 'dashboard'}
                    <div class="linkTiles tileSizeMd w-100 m-0 d-flex flex-column align-items-center mb-4">
                        <Tile icon="戻" 
                            title="Back to Hub" 
                            onclick={() => activeView = 'dashboard'}>
                            {#snippet description()}
                                Return to the lesson dashboard.
                            {/snippet}
                        </Tile>
                    </div>
                {/if}

                <div class="bg-white border mb-4">
                    <div class="standard-header">
                        <div class="header-title">
                            <span class="header-glyph" lang="ja">案</span>
                            <span class="header-text">Study Aids</span>
                        </div>
                    </div>
                    
                    {#if tools.length > 0}
                        <div class="linkTiles tileSizeMd w-100 m-0 d-flex flex-column align-items-center p-2">
                            {#if guideTool}
                                <Tile href="javascript:void(0)" icon="案" title="Study Guide" onclick={() => activeView = 'guide'}>
                                    {#snippet description()}
                                        Read the comprehensive, carefully prepared guide.
                                    {/snippet}
                                </Tile>
                            {/if}

                            {#each tools.filter(t => t.type !== 'guide') as tool}
                                <Tile href="javascript:void(0)" 
                                    icon={tool.type === 'flashcard' ? '札' : '問'} 
                                    title={capitalize(tool.type)}
                                    onclick={() => openTool(tool.id)}>
                                    {#snippet description()}
                                        Practice your knowledge.
                                    {/snippet}
                                </Tile>
                            {/each}
                        </div>
                    {:else}
                        <div class="p-4 text-center">
                            <p class="text-muted small mb-0">No materials prepared yet. Use the <strong>Prepare Material</strong> menu to generate aids.</p>
                        </div>
                    {/if}
                </div>

                {#if activeToolsJobs.length > 0}
                    <div class="bg-white border mb-4">
                        <div class="standard-header">
                            <div class="header-title">
                                <span class="header-glyph" lang="ja">作</span>
                                <span class="header-text">Current Jobs</span>
                            </div>
                        </div>
                        <div class="linkTiles tileSizeMd w-100 m-0 d-flex flex-column align-items-center p-2">
                            {#each activeToolsJobs as job}
                                <Tile
                                    icon={job.status === 'FAILED' ? '疑' : '作'} 
                                    title={job.status === 'FAILED' ? 'Failed Generation' : 'Preparing Aid'}
                                    class={job.status === 'FAILED' ? 'tile-error' : 'tile-processing'}
                                >
                                    {#snippet description()}
                                        {#if job.status === 'FAILED'}
                                            <div class="text-danger small fw-bold">
                                                <X size={14} class="me-1" /> Generation failed.
                                            </div>
                                        {:else}
                                            <div class="d-flex align-items-center gap-2">
                                                <div class="spinner-border spinner-border-sm text-success" role="status">
                                                    <span class="visually-hidden">Processing...</span>
                                                </div>
                                                <span>{job.progress}%</span>
                                            </div>
                                        {/if}
                                    {/snippet}

                                    {#snippet actions()}
                                        {#if job.status === 'FAILED'}
                                            <button 
                                                class="btn btn-link text-primary p-0 border-0 shadow-none me-2" 
                                                onclick={(e) => { e.preventDefault(); e.stopPropagation(); retryJob(job); }}
                                                title="Retry Generation"
                                            >
                                                <RotateCcw size={16} />
                                            </button>
                                            <button 
                                                class="btn btn-link text-muted p-0 border-0 shadow-none" 
                                                onclick={(e) => { e.preventDefault(); e.stopPropagation(); removeJob(job.id); }}
                                                title="Clear Error"
                                            >
                                                <X size={16} />
                                            </button>
                                        {/if}
                                    {/snippet}
                                </Tile>
                            {/each}
                        </div>
                    </div>
                {/if}
            </div>
        </div>
    </div>
{:else if loading}
    <div class="p-5 text-center">
        <div class="d-flex flex-column align-items-center gap-3">
            <div class="village-spinner mx-auto" role="status"></div>
            <p class="text-muted mb-0">Opening lesson dashboard...</p>
        </div>
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

<Modal 
    title="Customize {capitalize(pendingToolType)}" 
    glyph="作" 
    isOpen={showCreateModal} 
    onClose={() => showCreateModal = false}
>
    <div class="mb-4">
        <label class="form-label" for="tool-lang">Target Language</label>
        <select id="tool-lang" class="form-select rounded-0 border shadow-none" bind:value={toolOptions.language_code}>
            <option value="en-US">English (US)</option>
            <option value="it-IT">Italiano</option>
            <option value="es-ES">Español</option>
            <option value="de-DE">Deutsch</option>
            <option value="fr-FR">Français</option>
            <option value="ja-JP">日本語</option>
        </select>
        <div class="form-text mt-1" style="font-size: 0.7rem;">The assistant will translate and prepare content in this language.</div>
    </div>

    <div class="mb-0">
        <span class="form-label">Level of Detail</span>
        <div class="d-flex gap-2">
            {#each ['short', 'medium', 'long', 'comprehensive'] as len}
                <button 
                    class="btn flex-grow-1 rounded-0 border transition-all {toolOptions.length === len ? 'btn-primary' : 'btn-white bg-white text-dark'}"
                    onclick={() => toolOptions.length = len}
                >
                    {capitalize(len)}
                </button>
            {/each}
        </div>
    </div>

    {#snippet footer()}
        <button class="btn btn-success w-100" onclick={confirmCreateTool}>
            Create Material
        </button>
    {/snippet}
</Modal>

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

    /* Table of Contents Styling */
    .prose :global(#TOC) {
        background-color: #fcfcfc;
        border: 1px solid #eee;
        padding: 1rem 1.5rem;
        margin-bottom: 2rem;
        font-size: 0.9rem;
    }

    .prose :global(#TOC::before) {
        content: "Contents";
        display: block;
        font-weight: bold;
        text-transform: uppercase;
        font-size: 0.75rem;
        color: #666;
        margin-bottom: 0.75rem;
        letter-spacing: 0.05em;
    }

    .prose :global(#TOC ul) {
        list-style: none;
        padding-left: 0;
        margin-bottom: 0;
    }

    .prose :global(#TOC ul ul) {
        padding-left: 1.25rem;
        margin-top: 0.25rem;
    }

    .prose :global(#TOC li) {
        margin-bottom: 0.25rem;
    }

    .prose :global(#TOC a) {
        color: #568f27;
        text-decoration: none;
    }

    .prose :global(#TOC a:hover) {
        text-decoration: underline;
    }

    audio::-webkit-media-controls-enclosure {
        border-radius: 0;
        background-color: #f8f9fa;
    }
</style>
