<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { page } from "$app/state";
  import { browser } from "$app/environment";
  import { api } from "$lib/api/client";
  import Breadcrumb from "$lib/components/Breadcrumb.svelte";
  import StatusIndicator from "$lib/components/StatusIndicator.svelte";
  import { FileText, ChevronLeft, ChevronRight, Search } from "lucide-svelte";

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
      if (!examId || !lectureId || !docId) return;

      const [examR, lectureR, docR, pagesR] = await Promise.all([
        api.getExam(examId),
        api.getLecture(lectureId, examId),
        api.request(
          "GET",
          `/documents/details?document_id=${docId}&lecture_id=${lectureId}`,
        ),
        api.getDocumentPages(docId, lectureId),
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
    if (event.key === "ArrowRight") {
      nextPage();
    } else if (event.key === "ArrowLeft") {
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
      window.addEventListener("keydown", handleKeyDown);
    }
  });

  onDestroy(() => {
    if (browser) {
      window.removeEventListener("keydown", handleKeyDown);
    }
  });
</script>

{#if documentData && exam && lecture}
  <Breadcrumb
    items={[
      { label: "My Studies", href: "/exams" },
      { label: exam.title, href: `/exams/${examId}` },
      { label: lecture.title, href: `/exams/${examId}/lectures/${lectureId}` },
      { label: documentData.title, active: true },
    ]}
  />

  <header class="page-header">
    <h1 class="page-title mb-2">{documentData.title}</h1>
    <StatusIndicator
      type="page"
      current={currentPageIndex + 1}
      total={pages.length}
    />
  </header>

  <div class="container-fluid p-0">
    <div class="row">
      <!-- Main Content: Single Page Viewer -->
      <div class="col-12">
        {#if pages[currentPageIndex]}
          {@const p = pages[currentPageIndex]}
          <div class="well bg-white mb-3 p-0 border shadow-none">
            <div class="standard-header">
              <div class="header-title d-flex align-items-center gap-4">
                <span class="header-text"
                  >Page {p.page_number} / {pages.length}</span
                >
                <div class="d-flex align-items-center gap-2">
                  <span
                    class="text-muted"
                    style="font-size: 0.75rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.02em;"
                    >Go to page:</span
                  >
                  <input
                    type="number"
                    min="1"
                    max={pages.length}
                    class="form-control cozy-input p-1 text-center no-spinner"
                    style="width: 50px; height: 1.75rem; font-size: 0.8rem;"
                    placeholder=""
                    oninput={(e) => {
                      const val = parseInt(e.currentTarget.value);
                      if (!isNaN(val) && val >= 1 && val <= pages.length) {
                        currentPageIndex = val - 1;
                      }
                    }}
                    onblur={(e) => (e.currentTarget.value = "")}
                  />
                </div>
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
              <div
                class="bg-light d-flex align-items-start justify-content-center p-3 mb-4 border text-center"
              >
                <img
                  src={api.getAuthenticatedMediaUrl(
                    `/documents/pages/image?document_id=${docId}&lecture_id=${lectureId}&page_number=${p.page_number}`,
                  )}
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
                  <p class="text-muted">
                    No text content analyzed for this page.
                  </p>
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
  .transcript-text {
    font-family: inherit;
    color: var(--gray-800);
  }

  .cursor-pointer {
    cursor: pointer;
  }

  .list-group-item {
    border-radius: var(--border-radius);
    border-color: var(--gray-300);
    font-family: var(--font-primary);
    font-size: 0.85rem;
    color: var(--gray-700);

    &.active {
      background-color: var(--orange);
      border-color: var(--orange);
    }
  }

  kbd {
    background-color: var(--cream);
    border-radius: var(--border-radius);
    border: 1px solid var(--gray-300);
    box-shadow: none;
    color: var(--gray-700);
    display: inline-block;
    font-size: 0.85em;
    font-weight: 700;
    line-height: 1;
    padding: 2px 4px;
    white-space: nowrap;
  }
</style>
