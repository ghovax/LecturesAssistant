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
    import EditModal from '$lib/components/EditModal.svelte';
    import Modal from '$lib/components/Modal.svelte';
    import ConfirmModal from '$lib/components/ConfirmModal.svelte';
    import StatusIndicator from '$lib/components/StatusIndicator.svelte';
    import { FileText, Clock, ChevronLeft, ChevronRight, Volume2, Plus, X, Edit3, Loader2, Trash2, RotateCcw, Download } from 'lucide-svelte';

    let { id: examId, lectureId } = $derived(page.params);
    let exam = $state<any>(null);
    let lecture = $state<any>(null);
    let transcript = $state<any>(null);
    let mediaFiles = $state<any[]>([]);
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
    let handledJobIds = new Set<string>(); // Prevent duplicate auto-downloads
    let isInitialJobsLoad = true;

    // Derived state for job status
    let transcriptJobRunning = $derived(jobs.some(j => j.type === 'TRANSCRIBE_MEDIA' && (j.status === 'PENDING' || j.status === 'RUNNING')));
    let documentsJobRunning = $derived(jobs.some(j => j.type === 'INGEST_DOCUMENTS' && (j.status === 'PENDING' || j.status === 'RUNNING')));
    let transcriptJobFailed = $derived(jobs.some(j => j.type === 'TRANSCRIBE_MEDIA' && j.status === 'FAILED'));
    let documentsJobFailed = $derived(jobs.some(j => j.type === 'INGEST_DOCUMENTS' && j.status === 'FAILED'));
    let transcriptJob = $derived(jobs.find(j => j.type === 'TRANSCRIBE_MEDIA'));
    let documentsJob = $derived(jobs.find(j => j.type === 'INGEST_DOCUMENTS'));
    
    // Derived tools being built
    let activeToolsJobs = $derived(jobs.filter(j => j.type === 'BUILD_MATERIAL' && (j.status === 'PENDING' || j.status === 'RUNNING' || j.status === 'FAILED')));

    let hasGuide = $derived(tools.some(t => t.type === 'guide') || activeToolsJobs.some(j => j.payload?.type === 'guide'));

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
                        // ignore
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

                // Auto-download completed exports (only if not the very first load of the page)
                if (newJob.status === 'COMPLETED' && !handledJobIds.has(newJob.id)) {
                    if (newJob.type === 'PUBLISH_MATERIAL') {
                        if (!isInitialJobsLoad) {
                            let resultData: any;
                            try {
                                resultData = JSON.parse(newJob.result);
                            } catch (e) {
                                // ignore
                            }
                            
                            if (resultData?.file_path) {
                                const filePath = resultData.file_path;
                                // Attempt automatic download immediately
                                
                                // First attempt: Standard fetch-based blob download
                                api.downloadExport(filePath).catch(err => {
                                    // Fallback: Direct authenticated window.open (more likely to bypass certain blocks if fetch failed)
                                    const directUrl = api.getAuthenticatedMediaUrl(`/exports/download?path=${encodeURIComponent(filePath)}`);
                                    window.open(directUrl, '_blank');
                                });
                                
                                notifications.success('Your export has been downloaded.');
                            }
                        }
                        handledJobIds.add(newJob.id);
                    }
                }
            }

            // If there are active jobs, start polling
            const hasActiveJobs = newJobs.some((j: any) => j.status === 'PENDING' || j.status === 'RUNNING');
            if (hasActiveJobs && !jobPollingInterval && browser) {
                startJobPolling();
            } else if (!hasActiveJobs && jobPollingInterval) {
                stopJobPolling();
            }

            // Important: we mark initial load as done AFTER the first loop 
            // so that if a job was ALREADY finished when page loaded, it's marked as handled.
            // But we must do it before the next tick where jobs is assigned.
            isInitialJobsLoad = false;
            jobs = newJobs;
        } catch (e) {
            console.error('Failed to load jobs:', e);
        }
    }

    async function retryJob(job: any) {
        if (!job.payload) return;
        try {
            // Remove the failed job record first to clean up UI
            await api.request('DELETE', '/jobs', { job_id: job.id, delete: true });
            
            // Re-trigger the tool creation with the same payload
            await api.createTool(job.payload);
            
            notifications.success(`Retrying ${job.payload.type} generation...`);
            await loadJobs();
        } catch (e: any) {
            notifications.error('Failed to retry: ' + e.message);
        }
    }

    async function retryBaseJob(type: string) {
        try {
            // Find and delete the failed job record
            const failedJob = jobs.find(j => j.type === type && j.status === 'FAILED');
            if (failedJob) {
                await api.request('DELETE', '/jobs', { job_id: failedJob.id, delete: true });
            }
            
            await api.retryLectureJob(lectureId!, examId!, type);
            notifications.success(`Retrying ${type === 'TRANSCRIBE_MEDIA' ? 'transcription' : 'document ingestion'}...`);
            await loadJobs();
            // Refresh lecture metadata too
            await loadLectureData();
        } catch (e: any) {
            notifications.error('Failed to retry: ' + e.message);
        }
    }

    async function removeJob(jobId: string) {
        try {
            await api.request('DELETE', '/jobs', { job_id: jobId, delete: true });
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
            const [lectureR, transcriptR, docsR, toolsR, mediaR] = await Promise.all([
                api.getLecture(lectureId!, examId!),
                api.request('GET', `/transcripts/html?lecture_id=${lectureId}`),
                api.listDocuments(lectureId!),
                api.request('GET', `/tools?lecture_id=${lectureId}&exam_id=${examId}`),
                api.request('GET', `/media?lecture_id=${lectureId}`)
            ]);
            lecture = lectureR;
            transcript = transcriptR;
            documents = docsR ?? [];
            tools = toolsR ?? [];
            mediaFiles = mediaR ?? [];

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
            notifications.success(`We are preparing your study guide. It will appear in the dashboard once ready.`);
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

    async function handleExportTranscript(format: string) {
        // Check for existing completed job
        const existingJob = jobs.find(j => 
            j.type === 'PUBLISH_MATERIAL' && 
            j.status === 'COMPLETED' && 
            j.payload?.lecture_id === lectureId && 
            !j.payload?.document_id && 
            !j.payload?.tool_id &&
            j.payload?.format === format
        );

        if (existingJob) {
            const res = JSON.parse(existingJob.result || '{}');
            if (res.file_path) {
                api.downloadExport(res.file_path);
                notifications.success('Your export has been downloaded.');
                return;
            }
        }

        try {
            await api.exportTranscript({ 
                lecture_id: lectureId, 
                exam_id: examId, 
                format,
                include_images: true,
                include_qr_code: true
            });
            notifications.success(`We are preparing your transcript export.`);
            await loadJobs();
        } catch (e: any) {
            notifications.error(e.message || e);
        }
    }

    async function handleExportDocument(docId: string, format: string) {
        // Check for existing completed job
        const existingJob = jobs.find(j => 
            j.type === 'PUBLISH_MATERIAL' && 
            j.status === 'COMPLETED' && 
            j.payload?.document_id === docId && 
            j.payload?.format === format
        );

        if (existingJob) {
            const res = JSON.parse(existingJob.result || '{}');
            if (res.file_path) {
                api.downloadExport(res.file_path);
                notifications.success('Your export has been downloaded.');
                return;
            }
        }

        try {
            await api.exportDocument({ 
                document_id: docId, 
                lecture_id: lectureId, 
                exam_id: examId, 
                format,
                include_images: true,
                include_qr_code: true
            });
            notifications.success(`We are preparing your document analysis export.`);
            await loadJobs();
        } catch (e: any) {
            notifications.error(e.message || e);
        }
    }

    async function handleExportTool(toolId: string, format: string) {
        // Check for existing completed job
        const existingJob = jobs.find(j => 
            j.type === 'PUBLISH_MATERIAL' && 
            j.status === 'COMPLETED' && 
            j.payload?.tool_id === toolId && 
            j.payload?.format === format
        );

        if (existingJob) {
            const res = JSON.parse(existingJob.result || '{}');
            if (res.file_path) {
                api.downloadExport(res.file_path);
                notifications.success('Your export has been downloaded.');
                return;
            }
        }

        try {
            await api.exportTool({ 
                tool_id: toolId, 
                exam_id: examId, 
                format,
                include_images: true,
                include_qr_code: true
            });
            notifications.success(`We are preparing your study guide export.`);
            await loadJobs();
        } catch (e: any) {
            notifications.error(e.message || e);
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

    <header class="page-header">
        <div class="d-flex justify-content-between align-items-center mb-2">
            <div class="d-flex align-items-center gap-3">
                <h1 class="page-title m-0">{lecture.title}</h1>
                {#if lecture.estimated_cost > 0}
                    <span class="badge bg-light text-muted border fw-normal" style="font-family: 'JetBrains Mono', monospace; font-size: 0.7rem;">
                        ${lecture.estimated_cost.toFixed(4)}
                    </span>
                {/if}
            </div>
            <div class="d-flex align-items-center gap-3">
                <button class="btn btn-link btn-sm text-muted p-0 border-0 shadow-none d-flex align-items-center" onclick={() => showEditModal = true} title="Edit Lesson">
                    <Edit3 size={16} />
                </button>
                <button 
                    class="btn btn-success rounded-0" 
                    onclick={() => createTool('guide')}
                    disabled={lecture.status !== 'ready'}
                >
                    {hasGuide ? 'Recreate' : 'Create'} Study Guide
                </button>
            </div>
        </div>
        {#if lecture.description}
            <p class="page-description text-muted">{lecture.description}</p>
        {/if}
    </header>

    <div class="container-fluid p-0">
        <div class="row g-4">
            <!-- Main Content Area -->
            <div class="col-12">
                {#if activeView === 'dashboard'}
                    <div class="mb-4">
                        <div class="linkTiles">
                            <Tile
                                icon=""
                                title="Dialogue"
                                onclick={() => activeView = 'transcript'}
                                disabled={transcriptJobRunning || !transcript || !transcript.segments}
                                class={transcriptJobRunning ? 'tile-processing' : (transcriptJobFailed ? 'tile-error' : '')}
                                cost={transcript?.estimated_cost}
                            >
                                {#snippet actions()}
                                    {#if transcriptJobRunning}
                                        <!-- No actions while running -->
                                    {:else if transcriptJobFailed}
                                        <button 
                                            class="btn btn-link text-primary p-0 border-0 shadow-none" 
                                            onclick={(e) => { e.preventDefault(); e.stopPropagation(); retryBaseJob('TRANSCRIBE_MEDIA'); }} 
                                            title="Retry Transcription"
                                        >
                                            <RotateCcw size={16} />
                                        </button>
                                    {:else if transcript && transcript.segments}
                                        {@const isCompleted = transcriptJob?.status === 'COMPLETED'}
                                        <div class="dropdown" onclick={(e) => e.stopPropagation()}>
                                            <button 
                                                class="btn btn-link {isCompleted ? 'text-orange' : 'text-muted'} p-0 border-0 shadow-none dropdown-toggle no-caret" 
                                                data-bs-toggle="dropdown" 
                                                title="Export Options"
                                            >
                                                <Download size={16} />
                                            </button>
                                            <ul class="dropdown-menu dropdown-menu-end">
                                                <li><button class="dropdown-item" onclick={() => handleExportTranscript('pdf')}>Export PDF</button></li>
                                                <li><button class="dropdown-item" onclick={() => handleExportTranscript('docx')}>Export Word</button></li>
                                            </ul>
                                        </div>
                                    {/if}
                                {/snippet}

                                {#snippet description()}
                                    {#if transcriptJobRunning}
                                        <div class="d-flex align-items-center gap-2">
                                            <div class="spinner-border spinner-border-sm" role="status">
                                                <span class="visually-hidden">Processing...</span>
                                            </div>
                                            <span>{transcriptJob?.progress || 0}%</span>
                                        </div>
                                    {:else if transcriptJobFailed}
                                        <span class="text-danger">Transcription failed. Click to retry.</span>
                                    {:else if !transcript || !transcript.segments}
                                        <span class="text-muted">Not yet available.</span>
                                    {:else}
                                        Full lesson recording and text.
                                    {/if}
                                {/snippet}
                            </Tile>

                            {#if guideTool}
                                <Tile href="javascript:void(0)" icon="" title="Study Guide" onclick={() => activeView = 'guide'} cost={guideTool.estimated_cost}>
                                    {#snippet actions()}
                                        {@const guideExportJob = jobs.find(j => j.type === 'PUBLISH_MATERIAL' && j.status === 'COMPLETED' && j.payload?.tool_id === guideTool.id)}
                                        <div class="dropdown" onclick={(e) => e.stopPropagation()}>
                                            <button 
                                                class="btn btn-link {guideExportJob ? 'text-orange' : 'text-muted'} p-0 border-0 shadow-none dropdown-toggle no-caret" 
                                                data-bs-toggle="dropdown" 
                                                title="Export Options"
                                            >
                                                <Download size={16} />
                                            </button>
                                            <ul class="dropdown-menu dropdown-menu-end">
                                                <li><button class="dropdown-item" onclick={() => handleExportTool(guideTool.id, 'pdf')}>Export PDF</button></li>
                                                <li><button class="dropdown-item" onclick={() => handleExportTool(guideTool.id, 'docx')}>Export Word</button></li>
                                            </ul>
                                        </div>
                                    {/snippet}
                                    {#snippet description()}
                                        Read the comprehensive study guide.
                                    {/snippet}
                                </Tile>
                            {:else}
                                {@const guideJob = activeToolsJobs.find(j => j.payload?.type === 'guide')}
                                {#if guideJob}
                                    <Tile
                                        icon=""
                                        title="Study Guide"
                                        disabled={guideJob.status !== 'FAILED'}
                                        onclick={() => guideJob.status === 'FAILED' && retryJob(guideJob)}
                                        class={guideJob.status === 'FAILED' ? 'tile-error' : 'tile-processing'}
                                    >
                                        {#snippet actions()}
                                            {#if guideJob.status === 'FAILED'}
                                                <button class="btn btn-link text-primary p-0 border-0 shadow-none" onclick={(e) => { e.preventDefault(); e.stopPropagation(); retryJob(guideJob); }} title="Retry">
                                                    <RotateCcw size={16} />
                                                </button>
                                            {/if}
                                        {/snippet}
                                        {#snippet description()}
                                            {#if guideJob.status === 'FAILED'}
                                                <span class="text-danger">Generation failed. Click to retry.</span>
                                            {:else}
                                                <div class="d-flex align-items-center gap-2">
                                                    <div class="spinner-border spinner-border-sm" role="status"></div>
                                                    <span>{guideJob.progress || 0}%</span>
                                                </div>
                                                <div class="small mt-1 text-truncate">{guideJob.progress_message_text || 'Preparing...'}</div>
                                            {/if}
                                        {/snippet}
                                    </Tile>
                                {/if}
                            {/if}

                            {#each documents as doc}
                                <Tile href="javascript:void(0)" icon="" title={doc.title} onclick={() => openDocument(doc.id)} cost={doc.estimated_cost}>
                                    {#snippet actions()}
                                        {@const docExportJob = jobs.find(j => j.type === 'PUBLISH_MATERIAL' && j.status === 'COMPLETED' && j.payload?.document_id === doc.id)}
                                        <div class="dropdown" onclick={(e) => e.stopPropagation()}>
                                            <button 
                                                class="btn btn-link {docExportJob ? 'text-orange' : 'text-muted'} p-0 border-0 shadow-none dropdown-toggle no-caret" 
                                                data-bs-toggle="dropdown" 
                                                title="Export Options"
                                            >
                                                <Download size={16} />
                                            </button>
                                            <ul class="dropdown-menu dropdown-menu-end">
                                                <li><button class="dropdown-item" onclick={() => handleExportDocument(doc.id, 'pdf')}>Export PDF</button></li>
                                                <li><button class="dropdown-item" onclick={() => handleExportDocument(doc.id, 'docx')}>Export Word</button></li>
                                            </ul>
                                        </div>
                                    {/snippet}
                                    {#snippet description()}
                                        Reference material.
                                    {/snippet}
                                </Tile>
                            {/each}

                            {#if documentsJobRunning || documentsJobFailed}
                                <Tile 
                                    icon="" 
                                    title="Reference Materials" 
                                    class={documentsJobRunning ? 'tile-processing' : 'tile-error'} 
                                    disabled={documentsJobRunning}
                                    onclick={() => documentsJobFailed && retryBaseJob('INGEST_DOCUMENTS')}
                                >
                                    {#snippet actions()}
                                        {#if documentsJobFailed}
                                            <button 
                                                class="btn btn-link text-primary p-0 border-0 shadow-none" 
                                                onclick={(e) => { e.preventDefault(); e.stopPropagation(); retryBaseJob('INGEST_DOCUMENTS'); }} 
                                                title="Retry Document Ingestion"
                                            >
                                                <RotateCcw size={16} />
                                            </button>
                                        {/if}
                                    {/snippet}

                                    {#snippet description()}
                                        {#if documentsJobRunning}
                                            <div class="d-flex align-items-center gap-2">
                                                <div class="spinner-border spinner-border-sm" role="status">
                                                    <span class="visually-hidden">Processing...</span>
                                                </div>
                                                <span>{documentsJob?.progress || 0}%</span>
                                            </div>
                                        {:else if documentsJobFailed}
                                            <span class="text-danger">Processing failed. Click to retry.</span>
                                        {/if}
                                    {/snippet}
                                </Tile>
                            {/if}
                        </div>

                        <div class="bg-white border mt-4">
                            <div class="standard-header">
                                <div class="header-title">
                                    <span class="header-text">Source Assets</span>
                                </div>
                            </div>
                            <div class="p-4">
                                <div class="row g-4">
                                    {#if mediaFiles.length > 0}
                                        <div class="col-md-6">
                                            <div class="cozy-label">Recordings</div>
                                            <ul class="list-unstyled mb-0">
                                                {#each mediaFiles as media}
                                                    <li class="mb-2">
                                                        <span class="filename">{media.original_filename || 'Unknown recording'}</span>
                                                    </li>
                                                {/each}
                                            </ul>
                                        </div>
                                    {/if}
                                    {#if documents.length > 0}
                                        <div class="col-md-6">
                                            <div class="cozy-label">Reference Files</div>
                                            <ul class="list-unstyled mb-0">
                                                {#each documents as doc}
                                                    <li class="mb-2">
                                                        <span class="filename">{doc.original_filename || doc.title}</span>
                                                    </li>
                                                {/each}
                                            </ul>
                                        </div>
                                    {/if}
                                </div>
                            </div>
                        </div>
                    </div>
                {:else if activeView === 'guide'}
                    <!-- svelte-ignore a11y_click_events_have_key_events -->
                    <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
                    <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
                    <div 
                        class="well bg-white p-0 overflow-hidden mb-3 border" 
                        onclick={handleCitationClick}
                        onkeydown={(e) => e.key === 'Enter' && handleCitationClick(e as any)}
                        role="article"
                        tabindex="0"
                    >
                        <div class="standard-header">
                            <div class="header-title">
                                <span class="header-text">Study Guide</span>
                            </div>
                            <div class="d-flex align-items-center gap-2">
                                <div class="dropdown">
                                    <button class="btn btn-link btn-sm text-muted p-0 d-flex align-items-center shadow-none border-0 dropdown-toggle no-caret" data-bs-toggle="dropdown" title="Export">
                                        <Download size={18} />
                                    </button>
                                    <ul class="dropdown-menu dropdown-menu-end">
                                        <li><button class="dropdown-item" onclick={() => handleExportTool(guideTool?.id || '', 'pdf')}>Export PDF</button></li>
                                        <li><button class="dropdown-item" onclick={() => handleExportTool(guideTool?.id || '', 'docx')}>Export Word</button></li>
                                    </ul>
                                </div>
                                <button class="btn btn-link btn-sm text-danger p-0 d-flex align-items-center shadow-none border-0" title="Delete Guide" onclick={() => deleteTool(guideTool?.id || '')}>
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
                            
                    <div class="well bg-white p-0 overflow-hidden mb-3 border">
                        <div class="standard-header">
                            <div class="header-title">
                                <span class="header-text">Dialogue</span>
                            </div>
                            <div class="d-flex align-items-center gap-2">
                                <div class="dropdown">
                                    <button class="btn btn-link btn-sm text-muted p-0 d-flex align-items-center shadow-none border-0 dropdown-toggle no-caret" data-bs-toggle="dropdown" title="Export">
                                        <Download size={18} />
                                    </button>
                                    <ul class="dropdown-menu dropdown-menu-end">
                                        <li><button class="dropdown-item" onclick={() => handleExportTranscript('pdf')}>Export PDF</button></li>
                                        <li><button class="dropdown-item" onclick={() => handleExportTranscript('docx')}>Export Word</button></li>
                                    </ul>
                                </div>
                                <button class="btn btn-link btn-sm text-muted p-0 d-flex align-items-center shadow-none border-0" onclick={() => activeView = 'dashboard'}><X size={20} /></button>
                            </div>
                        </div>
                        
                        {#if transcript && transcript.segments}
                            {@const seg = transcript.segments[currentSegmentIndex]}
                            <div class="p-4">
                                <div class="transcript-nav mb-4 d-flex justify-content-between align-items-center p-2 border">
                                    <div class="d-flex align-items-center gap-3">
                                        <StatusIndicator type="count" label="Segment" current={currentSegmentIndex + 1} total={transcript?.segments?.length || 0} />
                                        <StatusIndicator type="time" current={formatTime(seg.start_millisecond)} total={formatTime(seg.end_millisecond)} />
                                        {#if seg.media_filename}
                                            <span class="text-muted small border-start ps-3 d-none d-lg-inline" style="font-size: 0.75rem;">{seg.media_filename}</span>
                                        {/if}
                                    </div>
                                    <div class="btn-group">

                                        <button class="btn btn-outline-success btn-sm p-1 d-flex align-items-center me-2" disabled={currentSegmentIndex === 0} onclick={prevSegment} title="Previous Segment"><ChevronLeft size={18} /></button>
                                        <button class="btn btn-outline-success btn-sm p-1 d-flex align-items-center" disabled={currentSegmentIndex === (transcript?.segments?.length || 0) - 1} onclick={nextSegment} title="Next Segment"><ChevronRight size={18} /></button>
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
                                        <div class="spinner-border" role="status">
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
                    <div class="well bg-white p-0 overflow-hidden mb-3 border">
                        <div class="standard-header">
                            <div class="header-title">
                                <span class="header-text">{doc?.title || 'Study Resource'}</span>
                            </div>
                            <div class="d-flex align-items-center gap-2">
                                <div class="dropdown">
                                    <button class="btn btn-link btn-sm text-muted p-0 d-flex align-items-center shadow-none border-0 dropdown-toggle no-caret" data-bs-toggle="dropdown" title="Export">
                                        <Download size={18} />
                                    </button>
                                    <ul class="dropdown-menu dropdown-menu-end">
                                        <li><button class="dropdown-item" onclick={() => handleExportDocument(selectedDocId || '', 'pdf')}>Export PDF</button></li>
                                        <li><button class="dropdown-item" onclick={() => handleExportDocument(selectedDocId || '', 'docx')}>Export Word</button></li>
                                    </ul>
                                </div>
                                <button class="btn btn-link btn-sm text-muted p-0 d-flex align-items-center shadow-none border-0" onclick={() => activeView = 'dashboard'}><X size={20} /></button>
                            </div>
                        </div>
                        
                        {#if selectedDocPages && selectedDocPages.length > 0}
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
                                    <div class="spinner-border" role="status">
                                        <span class="visually-hidden">Loading...</span>
                                    </div>
                                </div>
                            </div>
                        {/if}
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
    title="Configure Study Guide" 
    isOpen={showCreateModal} 
    onClose={() => showCreateModal = false}
>
    <div class="mb-4">
        <label class="form-label cozy-label" for="tool-lang">Target Language</label>
        <select id="tool-lang" class="form-select cozy-input" bind:value={toolOptions.language_code}>
            <option value="en-US">English (US)</option>
            <option value="it-IT">Italiano</option>
            <option value="es-ES">Español</option>
            <option value="de-DE">Deutsch</option>
            <option value="fr-FR">Français</option>
            <option value="ja-JP">日本語</option>
        </select>
        <div class="form-text mt-1 mb-4" style="font-size: 0.7rem;">The assistant will translate and prepare content in this language.</div>
    </div>

    <div class="mb-0">
        <span class="form-label cozy-label">Level of Detail</span>
        <div class="d-flex gap-2 mt-3">
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
        <div class="d-flex justify-content-end">
            <button class="btn btn-success px-4" onclick={confirmCreateTool}>
                Generate Guide
            </button>
        </div>
    {/snippet}
</Modal>

<style lang="scss">
    .page-description {
        font-family: 'Manrope', sans-serif;
        font-size: 0.85rem;
        line-height: 1.6;
        max-width: 600px;
        margin: 0;
    }

    .linkTiles {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
        gap: 0;
        background: transparent;
        overflow: hidden;
        
        &.flex-column {
            grid-template-columns: 1fr;
            overflow: visible;
        }

        :global(.tile-wrapper) {
            width: 100%;
            
            :global(a), :global(button) {
                width: 100%;
            }
        }
    }

    .cozy-label {
        font-family: 'Manrope', sans-serif;
        font-size: 0.75rem;
        font-weight: 600;
        color: var(--gray-500);
        text-transform: uppercase;
        letter-spacing: 0.05em;
        margin-bottom: 8px;
        display: block;
    }

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

    /* Table of Contents Styling */
    .prose :global(#TOC) {
        background-color: #fff;
        border: 1px solid var(--gray-300);
        padding: 1rem 1.5rem;
        margin-bottom: 2rem;
        font-size: 0.85rem;
    }

    .prose :global(#TOC::before) {
        content: "Contents";
        display: block;
        font-weight: bold;
        text-transform: uppercase;
        font-size: 0.75rem;
        color: var(--gray-500);
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
        color: var(--gray-700);
        text-decoration: none;
    }

    .prose :global(#TOC a:hover) {
        color: var(--orange);
        text-decoration: underline;
    }

    audio::-webkit-media-controls-enclosure {
        border-radius: 0;
        background-color: #fff;
    }
</style>
