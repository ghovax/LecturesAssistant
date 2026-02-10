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
    import { FileText, Clock, ChevronLeft, ChevronRight, Volume2, Plus, X } from 'lucide-svelte';

    let { id: examId, lectureId } = $derived(page.params);
    let exam = $state<any>(null);
    let lecture = $state<any>(null);
    let transcript = $state<any>(null);
    let documents = $state<any[]>([]);
    let tools = $state<any[]>([]);
    let guideTool = $derived(tools.find(t => t.type === 'guide'));
    let guideHTML = $state('');
    let loading = $state(true);
    let currentSegmentIndex = $state(0);
    let audioElement: HTMLAudioElement | null = $state(null);

    // View State
    let activeView = $state<'dashboard' | 'guide' | 'transcript' | 'doc' | 'tool'>('dashboard');
    let selectedDocId = $state<string | null>(null);
    let selectedDocPages = $state<any[]>([]);
    let selectedDocPageIndex = $state(0);
    let selectedToolId = $state<string | null>(null);

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
            selectedDocPages = await api.getDocumentPages(id, lectureId);
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
                
                try {
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
            if (activeView === 'transcript') nextSegment();
            if (activeView === 'doc') nextDocPage();
        } else if (event.key === 'ArrowLeft') {
            if (activeView === 'transcript') prevSegment();
            if (activeView === 'doc') prevDocPage();
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
        { label: exam.title, href: `/exams/${examId}` }, 
        { label: lecture.title, active: activeView === 'dashboard' },
        ...(activeView !== 'dashboard' ? [{ label: activeView === 'guide' ? 'Study Guide' : (activeView === 'transcript' ? 'Transcript' : 'Material'), active: true }] : [])
    ]} />

    <div class="d-flex justify-content-between align-items-center mb-4">
        <h2 class="m-0">{lecture.title}</h2>
        <div class="btn-group">
            <button class="btn btn-primary dropdown-toggle" data-bs-toggle="dropdown">
                <span class="glyphicon me-1"><Plus size={16} /></span> Create Study Kit
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
            <!-- Main Content Area (Left on Desktop) -->
            <div class="col-lg-9 col-md-8 order-md-1">
                {#if activeView === 'dashboard'}
                    <div class="mb-4">
                        <div class="wordBrief mb-5">
                            <div class="bg-light border-start border-4 border-primary p-3 shadow-sm">
                                <div class="small fw-bold text-muted text-uppercase mb-2" style="font-size: 0.7rem; letter-spacing: 0.1em;">Lecture Summary</div>
                                <div class="lead" style="font-size: 1.1rem; line-height: 1.5;">
                                    {lecture.description || 'No summary available for this lecture.'}
                                </div>
                            </div>
                        </div>

                        <div class="linkTiles tileSizeMd">
                            <Tile href="javascript:void(0)" icon="講" title="Transcript" onclick={() => activeView = 'transcript'}>
                                {#snippet description()}
                                    Comprehensive lecture dialogue and recordings.
                                {/snippet}
                            </Tile>

                            {#each documents as doc}
                                <Tile href="javascript:void(0)" icon="資" title={doc.title} onclick={() => openDocument(doc.id)}>
                                    {#snippet description()}
                                        {doc.page_count} pages • {doc.extraction_status === 'completed' ? 'Ready' : 'Reading...'}
                                    {/snippet}
                                </Tile>
                            {/each}
                        </div>
                    </div>
                {:else if activeView === 'guide'}
                    <div class="well bg-white p-0 overflow-hidden mb-5 border shadow-sm" onclick={handleCitationClick}>
                        <div class="bg-light px-4 py-2 border-bottom d-flex justify-content-between align-items-center">
                            <div class="d-flex align-items-center gap-2">
                                <span class="glyphicon m-0" style="font-size: 1.1rem; color: #568f27;">案</span>
                                <span class="fw-bold small" style="letter-spacing: 0.05em;">Study Guide</span>
                            </div>
                            <button class="btn btn-link btn-sm text-muted p-0 d-flex align-items-center" onclick={() => activeView = 'dashboard'}><X size={16} /></button>
                        </div>
                        <div class="p-4 prose">
                            {@html guideHTML}
                        </div>
                    </div>
                {:else if activeView === 'transcript'}
                    <div class="well bg-white p-0 overflow-hidden mb-5 border shadow-sm">
                        <div class="bg-light px-4 py-2 border-bottom d-flex justify-content-between align-items-center">
                            <div class="d-flex align-items-center gap-2">
                                <span class="glyphicon m-0" style="font-size: 1.1rem; color: #568f27;">講</span>
                                <span class="fw-bold small" style="letter-spacing: 0.05em;">Transcript</span>
                            </div>
                            <button class="btn btn-link btn-sm text-muted p-0 d-flex align-items-center" onclick={() => activeView = 'dashboard'}><X size={16} /></button>
                        </div>
                        
                        {#if transcript && transcript.segments}
                            {@const seg = transcript.segments[currentSegmentIndex]}
                            <div class="p-4">
                                <div class="mb-4 d-flex justify-content-between align-items-center bg-light p-2 border">
                                    <span class="fw-bold small">{formatTime(seg.start_millisecond)} &ndash; {formatTime(seg.end_millisecond)}</span>
                                    <div class="btn-group">
                                        <button class="btn btn-link btn-sm text-dark p-0 d-flex align-items-center me-2" disabled={currentSegmentIndex === 0} onclick={prevSegment}><ChevronLeft size={18} /></button>
                                        <button class="btn btn-link btn-sm text-dark p-0 d-flex align-items-center" disabled={currentSegmentIndex === transcript.segments.length - 1} onclick={nextSegment}><ChevronRight size={18} /></button>
                                    </div>
                                </div>

                                {#if seg.media_id}
                                    <div class="mb-4 bg-light p-3 border d-flex align-items-center gap-3">
                                        <Volume2 size={24} class="text-primary flex-shrink-0" />
                                        <audio controls class="w-100" style="height: 32px;" src="http://localhost:3000/api/media/content?media_id={seg.media_id}&session_token={localStorage.getItem('session_token')}#t={seg.original_start_milliseconds / 1000},{seg.original_end_milliseconds / 1000}"></audio>
                                    </div>
                                {/if}

                                <div class="transcript-text" style="font-size: 1rem; line-height: 1.6;">{@html seg.text_html}</div>
                            </div>
                        {:else}
                            <div class="p-5 text-center text-muted">Transcript is not available yet.</div>
                        {/if}
                    </div>
                {:else if activeView === 'doc'}
                    <div class="well bg-white p-0 overflow-hidden mb-5 border shadow-sm">
                        <div class="bg-light px-4 py-2 border-bottom d-flex justify-content-between align-items-center">
                            <div class="d-flex align-items-center gap-2">
                                <span class="glyphicon m-0" style="font-size: 1.1rem; color: #568f27;">資</span>
                                <span class="fw-bold small" style="letter-spacing: 0.05em;">Viewing Material</span>
                            </div>
                            <button class="btn btn-link btn-sm text-muted p-0 d-flex align-items-center" onclick={() => activeView = 'dashboard'}><X size={16} /></button>
                        </div>
                        
                        {#if selectedDocPages.length > 0}
                            {@const p = selectedDocPages[selectedDocPageIndex]}
                            <div class="p-4">
                                <div class="mb-4 d-flex justify-content-between align-items-center bg-light p-2 border">
                                    <span class="fw-bold small">Page {p.page_number} of {selectedDocPages.length}</span>
                                    <div class="btn-group">
                                        <button class="btn btn-link btn-sm text-dark p-0 d-flex align-items-center me-2" disabled={selectedDocPageIndex === 0} onclick={prevDocPage}><ChevronLeft size={18} /></button>
                                        <button class="btn btn-link btn-sm text-dark p-0 d-flex align-items-center" disabled={selectedDocPageIndex === selectedDocPages.length - 1} onclick={nextDocPage}><ChevronRight size={18} /></button>
                                    </div>
                                </div>

                                <div class="bg-light d-flex justify-content-center p-3 mb-4 border text-center">
                                    <img 
                                        src="http://localhost:3000/api/documents/pages/image?document_id={selectedDocId}&lecture_id={lectureId}&page_number={p.page_number}&session_token={localStorage.getItem('session_token')}" 
                                        alt="Page {p.page_number}"
                                        class="img-fluid shadow-sm border"
                                        style="max-height: 70vh; width: auto;"
                                    />
                                </div>
                                
                                <div class="transcript-text" style="font-size: 1rem; white-space: pre-wrap; line-height: 1.6;">
                                    {p.extracted_text || 'No text content extracted for this page.'}
                                </div>
                            </div>
                        {:else}
                            <div class="p-5 text-center text-muted">
                                <div class="village-spinner mx-auto mb-3"></div>
                                <p>Loading document pages...</p>
                            </div>
                        {/if}
                    </div>
                {:else if activeView === 'tool'}
                    <div class="well bg-white p-0 overflow-hidden mb-5 border shadow-sm">
                        <div class="bg-light px-4 py-2 border-bottom d-flex justify-content-between align-items-center">
                            <div class="d-flex align-items-center gap-2">
                                <span class="glyphicon m-0" style="font-size: 1.1rem; color: #568f27;">札</span>
                                <span class="fw-bold small" style="letter-spacing: 0.05em;">Viewing Study Kit</span>
                            </div>
                            <button class="btn btn-link btn-sm text-muted p-0 d-flex align-items-center" onclick={() => activeView = 'dashboard'}><X size={16} /></button>
                        </div>
                        <div class="p-4 text-center">
                            <a href="/exams/{examId}/tools/{selectedToolId}" class="btn btn-primary">Open Practice Mode</a>
                        </div>
                    </div>
                {/if}
            </div>

            <!-- Sidebar: Navigation Tiles ONLY (Right Side on Desktop) -->
            <div class="col-lg-3 col-md-4 order-md-2">
                <div class="linkTiles tileSizeMd w-100 m-0 d-flex flex-column align-items-center">
                    <Tile href="javascript:void(0)" 
                          icon={activeView === 'dashboard' ? '家' : '戻'} 
                          title={activeView === 'dashboard' ? 'Dashboard' : 'Back to Hub'} 
                          onclick={() => activeView = 'dashboard'}>
                        {#snippet description()}
                            {activeView === 'dashboard' ? 'Lecture overview and materials.' : 'Return to the lecture dashboard.'}
                        {/snippet}
                    </Tile>

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

    audio::-webkit-media-controls-enclosure {
        border-radius: 0;
        background-color: #f8f9fa;
    }
</style>
