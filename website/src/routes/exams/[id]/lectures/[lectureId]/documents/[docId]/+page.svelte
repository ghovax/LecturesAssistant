<script lang="ts">
    import { onMount, onDestroy } from 'svelte';
    import { page } from '$app/state';
    import { browser } from '$app/environment';
    import { api } from '$lib/api/client';
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';
    import { FileText, ChevronLeft, ChevronRight, Search } from 'lucide-svelte';

    let { id: examId, lectureId, docId } = $derived(page.params);
    let exam = $state<any>(null);
    let lecture = $state<any>(null);
    let documentData = $state<any>(null);
    let pages = $state<any[]>([]);
    let loading = $state(true);
    let currentPageIndex = $state(0);

    async function loadData() {
        loading = true;
        try {
            const [examR, lectureR, docR, pagesR] = await Promise.all([
                api.getExam(examId),
                api.getLecture(lectureId, examId),
                api.request('GET', `/documents/details?document_id=${docId}&lecture_id=${lectureId}`),
                api.getDocumentPages(docId, lectureId)
            ]);
            exam = examR;
            lecture = lectureR;
            documentData = docR;
            pages = pagesR ?? [];
        } catch (e) {
            console.error(e);
        } finally {
            loading = false;
        }
    }

    function nextPage() {
        if (currentPageIndex < pages.length - 1) {
            currentPageIndex++;
        }
    }

    function prevPage() {
        if (currentPageIndex > 0) {
            currentPageIndex--;
        }
    }

    function handleKeyDown(event: KeyboardEvent) {
        if (event.key === 'ArrowRight') {
            nextPage();
        } else if (event.key === 'ArrowLeft') {
            prevPage();
        }
    }

    onMount(() => {
        loadData();
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

{#if documentData && exam && lecture}
    <Breadcrumb items={[
        { label: 'My Studies', href: '/exams' }, 
        { label: exam.title, href: `/exams/${examId}` }, 
        { label: lecture.title, href: `/exams/${examId}/lectures/${lectureId}` },
        { label: documentData.title, active: true }
    ]} />

    <div class="d-flex justify-content-between align-items-start mb-3">
        <div>
            <h2 class="mb-1">{documentData.title}</h2>
            <span class="badge bg-dark">Page {currentPageIndex + 1} of {pages.length}</span>
        </div>
    </div>

    <div class="container-fluid p-0">
        <div class="row">
            <!-- Sidebar: Document Info & Navigation -->
            <div class="col-lg-3 col-md-4 order-md-2">
                <h3>Document Details</h3>
                <div class="well small mb-4">
                    <table class="table table-sm table-borderless m-0">
                        <tbody>
                            <tr>
                                <td style="width: 40%"><strong>Type</strong></td>
                                <td class="text-uppercase">{documentData.document_type}</td>
                            </tr>
                            <tr>
                                <td><strong>Status</strong></td>
                                <td>{documentData.extraction_status === 'completed' ? 'Fully Indexed' : 'Processing'}</td>
                            </tr>
                            <tr>
                                <td><strong>Source</strong></td>
                                <td class="text-truncate" style="max-width: 120px;" title={documentData.original_filename}>{documentData.original_filename}</td>
                            </tr>
                        </tbody>
                    </table>
                </div>

                <h3>Navigation</h3>
                <div class="list-group shadow-sm small overflow-auto" style="max-height: 50vh;">
                    {#each pages as p, i}
                        <button 
                            onclick={() => currentPageIndex = i} 
                            class="list-group-item list-group-item-action d-flex justify-content-between align-items-center text-start {currentPageIndex === i ? 'active' : ''}"
                        >
                            Page {p.page_number}
                            <span class={currentPageIndex === i ? 'text-white' : 'text-muted'}><Search size={12} /></span>
                        </button>
                    {/each}
                </div>
                <div class="mt-3 text-muted small">
                    <p><kbd>←</kbd> <kbd>→</kbd> Use arrow keys to navigate</p>
                </div>
            </div>

            <!-- Main Content: Single Page Viewer -->
            <div class="col-lg-9 col-md-8 order-md-1">
                {#if pages[currentPageIndex]}
                    {@const p = pages[currentPageIndex]}
                    <div class="well bg-white mb-5 p-0 overflow-hidden border shadow-sm">
                        <div class="bg-light px-4 py-2 border-bottom d-flex justify-content-between align-items-center">
                            <span class="fw-bold small text-uppercase">Page {p.page_number}</span>
                            <div class="btn-group">
                                <button 
                                    class="btn btn-link btn-sm text-dark p-0 me-2 shadow-none" 
                                    disabled={currentPageIndex === 0}
                                    onclick={prevPage}
                                    title="Previous Page (Left Arrow)"
                                >
                                    <ChevronLeft size={18} />
                                </button>
                                <button 
                                    class="btn btn-link btn-sm text-dark p-0 shadow-none" 
                                    disabled={currentPageIndex === pages.length - 1}
                                    onclick={nextPage}
                                    title="Next Page (Right Arrow)"
                                >
                                    <ChevronRight size={18} />
                                </button>
                            </div>
                        </div>
                        
                        <div class="p-4">
                            <!-- Page Image -->
                            <div class="bg-light d-flex align-items-start justify-content-center p-3 mb-4 border text-center">
                                <img 
                                    src="http://localhost:3000/api/documents/pages/image?document_id={docId}&lecture_id={lectureId}&page_number={p.page_number}&session_token={localStorage.getItem('session_token')}" 
                                    alt="Page {p.page_number}"
                                    class="img-fluid shadow-sm border"
                                    style="max-height: 80vh; width: auto;"
                                />
                            </div>
                            
                            <!-- Page Text -->
                            <div class="transcript-text" style="font-size: 1rem; white-space: pre-wrap; line-height: 1.6;">
                                {p.extracted_text || 'No text content extracted for this page.'}
                            </div>
                        </div>
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
        font-family: inherit;
        color: #333;
    }
    
    .cursor-pointer {
        cursor: pointer;
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
</style>