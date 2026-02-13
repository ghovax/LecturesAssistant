<script lang="ts">
    import { onMount, onDestroy } from 'svelte';
    import { page } from '$app/state';
    import { browser } from '$app/environment';
    import { api } from '$lib/api/client';
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';
    import StatusIndicator from '$lib/components/StatusIndicator.svelte';
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

    $effect(() => {
        if (examId && lectureId && docId) {
            loadData();
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
    });
</script>

{#if documentData && exam && lecture}
    <Breadcrumb items={[
        { label: 'My Studies', href: '/exams' }, 
        { label: exam.title, href: `/exams/${examId}` }, 
        { label: lecture.title, href: `/exams/${examId}/lectures/${lectureId}` },
        { label: documentData.title, active: true }
    ]} />

    <header class="page-header mb-5">
        <h1 class="page-title mb-2">{documentData.title}</h1>
        <StatusIndicator type="page" current={currentPageIndex + 1} total={pages.length} />
    </header>

    <div class="container-fluid p-0">
        <div class="row">
            <!-- Sidebar: Navigation -->
            <div class="col-lg-3 col-md-4 order-md-2">
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
            <div class="col-lg-8 order-md-1">
                {#if pages[currentPageIndex]}
                    {@const p = pages[currentPageIndex]}
                    <div class="well bg-white mb-3 p-0 overflow-hidden border shadow-none">
                        <div class="standard-header">
                            <div class="header-title">
                                <span class="header-text">Page {p.page_number} / {pages.length}</span>
                            </div>
                            <div class="btn-group">
                                <button 
                                    class="btn btn-link btn-sm text-dark p-0 me-2 shadow-none border-0" 
                                    disabled={currentPageIndex === 0}
                                    onclick={prevPage}
                                    title="Previous Page (Left Arrow)"
                                >
                                    <ChevronLeft size={18} />
                                </button>
                                <button 
                                    class="btn btn-link btn-sm text-dark p-0 shadow-none border-0" 
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
                                    src={api.getAuthenticatedMediaUrl(`/documents/pages/image?document_id=${docId}&lecture_id=${lectureId}&page_number=${p.page_number}`)} 
                                    alt="Page {p.page_number}"
                                    class="img-fluid shadow-sm border"
                                    style="max-height: 80vh; width: auto;"
                                />
                            </div>
                            
                            <!-- Page Text -->
                            <div class="prose">
                                {#if p.extracted_html}
                                    {@html p.extracted_html}
                                {:else}
                                    <p class="text-muted">No text content analyzed for this page.</p>
                                {/if}
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

<style lang="scss">
    .page-title {
        font-family: 'Manrope', sans-serif;
        font-size: 32px;
        font-weight: 500;
        color: var(--gray-900);
        letter-spacing: -0.02em;
    }

    .transcript-text {
        font-family: inherit;
        color: var(--gray-800);
    }
    
    .cursor-pointer {
        cursor: pointer;
    }

    .list-group-item {
        border-radius: 0;
        border-color: var(--gray-300);
        font-family: 'Manrope', sans-serif;
        font-size: 14px;
        color: var(--gray-700);

        &.active {
            background-color: var(--orange);
            border-color: var(--orange);
        }
    }

    kbd {
        background-color: var(--cream);
        border-radius: 0;
        border: 1px solid var(--gray-300);
        box-shadow: none;
        color: var(--gray-700);
        display: inline-block;
        font-size: .85em;
        font-weight: 700;
        line-height: 1;
        padding: 2px 4px;
        white-space: nowrap;
    }
</style>