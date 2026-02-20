<script lang="ts">
  import { onMount, onDestroy, tick } from "svelte";
  import { page } from "$app/state";
  import { browser } from "$app/environment";
  import { api } from "$lib/api/client";
  import { notifications } from "$lib/stores/notifications.svelte";
  import { capitalize } from "$lib/utils";
  import Breadcrumb from "$lib/components/Breadcrumb.svelte";
  import Tile from "$lib/components/Tile.svelte";
  import CitationPopup from "$lib/components/CitationPopup.svelte";
  import EditModal from "$lib/components/EditModal.svelte";
  import Modal from "$lib/components/Modal.svelte";
  import ConfirmModal from "$lib/components/ConfirmModal.svelte";
  import StatusIndicator from "$lib/components/StatusIndicator.svelte";
  import {
    FileText,
    Clock,
    ChevronLeft,
    ChevronRight,
    Volume2,
    Plus,
    X,
    Edit3,
    Loader2,
    Trash2,
    RotateCcw,
    Download,
  } from "lucide-svelte";
  import ExportMenu from "$lib/components/ExportMenu.svelte";

  let { id: examId, lectureId } = $derived(page.params);
  let exam = $state<any>(null);
  let lecture = $state<any>(null);
  let transcript = $state<any>(null);
  let mediaFiles = $state<any[]>([]);
  let documents = $state<any[]>([]);
  let tools = $state<any[]>([]);
  let jobs = $state<any[]>([]);
  let guideTool = $derived(tools.find((t) => t.type === "guide"));
  let guideHTML = $state("");
  let guideCitations = $state<any[]>([]);
  let loading = $state(true);
  let currentSegmentIndex = $state(0);
  let audioElement: HTMLAudioElement | null = $state(null);
  let socket: WebSocket | null = null;
  let handledJobIds = new Set<string>(); // Prevent duplicate auto-downloads
  let downloadLock = new Set<string>(); // Strict lock for actual download trigger
  let isInitialJobsLoad = true;
  let exporting = $state<Record<string, boolean>>({}); // Track active exports: "resourceId:format" -> boolean

  // Derived state for job status
  let transcriptJobRunning = $derived(
    jobs.some(
      (j) =>
        j.type === "TRANSCRIBE_MEDIA" &&
        (j.status === "PENDING" || j.status === "RUNNING"),
    ),
  );
  let documentsJobRunning = $derived(
    jobs.some(
      (j) =>
        j.type === "INGEST_DOCUMENTS" &&
        (j.status === "PENDING" || j.status === "RUNNING"),
    ),
  );
  let transcriptJobFailed = $derived(
    jobs.some((j) => j.type === "TRANSCRIBE_MEDIA" && j.status === "FAILED"),
  );
  let documentsJobFailed = $derived(
    jobs.some((j) => j.type === "INGEST_DOCUMENTS" && j.status === "FAILED"),
  );
  let transcriptJob = $derived(jobs.find((j) => j.type === "TRANSCRIBE_MEDIA"));
  let documentsJob = $derived(jobs.find((j) => j.type === "INGEST_DOCUMENTS"));

  // Derived tools being built
  let activeToolsJobs = $derived(
    jobs.filter(
      (j) =>
        j.type === "BUILD_MATERIAL" &&
        (j.status === "PENDING" ||
          j.status === "RUNNING" ||
          j.status === "FAILED"),
    ),
  );

  let hasGuide = $derived(
    tools.some((t) => t.type === "guide") ||
      activeToolsJobs.some((j) => j.payload?.type === "guide"),
  );

  // View State
  let activeView = $state<
    "dashboard" | "guide" | "transcript" | "document" | "tool" | "docs_list"
  >("dashboard");

  let selectedDocId = $state<string | null>(null);
  let selectedDocPages = $state<any[]>([]);
  let selectedDocPageIndex = $state(0);
  let selectedToolId = $state<string | null>(null);

  // Tool Creation State
  let showCreateModal = $state(false);
  let showEditModal = $state(false);
  let pendingToolType = $state<string>("guide");
  let toolOptions = $state({
    length: "medium",
    language_code: "en-US",
  });

  // Confirmation Modal State
  let confirmModal = $state({
    isOpen: false,
    title: "",
    message: "",
    confirmText: "Confirm",
    onConfirm: () => {},
    isDanger: false,
  });

  function showConfirm(options: {
    title: string;
    message: string;
    confirmText?: string;
    onConfirm: () => void;
    isDanger?: boolean;
  }) {
    confirmModal = {
      isOpen: true,
      title: options.title,
      message: options.message,
      confirmText: options.confirmText ?? "Confirm",
      onConfirm: () => {
        options.onConfirm();
        confirmModal.isOpen = false;
      },
      isDanger: options.isDanger ?? false,
    };
  }

  // Citation Popup State
  let activeCitation = $state<{
    content: string;
    x: number;
    y: number;
    sourceFile?: string;
    sourcePages?: number[];
  } | null>(null);

  function formatTime(ms: number) {
    const totalSeconds = Math.floor(ms / 1000);
    const minutes = Math.floor(totalSeconds / 60);
    const seconds = totalSeconds % 60;
    return `${minutes}:${seconds.toString().padStart(2, "0")}`;
  }

  function setupWebSocket() {
    if (!browser || !lectureId || lectureId === "undefined") return;

    if (socket) {
      socket.close();
    }

    const token = localStorage.getItem("session_token");
    const baseUrl = api.getBaseUrl().replace("http", "ws");
    socket = new WebSocket(`${baseUrl}/socket?session_token=${token}`);

    socket.onopen = () => {
      if (lectureId && lectureId !== "undefined") {
        socket?.send(
          JSON.stringify({
            type: "subscribe",
            channel: `lecture:${lectureId}`,
          }),
        );
      }
    };

    socket.onmessage = async (event) => {
      const data = JSON.parse(event.data);
      if (data.type === "job:progress") {
        handleJobUpdate(data.payload);
      }
    };

    socket.onclose = () => {
      // Only reconnect if the component hasn't been destroyed (implicitly handled by socket being set to null in onDestroy)
    };
  }

  async function handleJobUpdate(update: any) {
    const index = jobs.findIndex((j) => j.id === update.id);

    if (index !== -1) {
      // Update existing job
      const updatedJob = { ...jobs[index], ...update };
      // Ensure payload is still parsed if update doesn't have it
      if (typeof updatedJob.payload === "string") {
        try {
          updatedJob.payload = JSON.parse(updatedJob.payload);
        } catch (e) {}
      }
      jobs[index] = updatedJob;
    } else {
      // New job we didn't know about
      jobs = [update, ...jobs];
    }

    // notification for failures
    if (update.status === "FAILED") {
      notifications.error(`${update.error || "A background task failed."}`);
    }

    // Clear exporting state if a PUBLISH_MATERIAL job finishes
    if (
      update.type === "PUBLISH_MATERIAL" &&
      (update.status === "COMPLETED" ||
        update.status === "FAILED" ||
        update.status === "CANCELLED")
    ) {
      const payload =
        typeof update.payload === "string"
          ? JSON.parse(update.payload)
          : update.payload;
      const resourceId =
        payload.tool_id || payload.document_id || payload.lecture_id;
      const format = payload.format;
      if (resourceId && format) {
        if (format === "pdf") {
          exporting[`${resourceId}:pdf:true`] = false;
          exporting[`${resourceId}:pdf:false`] = false;
        } else {
          exporting[`${resourceId}:${format}`] = false;
        }
      }
    }

    // Auto-download for completed exports
    if (
      update.status === "COMPLETED" &&
      update.type === "PUBLISH_MATERIAL" &&
      !downloadLock.has(update.id)
    ) {
      downloadLock.add(update.id);
      try {
        const result =
          typeof update.result === "string"
            ? JSON.parse(update.result)
            : update.result;
        if (result?.file_path) {
          api.downloadExport(result.file_path).catch(() => {
            const directUrl = api.getAuthenticatedMediaUrl(
              `/exports/download?path=${encodeURIComponent(result.file_path)}`,
            );
            window.open(directUrl, "_blank");
          });
          notifications.success("Your export has been downloaded.");
        }
      } catch (e) {
        console.error("Failed to parse job result for auto-download", e);
      }
    }

    // Refresh lecture data if critical jobs finish
    if (
      update.status === "COMPLETED" &&
      (update.type === "TRANSCRIBE_MEDIA" ||
        update.type === "INGEST_DOCUMENTS" ||
        update.type === "BUILD_MATERIAL")
    ) {
      await loadLectureData();
    }
  }

  async function loadJobs() {
    try {
      const jobsR = await api.request("GET", `/jobs?lecture_id=${lectureId}`);
      const rawJobs = jobsR ?? [];
      jobs = rawJobs.map((j: any) => {
        if (typeof j.payload === "string") {
          try {
            j.payload = JSON.parse(j.payload);
          } catch (e) {
            // ignore
          }
        }
        // Mark existing completed exports as handled so they don't auto-download on load
        if (j.status === "COMPLETED" && j.type === "PUBLISH_MATERIAL") {
          handledJobIds.add(j.id);
        }
        return j;
      });
      isInitialJobsLoad = false;
    } catch (e) {
      console.error("Failed to load jobs:", e);
    }
  }

  async function loadLectureData() {
    try {
      const [lectureR, docsR, toolsR, mediaR] = await Promise.all([
        api.getLecture(lectureId!, examId!),
        api.listDocuments(lectureId!),
        api.request("GET", `/tools?lecture_id=${lectureId}&exam_id=${examId}`),
        api.request("GET", `/media?lecture_id=${lectureId}`),
      ]);
      lecture = lectureR;
      documents = docsR ?? [];
      tools = toolsR ?? [];
      mediaFiles = mediaR ?? [];

      // Fetch transcript separately - it may not exist for document-only lectures
      try {
        transcript = await api.request("GET", `/transcripts/html?lecture_id=${lectureId}`);
      } catch (e) {
        // Transcript not found is OK for lectures without media
        transcript = null;
      }

      if (guideTool) {
        const htmlRes = await api.getToolHTML(guideTool.id, examId!);
        guideHTML = htmlRes.content_html.replaceAll(
          'src="/api/',
          `src="${api.getBaseUrl()}/`,
        );
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
        api.getSettings(),
      ]);
      exam = examR;

      if (settingsR?.llm?.language) {
        toolOptions.language_code = settingsR.llm.language;
      }

      await Promise.all([loadLectureData(), loadJobs()]);

      if (browser) {
        setupWebSocket();
      }
    } catch (e) {
      console.error(e);
    } finally {
      loading = false;
    }
  }

  async function openDocument(id: string) {
    selectedDocId = id;
    activeView = "document";
    selectedDocPageIndex = 0;
    try {
      selectedDocPages = await api.getDocumentPages(id, lectureId!);
    } catch (e) {
      console.error("Failed to load document pages", e);
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
    const tool = tools.find((t) => t.id === id);
    if (tool?.type === "guide") {
      activeView = "guide";
    } else {
      selectedToolId = id;
      activeView = "tool";
    }
  }

  async function deleteTool(id: string) {
    showConfirm({
      title: "Delete Material",
      message:
        "Are you sure you want to remove this study material? This cannot be undone.",
      isDanger: true,
      confirmText: "Remove",
      onConfirm: async () => {
        try {
          await api.request("DELETE", "/tools", {
            tool_id: id,
            exam_id: examId,
          });
          notifications.success("Material removed.");
          activeView = "dashboard";
          await loadLectureData();
        } catch (e: any) {
          notifications.error(e.message || e);
        }
      },
    });
  }

  async function handleCitationClick(event: MouseEvent) {
    const target = event.target as HTMLElement;
    const footnoteRef = target.closest(".footnote-ref");

    if (footnoteRef && guideTool) {
      event.preventDefault();
      const href = footnoteRef.getAttribute("href");
      if (href && href.startsWith("#")) {
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
            sourcePages: meta.source_pages,
          };
        }
      }
    }
  }

  function nextSegment() {
    if (
      transcript?.segments &&
      currentSegmentIndex < transcript.segments.length - 1
    ) {
      currentSegmentIndex++;
    }
  }

  function prevSegment() {
    if (currentSegmentIndex > 0) {
      currentSegmentIndex--;
    }
  }

  function handleKeyDown(event: KeyboardEvent) {
    if (
      event.target instanceof HTMLInputElement ||
      event.target instanceof HTMLTextAreaElement
    )
      return;

    if (event.key === "ArrowRight") {
      if (activeView === "transcript") nextSegment();
      if (activeView === "document") nextDocPage();
    } else if (event.key === "ArrowLeft") {
      if (activeView === "transcript") prevSegment();
      if (activeView === "document") prevDocPage();
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
        language_code: toolOptions.language_code,
      });
      notifications.success(
        `We are preparing your study guide. It will appear in the dashboard once ready.`,
      );
      // Refresh jobs to start polling immediately
      loadJobs();
    } catch (e: any) {
      notifications.error(e.message || e);
    }
  }

  async function handleEditConfirm(newTitle: string, newDesc: string) {
    if (!newTitle) return;
    try {
      await api.request("PATCH", "/lectures", {
        exam_id: examId,
        lecture_id: lectureId,
        title: newTitle,
        description: newDesc,
      });
      lecture.title = newTitle;
      lecture.description = newDesc;
      showEditModal = false;
      notifications.success("Lesson updated.");
    } catch (e: any) {
      notifications.error("Failed to update: " + (e.message || e));
    }
  }

  async function handleExportTranscript(
    format: string,
    includeImages: boolean = true,
  ) {
    // Check for existing completed job
    const existingJob = jobs.find(
      (j) =>
        j.type === "PUBLISH_MATERIAL" &&
        j.status === "COMPLETED" &&
        j.payload?.lecture_id === lectureId &&
        !j.payload?.document_id &&
        !j.payload?.tool_id &&
        j.payload?.format === format &&
        (format !== "pdf" || j.payload?.include_images === includeImages),
    );

    if (existingJob) {
      const res = JSON.parse(existingJob.result || "{}");
      if (res.file_path) {
        api.downloadExport(res.file_path);
        notifications.success("Your export has been downloaded.");
        return;
      }
    }

    try {
      const exportKey =
        format === "pdf"
          ? `${lectureId}:pdf:${includeImages}`
          : `${lectureId}:${format}`;
      exporting[exportKey] = true;
      await api.exportTranscript({
        lecture_id: lectureId,
        exam_id: examId,
        format,
        include_images: includeImages,
      });
      notifications.success(`We are preparing your transcript export.`);
      await loadJobs();
    } catch (e: any) {
      const exportKey =
        format === "pdf"
          ? `${lectureId}:pdf:${includeImages}`
          : `${lectureId}:${format}`;
      exporting[exportKey] = false;
      notifications.error(e.message || e);
    }
  }

  async function handleExportDocument(
    docId: string,
    format: string,
    includeImages: boolean = true,
  ) {
    // Check for existing completed job
    const existingJob = jobs.find(
      (j) =>
        j.type === "PUBLISH_MATERIAL" &&
        j.status === "COMPLETED" &&
        j.payload?.document_id === docId &&
        j.payload?.format === format &&
        (format !== "pdf" || j.payload?.include_images === includeImages),
    );

    if (existingJob) {
      const res = JSON.parse(existingJob.result || "{}");
      if (res.file_path) {
        api.downloadExport(res.file_path);
        notifications.success("Your export has been downloaded.");
        return;
      }
    }

    try {
      const exportKey =
        format === "pdf"
          ? `${docId}:pdf:${includeImages}`
          : `${docId}:${format}`;
      exporting[exportKey] = true;
      await api.exportDocument({
        document_id: docId,
        lecture_id: lectureId,
        exam_id: examId,
        format,
        include_images: includeImages,
      });
      notifications.success(`We are preparing your document analysis export.`);
      await loadJobs();
    } catch (e: any) {
      const exportKey =
        format === "pdf"
          ? `${docId}:pdf:${includeImages}`
          : `${docId}:${format}`;
      exporting[exportKey] = false;
      notifications.error(e.message || e);
    }
  }

  async function handleExportTool(
    toolId: string,
    format: string,
    includeImages: boolean = true,
  ) {
    // Check for existing completed job
    const existingJob = jobs.find(
      (j) =>
        j.type === "PUBLISH_MATERIAL" &&
        j.status === "COMPLETED" &&
        j.payload?.tool_id === toolId &&
        j.payload?.format === format &&
        (format !== "pdf" || j.payload?.include_images === includeImages),
    );

    if (existingJob) {
      const res = JSON.parse(existingJob.result || "{}");
      if (res.file_path) {
        api.downloadExport(res.file_path);
        notifications.success("Your export has been downloaded.");
        return;
      }
    }

    try {
      const exportKey =
        format === "pdf"
          ? `${toolId}:pdf:${includeImages}`
          : `${toolId}:${format}`;
      exporting[exportKey] = true;
      await api.exportTool({
        tool_id: toolId,
        exam_id: examId,
        format,
        include_images: includeImages,
      });
      notifications.success(`We are preparing your study guide export.`);
      await loadJobs();
    } catch (e: any) {
      const exportKey =
        format === "pdf"
          ? `${toolId}:pdf:${includeImages}`
          : `${toolId}:${format}`;
      exporting[exportKey] = false;
      notifications.error(e.message || e);
    }
  }

  async function retryJob(job: any) {
    if (!job.payload) return;
    try {
      await api.request("DELETE", "/jobs", { job_id: job.id, delete: true });
      await api.createTool(job.payload);
      notifications.success(`Retrying ${job.payload.type} generation...`);
      loadJobs();
    } catch (e: any) {
      notifications.error("Failed to retry: " + e.message);
    }
  }

  async function retryBaseJob(type: string) {
    try {
      const failedJob = jobs.find(
        (j) => j.type === type && j.status === "FAILED",
      );
      if (failedJob) {
        await api.request("DELETE", "/jobs", {
          job_id: failedJob.id,
          delete: true,
        });
      }
      await api.retryLectureJob(lectureId!, examId!, type);
      notifications.success(
        `Retrying ${type === "TRANSCRIBE_MEDIA" ? "transcription" : "document ingestion"}...`,
      );
      loadJobs();
      loadLectureData();
    } catch (e: any) {
      notifications.error("Failed to retry: " + e.message);
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
      window.addEventListener("keydown", handleKeyDown);
    }
  });

  onDestroy(() => {
    if (browser) {
      window.removeEventListener("keydown", handleKeyDown);
    }
    socket?.close();
  });
</script>

{#if showEditModal && lecture}
  <EditModal
    title="Edit Lesson"
    initialTitle={lecture.title}
    initialDescription={lecture.description || ""}
    onConfirm={handleEditConfirm}
    onCancel={() => (showEditModal = false)}
  />
{/if}

<ConfirmModal
  isOpen={confirmModal.isOpen}
  title={confirmModal.title}
  message={confirmModal.message}
  confirmText={confirmModal.confirmText}
  isDanger={confirmModal.isDanger}
  onConfirm={confirmModal.onConfirm}
  onCancel={() => (confirmModal.isOpen = false)}
/>

{#if lecture && exam}
  <Breadcrumb
    items={[
      { label: "My Studies", href: "/exams" },
      { label: exam.title, href: `/exams/${examId}` },
      {
        label: lecture.title,
        href: activeView === "dashboard" ? undefined : "javascript:void(0)",
        active: activeView === "dashboard",
        onclick:
          activeView === "dashboard"
            ? undefined
            : () => (activeView = "dashboard"),
      },
      ...(activeView !== "dashboard"
        ? [
            {
              label:
                activeView === "guide"
                  ? "Study Guide"
                  : activeView === "transcript"
                    ? "Dialogue"
                    : activeView === "docs_list"
                      ? "Reference Materials"
                      : activeView === "document"
                        ? documents.find((d) => d.id === selectedDocId)
                            ?.title || "Reference"
                        : activeView === "tool"
                          ? tools.find((t) => t.id === selectedToolId)?.title ||
                            "Study Aid"
                          : "Resource",
              active: true,
            },
          ]
        : []),
    ]}
  />

  <header class="page-header">
    <div class="d-flex justify-content-between align-items-center mb-2">
      <div class="d-flex align-items-center gap-3">
        <h1 class="page-title m-0">{lecture.title}</h1>
      </div>
      <div class="d-flex align-items-center gap-3">
        <button
          class="btn btn-link btn-sm text-muted p-0 border-0 shadow-none d-flex align-items-center"
          onclick={() => (showEditModal = true)}
          title="Edit Lesson"
        >
          <Edit3 size={16} />
        </button>
        <button
          class="btn btn-success"
          onclick={() => createTool("guide")}
          disabled={lecture.status !== "ready"}
        >
          {hasGuide ? "Recreate" : "Create"} Study Guide
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
        {#if !loading && activeView === "dashboard"}
          <div
            class="mb-4 bg-white border dashboard-card"
            style="width: fit-content; max-width: 100%;"
          >
            <div class="standard-header">
              <div class="header-title">
                <span class="header-text">Workspace</span>
              </div>
            </div>
            <div class="link-tiles">
              <Tile
                icon=""
                title="Dialogue"
                onclick={() => (activeView = "transcript")}
                disabled={transcriptJobRunning ||
                  !transcript ||
                  !transcript.segments}
                class={transcriptJobRunning
                  ? "processing"
                  : transcriptJobFailed
                    ? "error"
                    : ""}
                cost={transcript?.estimated_cost}
              >
                {#snippet actions()}
                  {#if transcriptJobRunning}
                    <!-- No actions while running -->
                  {:else if transcriptJobFailed}
                    <button
                      class="btn btn-link text-primary p-0 border-0 shadow-none"
                      onclick={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        retryBaseJob("TRANSCRIBE_MEDIA");
                      }}
                      title="Retry Transcription"
                    >
                      <RotateCcw size={16} />
                    </button>
                  {:else if transcript && transcript.segments}
                    {@const isCompleted = transcriptJob?.status === "COMPLETED"}
                    {@const isExportingPDFWithImages =
                      exporting[`${lectureId}:pdf:true`]}
                    {@const isExportingPDFNoImages =
                      exporting[`${lectureId}:pdf:false`]}
                    {@const isExportingDocx = exporting[`${lectureId}:docx`]}
                    <ExportMenu
                      {isCompleted}
                      {isExportingPDFWithImages}
                      {isExportingPDFNoImages}
                      {isExportingDocx}
                      onExport={handleExportTranscript}
                    />
                  {/if}
                {/snippet}

                {#snippet description()}
                  {#if transcriptJobRunning}
                    <div class="d-flex align-items-center gap-2">
                      <div
                        class="spinner-border spinner-border-sm"
                        role="status"
                      >
                        <span class="visually-hidden">Processing...</span>
                      </div>
                      <span>{transcriptJob?.progress || 0}%</span>
                    </div>
                  {:else if transcriptJobFailed}
                    <span class="text-danger"
                      >Transcription failed. Click to retry.</span
                    >
                  {:else if !transcript || !transcript.segments}
                    <span class="text-muted">Not yet available.</span>
                  {:else}
                    Full lesson recording and text.
                  {/if}
                {/snippet}
              </Tile>

              {#if documents.length > 0 || documentsJobRunning || documentsJobFailed}
                <Tile
                  icon=""
                  title="Reference Materials"
                  class={documentsJobRunning
                    ? "processing"
                    : documentsJobFailed
                      ? "error"
                      : ""}
                  disabled={documentsJobRunning}
                  onclick={() => {
                    if (documentsJobFailed) {
                      retryBaseJob("INGEST_DOCUMENTS");
                    } else {
                      activeView = "docs_list";
                    }
                  }}
                >
                  {#snippet actions()}
                    {#if documentsJobFailed}
                      <button
                        class="btn btn-link text-primary p-0 border-0 shadow-none"
                        onclick={(e) => {
                          e.preventDefault();
                          e.stopPropagation();
                          retryBaseJob("INGEST_DOCUMENTS");
                        }}
                        title="Retry Document Ingestion"
                      >
                        <RotateCcw size={16} />
                      </button>
                    {/if}
                  {/snippet}

                  {#snippet description()}
                    {#if documentsJobRunning}
                      <div class="d-flex align-items-center gap-2">
                        <div
                          class="spinner-border spinner-border-sm"
                          role="status"
                        >
                          <span class="visually-hidden">Processing...</span>
                        </div>
                        <span>{documentsJob?.progress || 0}%</span>
                      </div>
                    {:else if documentsJobFailed}
                      <span class="text-danger"
                        >Processing failed. Click to retry.</span
                      >
                    {:else}
                      Access and view your uploaded reference materials.
                    {/if}
                  {/snippet}
                </Tile>
              {/if}

              {#if activeToolsJobs.find((j) => j.payload?.type === "guide")}
                {@const guideJob = activeToolsJobs.find(
                  (j) => j.payload?.type === "guide",
                )}
                <Tile
                  icon=""
                  title="Study Guide"
                  disabled={guideJob.status !== "FAILED"}
                  onclick={() =>
                    guideJob.status === "FAILED" && retryJob(guideJob)}
                  class={guideJob.status === "FAILED"
                    ? "error"
                    : "processing"}
                >
                  {#snippet actions()}
                    {#if guideJob.status === "FAILED"}
                      <button
                        class="btn btn-link text-primary p-0 border-0 shadow-none"
                        onclick={(e) => {
                          e.preventDefault();
                          e.stopPropagation();
                          retryJob(guideJob);
                        }}
                        title="Retry"
                      >
                        <RotateCcw size={16} />
                      </button>
                    {/if}
                  {/snippet}
                  {#snippet description()}
                    {#if guideJob.status === "FAILED"}
                      <span class="text-danger"
                        >Generation failed. Click to retry.</span
                      >
                    {:else}
                      <div class="d-flex align-items-center gap-2">
                        <div
                          class="spinner-border spinner-border-sm"
                          role="status"
                        ></div>
                        <span>{guideJob.progress || 0}%</span>
                      </div>
                    {/if}
                  {/snippet}
                </Tile>
              {:else if guideTool}
                <Tile
                  href="javascript:void(0)"
                  icon=""
                  title="Study Guide"
                  onclick={() => (activeView = "guide")}
                  cost={guideTool.estimated_cost}
                >
                  {#snippet actions()}
                    {@const isExportingPDFWithImages =
                      exporting[`${guideTool.id}:pdf:true`]}
                    {@const isExportingPDFNoImages =
                      exporting[`${guideTool.id}:pdf:false`]}
                    {@const isExportingDocx = exporting[`${guideTool.id}:docx`]}
                    <ExportMenu
                      isCompleted={true}
                      {isExportingPDFWithImages}
                      {isExportingPDFNoImages}
                      {isExportingDocx}
                      onExport={(format, includeImages) =>
                        handleExportTool(guideTool.id, format, includeImages)}
                    />
                  {/snippet}
                  {#snippet description()}
                    Read the comprehensive study guide.
                  {/snippet}
                </Tile>
              {/if}
            </div>
          </div>

          <div class="bg-white border mt-4 source-assets-card">
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
                          <span class="filename"
                            >{media.original_filename ||
                              "Unknown recording"}</span
                          >
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
                          <span class="filename"
                            >{doc.original_filename || doc.title}</span
                          >
                        </li>
                      {/each}
                    </ul>
                  </div>
                {/if}
              </div>
            </div>
          </div>
        {/if}
        {#if !loading && activeView === "docs_list"}
          <div
            class="bg-white border mb-3 reference-materials-card"
            style="width: fit-content; max-width: 100%;"
          >
            <div class="standard-header">
              <div class="header-title">
                <span class="header-text">Reference Materials</span>
              </div>
              <button
                class="btn btn-link btn-sm text-muted p-0 d-flex align-items-center shadow-none border-0"
                onclick={() => (activeView = "dashboard")}
                aria-label="Close Reference Materials"><X size={20} /></button
              >
            </div>
            <div class="link-tiles">
              {#each documents as doc}
                <Tile
                  href="javascript:void(0)"
                  icon=""
                  title={doc.title}
                  monospaceTitle={true}
                  onclick={() => openDocument(doc.id)}
                  cost={doc.estimated_cost}
                  disabled={doc.extraction_status !== "completed"}
                  class={doc.extraction_status !== "completed"
                    ? "processing"
                    : ""}
                >
                  {#snippet description()}
                    Analyzed on {new Date(doc.created_at).toLocaleDateString(
                      undefined,
                      { day: "numeric", month: "long" },
                    )}
                  {/snippet}
                  {#snippet actions()}
                    {@const isExportingPDFWithImages =
                      exporting[`${doc.id}:pdf:true`]}
                    {@const isExportingPDFNoImages =
                      exporting[`${doc.id}:pdf:false`]}
                    {@const isExportingDocx = exporting[`${doc.id}:docx`]}
                    <ExportMenu
                      isCompleted={doc.extraction_status === "completed"}
                      {isExportingPDFWithImages}
                      {isExportingPDFNoImages}
                      {isExportingDocx}
                      onExport={(format, includeImages) =>
                        handleExportDocument(doc.id, format, includeImages)}
                    />
                  {/snippet}
                </Tile>
              {/each}
            </div>
          </div>
        {/if}
        {#if !loading && activeView === "guide"}
          <!-- svelte-ignore a11y_click_events_have_key_events -->
          <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
          <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
          <div
            class="well bg-white p-0 mb-3 border"
            onclick={handleCitationClick}
            onkeydown={(e) =>
              e.key === "Enter" && handleCitationClick(e as any)}
            role="article"
            tabindex="0"
          >
            <div class="standard-header">
              <div class="header-title">
                <span class="header-text">Study Guide</span>
              </div>
              <div class="header-actions">
                {#if guideTool}
                  {@const isExportingPDFWithImages =
                    exporting[`${guideTool.id}:pdf:true`]}
                  {@const isExportingPDFNoImages =
                    exporting[`${guideTool.id}:pdf:false`]}
                  {@const isExportingDocx = exporting[`${guideTool.id}:docx`]}
                  <ExportMenu
                    isCompleted={true}
                    {isExportingPDFWithImages}
                    {isExportingPDFNoImages}
                    {isExportingDocx}
                    onExport={(format, includeImages) =>
                      handleExportTool(guideTool.id, format, includeImages)}
                  />
                {/if}
                <button
                  class="btn btn-link btn-sm text-danger shadow-none border-0"
                  title="Delete Guide"
                  aria-label="Delete Guide"
                  onclick={() => deleteTool(guideTool?.id || "")}
                >
                  <Trash2 size={18} />
                </button>
                <button
                  class="btn btn-link btn-sm text-muted shadow-none border-0"
                  onclick={() => (activeView = "dashboard")}
                  aria-label="Close Study Guide"><X size={20} /></button
                >
              </div>
            </div>
            <div class="p-4 prose">
              {@html guideHTML}
            </div>
          </div>
        {/if}
        {#if !loading && activeView === "transcript"}
          <div class="well bg-white p-0 mb-3 border">
            <div class="standard-header">
              <div class="header-title">
                <span class="header-text">Dialogue</span>
              </div>
              <div class="header-actions">
                {#if true}
                  {@const isExportingPDFWithImages =
                    exporting[`${lectureId}:pdf:true`]}
                  {@const isExportingPDFNoImages =
                    exporting[`${lectureId}:pdf:false`]}
                  {@const isExportingDocx = exporting[`${lectureId}:docx`]}
                  <ExportMenu
                    isCompleted={true}
                    {isExportingPDFWithImages}
                    {isExportingPDFNoImages}
                    {isExportingDocx}
                    onExport={handleExportTranscript}
                  />
                {/if}
                <button
                  class="btn btn-link btn-sm text-muted shadow-none border-0"
                  onclick={() => (activeView = "dashboard")}
                  aria-label="Close Dialogue"><X size={20} /></button
                >
              </div>
            </div>

            {#if transcript && transcript.segments}
              {@const seg = transcript.segments[currentSegmentIndex]}
              <div class="p-4">
                <div
                  class="transcript-nav mb-4 d-flex justify-content-between align-items-center p-2 border"
                >
                  <div class="d-flex align-items-center gap-3">
                    <StatusIndicator
                      type="count"
                      label="Segment"
                      current={currentSegmentIndex + 1}
                      total={transcript?.segments?.length || 0}
                    />
                    <StatusIndicator
                      type="time"
                      current={formatTime(seg.start_millisecond)}
                      total={formatTime(seg.end_millisecond)}
                    />
                    {#if seg.media_filename}
                      <span
                        class="text-muted small border-start ps-3 d-none d-lg-inline font-monospace"
                        style="font-size: 0.8rem;">{seg.media_filename}</span
                      >
                    {/if}
                  </div>
                  <div class="btn-group">
                    <button
                      class="btn btn-outline-success btn-sm p-1 d-flex align-items-center me-2"
                      disabled={currentSegmentIndex === 0}
                      onclick={prevSegment}
                      title="Previous Segment"><ChevronLeft size={18} /></button
                    >
                    <button
                      class="btn btn-outline-success btn-sm p-1 d-flex align-items-center"
                      disabled={currentSegmentIndex ===
                        (transcript?.segments?.length || 0) - 1}
                      onclick={nextSegment}
                      title="Next Segment"><ChevronRight size={18} /></button
                    >
                  </div>
                </div>

                {#if seg.media_id}
                  <div class="mb-4 bg-white p-0 border">
                    <audio
                      bind:this={audioElement}
                      controls
                      class="w-100"
                      style="height: 40px; display: block; background: #fff;"
                      src={api.getAuthenticatedMediaUrl(
                        `/media/content?media_id=${seg.media_id}`,
                      ) +
                        `#t=${seg.original_start_milliseconds / 1000},${seg.original_end_milliseconds / 1000}`}
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
                    <p class="text-muted mb-0">
                      Transcribing audio... {transcriptJob?.progress || 0}%
                    </p>
                    {#if transcriptJob?.progress_message_text}
                      <p class="text-muted small mb-0">
                        {transcriptJob.progress_message_text}
                      </p>
                    {/if}
                  </div>
                {:else}
                  <p class="text-muted mb-0">Dialogue is not available yet.</p>
                {/if}
              </div>
            {/if}
          </div>
        {/if}
        {#if !loading && activeView === "document"}
          {@const doc = documents.find((d) => d.id === selectedDocId)}
          <div class="well bg-white p-0 mb-3 border">
            <div class="standard-header">
              <div class="header-title">
                <span class="header-text font-monospace"
                  >{doc?.title || "Study Resource"}</span
                >
              </div>
              <div class="header-actions">
                {#if true}
                  {@const isExportingPDFWithImages =
                    exporting[`${selectedDocId}:pdf:true`]}
                  {@const isExportingPDFNoImages =
                    exporting[`${selectedDocId}:pdf:false`]}
                  {@const isExportingDocx = exporting[`${selectedDocId}:docx`]}
                  <ExportMenu
                    isCompleted={true}
                    {isExportingPDFWithImages}
                    {isExportingPDFNoImages}
                    {isExportingDocx}
                    onExport={(format, includeImages) =>
                      handleExportDocument(
                        selectedDocId || "",
                        format,
                        includeImages,
                      )}
                  />
                {/if}
                <button
                  class="btn btn-link btn-sm text-muted shadow-none border-0"
                  onclick={() => (activeView = "dashboard")}
                  aria-label="Close Document"><X size={20} /></button
                >
              </div>
            </div>

            {#if selectedDocPages && selectedDocPages.length > 0}
              {@const p = selectedDocPages[selectedDocPageIndex]}
              <div class="p-4">
                <div
                  class="document-nav mb-4 d-flex justify-content-between align-items-center p-2 border"
                  style="border-radius: var(--border-radius);"
                >
                  <div class="d-flex align-items-center gap-4">
                    <StatusIndicator
                      type="page"
                      label="Page"
                      current={p.page_number}
                      total={selectedDocPages.length}
                    />
                    <div class="d-flex align-items-center gap-2">
                      <span
                        class="text-muted"
                        style="font-size: 0.75rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.02em;"
                        >Go to page:</span
                      >
                      <input
                        type="number"
                        min="1"
                        max={selectedDocPages.length}
                        class="form-control cozy-input p-1 text-center no-spinner"
                        style="width: 50px; height: 1.75rem; font-size: 0.8rem;"
                        placeholder=""
                        oninput={(e) => {
                          const val = parseInt(e.currentTarget.value);
                          if (
                            !isNaN(val) &&
                            val >= 1 &&
                            val <= selectedDocPages.length
                          ) {
                            selectedDocPageIndex = val - 1;
                          }
                        }}
                        onblur={(e) => (e.currentTarget.value = "")}
                      />
                    </div>
                  </div>
                  <div class="btn-group">
                    <button
                      class="btn btn-outline-primary btn-sm p-1 d-flex align-items-center me-2"
                      disabled={selectedDocPageIndex === 0}
                      onclick={prevDocPage}
                      title="Previous Page"><ChevronLeft size={18} /></button
                    >
                    <button
                      class="btn btn-outline-primary btn-sm p-1 d-flex align-items-center"
                      disabled={selectedDocPageIndex ===
                        selectedDocPages.length - 1}
                      onclick={nextDocPage}
                      title="Next Page"><ChevronRight size={18} /></button
                    >
                  </div>
                </div>

                <div
                  class="bg-light d-flex justify-content-center p-3 mb-4 border text-center position-relative"
                  style="border-radius: var(--border-radius);"
                >
                  <!-- Left click area for previous page -->
                  {#if selectedDocPageIndex > 0}
                    <div
                      class="page-nav-overlay page-nav-left"
                      onclick={prevDocPage}
                      title="Previous page"
                    ></div>
                  {/if}

                  <img
                    src={api.getAuthenticatedMediaUrl(
                      `/documents/pages/image?document_id=${selectedDocId}&lecture_id=${lectureId}&page_number=${p.page_number}`,
                    )}
                    alt="Page {p.page_number}"
                    class="img-fluid shadow-sm border"
                    style="width: 100%; height: auto;"
                  />

                  <!-- Right click area for next page -->
                  {#if selectedDocPageIndex < selectedDocPages.length - 1}
                    <div
                      class="page-nav-overlay page-nav-right"
                      onclick={nextDocPage}
                      title="Next page"
                    ></div>
                  {/if}
                </div>

                <div class="prose">
                  {#if p.extracted_html}
                    {@html p.extracted_html}
                  {:else}
                    <p>
                      {p.extracted_text || "No content analyzed for this page."}
                    </p>
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
    onClose={() => (activeCitation = null)}
  />
{/if}

<Modal
  title="Configure Study Guide"
  isOpen={showCreateModal}
  onClose={() => (showCreateModal = false)}
>
  <div class="mb-4">
    <label class="form-label cozy-label" for="tool-lang">Target Language</label>
    <select
      id="tool-lang"
      class="form-select cozy-input"
      bind:value={toolOptions.language_code}
    >
      <option value="en-US">English (US)</option>
      <option value="it-IT">Italiano</option>
      <option value="es-ES">Español</option>
      <option value="de-DE">Deutsch</option>
      <option value="tr-TR">Türkçe</option>
      <option value="fr-FR">Français</option>
      <option value="ja-JP">日本語</option>
    </select>
    <div class="form-text mt-1 mb-4" style="font-size: 0.75rem;">
      The assistant will translate and prepare content in this language.
    </div>
  </div>

  <div class="mb-0">
    <span class="form-label cozy-label">Level of Detail</span>
    <div class="d-flex gap-2 mt-3">
      {#each ["short", "medium", "long", "comprehensive"] as len}
        <button
          class="btn detail-level-btn flex-grow-1 border transition-all {toolOptions.length ===
          len
            ? 'btn-primary'
            : 'btn-white bg-white text-dark'}"
          data-length={len}
          onclick={() => (toolOptions.length = len)}
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
    font-family: var(--font-primary);
    font-size: 0.95rem;
    line-height: 1.6;
    max-width: 600px;
    margin: 0;
  }

  .page-nav-overlay {
    position: absolute;
    top: 0;
    bottom: 0;
    width: 15%;
    z-index: 10;
    cursor: pointer;

    &.page-nav-left {
      left: 0;
    }

    &.page-nav-right {
      right: 0;
    }
  }

  .link-tiles {
    display: flex;
    flex-wrap: wrap;
    gap: 0;
    background: transparent;
    overflow: visible;

    &.flex-column {
      flex-direction: column;
      overflow: visible;
    }

    :global(.action-tile),
    :global(.tile-wrapper) {
      width: 250px;
      border-right: 1px solid var(--gray-300);
      border-radius: 0 !important;

      &:last-child {
        border-right: none;
      }
    }

    &.flex-column :global(.action-tile),
    &.flex-column :global(.tile-wrapper) {
      border-right: none;
      border-bottom: 1px solid var(--gray-300);

      &:last-child {
        border-bottom: none;
      }
    }
  }

  .prose :global(h2) {
    font-size: 1.25rem;
    margin-top: 2rem;
    border-bottom: 1px solid var(--gray-300);
    padding-bottom: 0.5rem;
    color: var(--gray-900);
  }
  .prose :global(h3) {
    font-size: 1.1rem;
    margin-top: 1.5rem;
    color: var(--gray-700);
  }
  .prose :global(p) {
    line-height: 1.6;
    margin-bottom: 1rem;
    font-size: 0.9rem;
  }
  .prose :global(ul) {
    margin-bottom: 1rem;
    font-size: 0.9rem;
  }
  .prose :global(li) {
    margin-bottom: 0.5rem;
    font-size: 0.9rem;
  }

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
    color: var(--orange);
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
    font-size: 0.8rem;
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
    border-radius: var(--border-radius);
    background-color: #fff;
  }

  .detail-level-btn[data-length="short"] {
    background-color: rgba(220, 38, 38, 0.2) !important;
    border-color: rgba(220, 38, 38, 0.5) !important;
    color: rgba(185, 28, 28, 1) !important;
  }
  .detail-level-btn[data-length="short"]:hover {
    background-color: rgba(220, 38, 38, 0.4) !important;
  }
  .detail-level-btn[data-length="short"].btn-primary {
    background-color: rgba(220, 38, 38, 1) !important;
    border-color: rgba(185, 28, 28, 1) !important;
    color: #fff !important;
  }

  .detail-level-btn[data-length="medium"] {
    background-color: rgba(245, 158, 11, 0.2) !important;
    border-color: rgba(245, 158, 11, 0.5) !important;
    color: rgba(180, 83, 9, 1) !important;
  }
  .detail-level-btn[data-length="medium"]:hover {
    background-color: rgba(245, 158, 11, 0.4) !important;
  }
  .detail-level-btn[data-length="medium"].btn-primary {
    background-color: rgba(245, 158, 11, 1) !important;
    border-color: rgba(217, 119, 6, 1) !important;
    color: #fff !important;
  }

  .detail-level-btn[data-length="long"] {
    background-color: rgba(132, 204, 22, 0.2) !important;
    border-color: rgba(132, 204, 22, 0.5) !important;
    color: rgba(63, 98, 18, 1) !important;
  }
  .detail-level-btn[data-length="long"]:hover {
    background-color: rgba(132, 204, 22, 0.4) !important;
  }
  .detail-level-btn[data-length="long"].btn-primary {
    background-color: rgba(132, 204, 22, 1) !important;
    border-color: rgba(101, 163, 13, 1) !important;
    color: #fff !important;
  }

  .detail-level-btn[data-length="comprehensive"] {
    background-color: rgba(34, 197, 94, 0.2) !important;
    border-color: rgba(34, 197, 94, 0.5) !important;
    color: rgba(22, 101, 52, 1) !important;
  }
  .detail-level-btn[data-length="comprehensive"]:hover {
    background-color: rgba(34, 197, 94, 0.4) !important;
  }
  .detail-level-btn[data-length="comprehensive"].btn-primary {
    background-color: rgba(34, 197, 94, 1) !important;
    border-color: rgba(22, 163, 74, 1) !important;
    color: #fff !important;
  }

  .reference-materials-card {
    border-radius: var(--border-radius) !important;
    overflow: hidden;
  }
</style>
