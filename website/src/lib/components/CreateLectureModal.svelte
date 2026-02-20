<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "$lib/api/client";
  import { notifications } from "$lib/stores/notifications.svelte";
  import { goto } from "$app/navigation";
  import Modal from "./Modal.svelte";
  import {
    Upload,
    FileText,
    Info,
    X,
    Music,
    GripVertical,
    FileUp,
    Film,
    Mic,
  } from "lucide-svelte";

  interface Props {
    isOpen: boolean;
    examId: string;
    onClose: () => void;
    onLectureCreated?: () => void;
  }

  let { isOpen, examId, onClose, onLectureCreated }: Props = $props();

  let exam = $state<any>(null);
  let title = $state("");
  let description = $state("");
  let language = $state("");
  let mediaFiles = $state<File[]>([]);
  let documentFiles = $state<File[]>([]);
  let uploading = $state(false);
  let status = $state("");
  let isDragging = $state(false);

  const mediaExtensions = [
    "mp4",
    "mkv",
    "mov",
    "webm",
    "mp3",
    "wav",
    "m4a",
    "flac",
  ];
  const docExtensions = ["pdf", "pptx", "docx"];

  function handleFiles(files: FileList | File[]) {
    const selected = Array.from(files);
    const newMedia: File[] = [];
    const newDocs: File[] = [];

    selected.forEach((file) => {
      const ext = file.name.split(".").pop()?.toLowerCase() || "";
      if (mediaExtensions.includes(ext)) {
        newMedia.push(file);
      } else if (docExtensions.includes(ext)) {
        newDocs.push(file);
      } else {
        notifications.info(`Skipped unsupported file: ${file.name}`);
      }
    });

    mediaFiles = [...mediaFiles, ...newMedia];
    documentFiles = [...documentFiles, ...newDocs];
  }

  function onFileSelect(e: Event) {
    const input = e.target as HTMLInputElement;
    if (input.files) handleFiles(input.files);
  }

  function onDrop(e: DragEvent) {
    e.preventDefault();
    isDragging = false;
    if (e.dataTransfer?.files) handleFiles(e.dataTransfer.files);
  }

  function removeMedia(index: number) {
    mediaFiles = mediaFiles.filter((_, i) => i !== index);
  }

  function removeDocument(index: number) {
    documentFiles = documentFiles.filter((_, i) => i !== index);
  }

  function resetForm() {
    title = "";
    description = "";
    language = "";
    mediaFiles = [];
    documentFiles = [];
    status = "";
  }

  function handleClose() {
    if (!uploading) {
      resetForm();
      onClose();
    }
  }

  async function handleUpload() {
    if (
      !examId ||
      !title ||
      (mediaFiles.length === 0 && documentFiles.length === 0)
    )
      return;

    uploading = true;
    try {
      const formData = new FormData();
      formData.append("exam_id", examId);
      formData.append("title", title);
      formData.append("description", description || "");
      if (language) formData.append("language", language);

      mediaFiles.forEach((file) => formData.append("media", file));
      documentFiles.forEach((file) => formData.append("documents", file));

      status = "Processing upload...";
      await api.createLecture(formData);

      notifications.success(
        "The lesson has been added. We are now preparing your materials.",
      );
      resetForm();
      onClose();
      onLectureCreated?.();
    } catch (e: any) {
      notifications.error(e.message || e);
      uploading = false;
    }
  }

  $effect(() => {
    if (isOpen && examId) {
      onMount(async () => {
        const [examData, settings] = await Promise.all([
          api.getExam(examId),
          api.getSettings(),
        ]);
        exam = examData;
        if (exam?.language) language = exam.language;
        else if (settings?.llm?.language) language = settings.llm.language;
      });
    }
  });
</script>

<Modal
  title="Create New Lesson"
  {isOpen}
  onClose={handleClose}
  maxWidth="750px"
>
  {#if exam}
    <div class="d-flex flex-column gap-4">
      <!-- Step 1: Lesson Details -->
      <section>
        <div class="section-header mb-3">
          <span class="section-badge">1</span>
          <h6 class="section-title">Lesson Details</h6>
        </div>
        <div class="row g-3">
          <div class="col-12">
            <label for="lesson-title" class="cozy-label">Title</label>
            <input
              id="lesson-title"
              type="text"
              class="form-control cozy-input"
              placeholder="e.g. Cellular Respiration"
              bind:value={title}
              required
              disabled={uploading}
            />
          </div>
          <div class="col-12">
            <label for="lesson-desc" class="cozy-label"
              >Description (Optional)</label
            >
            <textarea
              id="lesson-desc"
              class="form-control cozy-input"
              rows="3"
              placeholder="What is this lesson about?"
              bind:value={description}
              disabled={uploading}
              style="height: auto !important;"
            ></textarea>
          </div>
          <div class="col-md-6">
            <label for="lesson-lang" class="cozy-label"
              >Processing Language</label
            >
            <select
              id="lesson-lang"
              class="form-select cozy-input"
              bind:value={language}
              disabled={uploading}
            >
              <option value="en-US">English (US)</option>
              <option value="it-IT">Italian</option>
              <option value="ja-JP">Japanese</option>
              <option value="es-ES">Spanish</option>
              <option value="fr-FR">French</option>
              <option value="de-DE">German</option>
              <option value="tr-TR">Turkish</option>
              <option value="zh-CN">Chinese (Simplified)</option>
              <option value="pt-BR">Portuguese (Brazilian)</option>
            </select>
          </div>
        </div>
      </section>

      <!-- Step 2: Upload Materials -->
      <section>
        <div class="section-header mb-3">
          <span class="section-badge">2</span>
          <h6 class="section-title">Upload Materials</h6>
        </div>

        <!-- Dropzone -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <!-- svelte-ignore a11y_mouse_events_have_key_events -->
        <div
          class="dropzone-refined mb-3 {isDragging ? 'is-dragging' : ''}"
          ondragover={(e) => {
            e.preventDefault();
            isDragging = true;
          }}
          ondragleave={() => (isDragging = false)}
          ondrop={onDrop}
        >
          <input
            type="file"
            id="file-input"
            class="d-none"
            multiple
            onchange={onFileSelect}
            disabled={uploading}
          />
          <label for="file-input" class="dropzone-label-refined">
            <div class="dropzone-content">
              <FileUp size={40} class="text-orange mb-3" />
              <p class="dropzone-text mb-2">
                <span class="fw-bold">Click to browse</span> or drag files here
              </p>
              <p class="dropzone-hint mb-0">
                <Film size={12} class="me-1" /> MP4, MOV, MKV, WebM
                <span class="mx-2">•</span>
                <Mic size={12} class="me-1" /> MP3, WAV, M4A
                <span class="mx-2">•</span>
                <FileText size={12} class="me-1" /> PDF, PPTX, DOCX
              </p>
            </div>
          </label>
        </div>

        <!-- File Lists -->
        {#if mediaFiles.length > 0 || documentFiles.length > 0}
          <div class="uploaded-files">
            {#if mediaFiles.length > 0}
              <div class="file-group mb-3">
                <div class="file-group-header">
                  <Music size={14} class="text-orange me-2" />
                  <span class="file-group-title"
                    >Recordings ({mediaFiles.length})</span
                  >
                </div>
                <div class="file-list">
                  {#each mediaFiles as file, i}
                    <div
                      class="file-item recording"
                      draggable={!uploading}
                      ondragstart={(e: DragEvent) =>
                        !uploading &&
                        e.dataTransfer?.setData("text/plain", i.toString())}
                      ondragover={(e: DragEvent) => {
                        e.preventDefault();
                        if (
                          !uploading &&
                          e.currentTarget instanceof HTMLElement
                        )
                          e.currentTarget.style.borderTop =
                            "2px solid var(--orange)";
                      }}
                      ondragleave={(e: DragEvent) => {
                        if (e.currentTarget instanceof HTMLElement)
                          e.currentTarget.style.borderTop = "";
                      }}
                      ondrop={(e: DragEvent) => {
                        e.preventDefault();
                        if (e.currentTarget instanceof HTMLElement)
                          e.currentTarget.style.borderTop = "";
                        if (uploading) return;
                        const fromIndex = parseInt(
                          e.dataTransfer?.getData("text/plain") || "-1",
                        );
                        if (fromIndex !== -1 && fromIndex !== i) {
                          const files = [...mediaFiles];
                          const [moved] = files.splice(fromIndex, 1);
                          files.splice(i, 0, moved);
                          mediaFiles = files;
                        }
                      }}
                    >
                      <div
                        class="d-flex align-items-center gap-2 overflow-hidden"
                      >
                        <GripVertical
                          size={14}
                          class="text-muted flex-shrink-0"
                          style="cursor: grab;"
                        />
                        <Film size={14} class="text-orange flex-shrink-0" />
                        <span
                          class="text-truncate small filename"
                          title={file.name}>{file.name}</span
                        >
                      </div>
                      <button
                        class="btn btn-link btn-sm text-danger p-0 border-0 shadow-none ms-2 flex-shrink-0"
                        onclick={() => removeMedia(i)}
                        disabled={uploading}
                        aria-label="Remove {file.name}"
                      >
                        <X size={14} />
                      </button>
                    </div>
                  {/each}
                </div>
              </div>
            {/if}

            {#if documentFiles.length > 0}
              <div class="file-group">
                <div class="file-group-header">
                  <FileText size={14} class="text-primary me-2" />
                  <span class="file-group-title"
                    >Documents ({documentFiles.length})</span
                  >
                </div>
                <div class="file-list">
                  {#each documentFiles as file, i}
                    <div class="file-item document">
                      <div
                        class="d-flex align-items-center gap-2 overflow-hidden"
                      >
                        <div style="width: 14px;"></div>
                        <FileText
                          size={14}
                          class="text-primary flex-shrink-0"
                        />
                        <span
                          class="text-truncate small filename"
                          title={file.name}>{file.name}</span
                        >
                      </div>
                      <button
                        class="btn btn-link btn-sm text-danger p-0 border-0 shadow-none ms-2 flex-shrink-0"
                        onclick={() => removeDocument(i)}
                        disabled={uploading}
                        aria-label="Remove {file.name}"
                      >
                        <X size={14} />
                      </button>
                    </div>
                  {/each}
                </div>
              </div>
            {/if}
          </div>
        {/if}
      </section>

      <!-- Action Button -->
      <section class="mt-2">
        {#if uploading}
          <div class="text-center py-3">
            <div class="village-spinner mx-auto mb-3"></div>
            <p
              class="text-muted fw-bold small uppercase letter-spacing-05 mb-0"
            >
              {status}
            </p>
          </div>
        {:else}
          <div class="d-flex justify-content-between align-items-center">
            <div class="d-flex align-items-center gap-2 text-muted small">
              <Info size={14} />
              <span>Multiple files will be combined into one lesson.</span>
            </div>
            <button
              class="btn btn-success px-4"
              onclick={handleUpload}
              disabled={!title ||
                (mediaFiles.length === 0 && documentFiles.length === 0)}
            >
              <Upload size={18} />
              Start Processing Lesson
            </button>
          </div>
        {/if}
      </section>
    </div>
  {/if}
</Modal>

<style lang="scss">
  /* Section Headers */
  .section-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .section-badge {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    background: var(--orange);
    color: #fff;
    border-radius: 50%;
    font-size: 0.7rem;
    font-weight: 700;
    flex-shrink: 0;
  }

  .section-title {
    font-size: 0.85rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--gray-700);
    margin: 0;
  }

  /* Dropzone */
  .dropzone-refined {
    border: 1.25px dashed var(--gray-300);
    border-radius: var(--border-radius);
    background: #fff;
    padding: 40px 24px;
    transition: all 0.2s ease;
    cursor: pointer;
    text-align: center;

    &:hover {
      border-color: var(--gray-400);
      background: #fafaf9;
    }

    &.is-dragging {
      border-color: var(--orange);
      background: #fffafa;
      border-style: solid;
    }

    .dropzone-label-refined {
      cursor: pointer;
      display: block;
      margin: 0;
    }

    .dropzone-content {
      display: flex;
      flex-direction: column;
      align-items: center;
    }

    .dropzone-text {
      font-size: 0.9rem;
      color: var(--gray-700);
    }

    .dropzone-hint {
      font-size: 0.75rem;
      color: var(--gray-500);
      display: flex;
      flex-wrap: wrap;
      justify-content: center;
      align-items: center;
      gap: 0.25rem;
    }
  }

  /* File Lists */
  .uploaded-files {
    margin-top: 0.5rem;
  }

  .file-group {
    background: #fff;
    border: 1px solid var(--gray-200);
    border-radius: var(--border-radius);
    overflow: hidden;
  }

  .file-group-header {
    display: flex;
    align-items: center;
    padding: 10px 14px;
    background: var(--gray-50);
    border-bottom: 1px solid var(--gray-200);
  }

  .file-group-title {
    font-size: 0.75rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--gray-600);
  }

  .file-list {
    padding: 8px;
  }

  .file-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 12px;
    background: #fff;
    border: 1px solid var(--gray-200);
    border-radius: var(--border-radius);
    margin-bottom: 6px;
    transition: all 0.2s ease;

    &:last-child {
      margin-bottom: 0;
    }

    &:hover {
      border-color: var(--gray-300);
      background: #fafaf9;
    }

    &.recording {
      border-left: 3px solid var(--orange);
      cursor: grab;

      &:active {
        cursor: grabbing;
      }
    }

    &.document {
      border-left: 3px solid var(--gray-700);
    }

    .filename {
      font-family: var(--font-mono);
      font-size: 0.8rem;
    }
  }

  .uppercase {
    text-transform: uppercase;
  }
  .letter-spacing-05 {
    letter-spacing: 0.05em;
  }

  textarea:focus {
    outline: none;
    box-shadow: none;
  }
</style>
