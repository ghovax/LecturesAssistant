<script lang="ts">
    import { onMount } from 'svelte';
    import { page } from '$app/state';
    import { api } from '$lib/api/client';
    import { notifications } from '$lib/stores/notifications.svelte';
    import { goto } from '$app/navigation';
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';
    import Highlighter from '$lib/components/Highlighter.svelte';
    import { Upload, FileText, Info, X, Music, GripVertical, FileUp } from 'lucide-svelte';

    let examId = $derived(page.params.id);
    let exam = $state<any>(null);
    let title = $state('');
    let description = $state('');
    let language = $state('');
    let mediaFiles = $state<File[]>([]);
    let documentFiles = $state<File[]>([]);
    let uploading = $state(false);
    let status = $state('');
    let isDragging = $state(false);
    let isHovered = $state(false);

    const mediaExtensions = ['mp4', 'mkv', 'mov', 'webm', 'mp3', 'wav', 'm4a', 'flac'];
    const docExtensions = ['pdf', 'pptx', 'docx'];

    function handleFiles(files: FileList | File[]) {
        const selected = Array.from(files);
        const newMedia: File[] = [];
        const newDocs: File[] = [];

        selected.forEach(file => {
            const ext = file.name.split('.').pop()?.toLowerCase() || '';
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

    async function handleUpload() {
        if (!title || (mediaFiles.length === 0 && documentFiles.length === 0)) return;

        uploading = true;

        try {
            const formData = new FormData();
            formData.append('exam_id', examId);
            formData.append('title', title);
            formData.append('description', description);
            if (language) {
                formData.append('language', language);
            }

            mediaFiles.forEach(file => formData.append('media', file));
            documentFiles.forEach(file => formData.append('documents', file));

            status = 'Processing upload...';
            await api.createLecture(formData);

            status = 'Success! Redirecting...';
            notifications.success('The lesson has been added. We are now preparing your materials.');
            goto(`/exams/${examId}`);
        } catch (e: any) {
            notifications.error(e.message || e);
            uploading = false;
        }
    }

    onMount(async () => {
        const [examData, settings] = await Promise.all([
            api.getExam(examId),
            api.getSettings()
        ]);
        exam = examData;

        if (exam?.language) {
            language = exam.language;
        } else if (settings?.llm?.language) {
            language = settings.llm.language;
        }
    });
</script>

{#if exam}
    <Breadcrumb items={[{ label: 'My Studies', href: '/exams' }, { label: exam.title, href: `/exams/${examId}` }, { label: 'Add Lesson', active: true }]} />

    <header class="page-header">
        <h1 class="page-title">Create New Lesson</h1>
    </header>

    <div class="row g-4">
        <!-- Metadata Column -->
        <div class="col-lg-5">
            <div class="bg-white border h-100">
                <div class="standard-header">
                    <div class="header-title">
                        <span class="header-text">Lesson Details</span>
                    </div>
                </div>
                <div class="p-4">
                    <div class="mb-4">
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

                    <div class="mb-4">
                        <label for="lesson-desc" class="cozy-label">Description (Optional)</label>
                        <textarea
                            id="lesson-desc"
                            class="form-control cozy-input"
                            rows="4"
                            placeholder="What is this lesson about?"
                            bind:value={description}
                            disabled={uploading}
                            style="height: auto !important;"
                        ></textarea>
                    </div>

                    <div class="mb-0">
                        <label for="lesson-lang" class="cozy-label">Processing Language</label>
                        <select id="lesson-lang" class="form-select cozy-input" bind:value={language} disabled={uploading}>
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
                        <div class="form-text mt-2 small opacity-75">Transcripts and analysis will use this language.</div>
                    </div>
                </div>
            </div>
        </div>

        <!-- Unified Files Column -->
        <div class="col-lg-7">
            <div class="bg-white border h-100 d-flex flex-column">
                <div class="standard-header">
                    <div class="header-title">
                        <span class="header-text">Lesson Materials</span>
                    </div>
                </div>
                
                <div class="p-4 flex-grow-1">
                    <!-- Dropzone -->
                    <!-- svelte-ignore a11y_no_static_element_interactions -->
                    <!-- svelte-ignore a11y_mouse_events_have_key_events -->
                    <div 
                        class="dropzone-refined mb-4 {isDragging ? 'is-dragging' : ''}"
                        ondragover={(e) => { e.preventDefault(); isDragging = true; }}
                        ondragleave={() => isDragging = false}
                        {onDrop}
                        onmouseenter={() => isHovered = true}
                        onmouseleave={() => isHovered = false}
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
                            <div class="icon-stack mb-3">
                                <FileUp size={40} class="text-orange" />
                            </div>
                            <div class="text-center">
                                <Highlighter show={isHovered || isDragging} padding={2} offsetY={2}>
                                    <span class="fw-bold h6 mb-1 d-block text-dark">Click or drag files here</span>
                                </Highlighter>
                                <p class="text-muted mb-0 mt-2 small px-4">
                                    Recordings (MP4, MP3, WAV...) and Reference Materials (PDF, PPTX, DOCX)
                                </p>
                            </div>
                        </label>
                    </div>

                    {#if mediaFiles.length > 0 || documentFiles.length > 0}
                        <div class="file-list">
                            <!-- Group: Recordings -->
                            {#if mediaFiles.length > 0}
                                <div class="mb-4">
                                    <div class="cozy-label mb-2 opacity-75" style="font-size: 0.65rem;">Recordings (Ordered)</div>
                                    {#each mediaFiles as file, i}
                                        <div 
                                            class="file-item recording"
                                            draggable={!uploading}
                                            ondragstart={(e: DragEvent) => !uploading && e.dataTransfer?.setData('text/plain', i.toString())}
                                            ondragover={(e: DragEvent) => { e.preventDefault(); if(!uploading && e.currentTarget instanceof HTMLElement) e.currentTarget.style.borderTop = '2px solid var(--orange)'; }}
                                            ondragleave={(e: DragEvent) => { if(e.currentTarget instanceof HTMLElement) e.currentTarget.style.borderTop = ''; }}
                                            ondrop={(e: DragEvent) => {
                                                e.preventDefault();
                                                if(e.currentTarget instanceof HTMLElement) e.currentTarget.style.borderTop = '';
                                                if (uploading) return;
                                                const fromIndex = parseInt(e.dataTransfer?.getData('text/plain') || '-1');
                                                if (fromIndex !== -1 && fromIndex !== i) {
                                                    const files = [...mediaFiles];
                                                    const [moved] = files.splice(fromIndex, 1);
                                                    files.splice(i, 0, moved);
                                                    mediaFiles = files;
                                                }
                                            }}
                                        >
                                            <div class="d-flex align-items-center gap-3 overflow-hidden">
                                                <GripVertical size={14} class="text-muted flex-shrink-0" />
                                                <Music size={16} class="text-orange flex-shrink-0" />
                                                <span class="text-truncate small fw-bold" title={file.name}>{file.name}</span>
                                            </div>
                                            <button class="btn btn-link btn-sm text-danger p-0 border-0 shadow-none ms-2" onclick={() => removeMedia(i)} disabled={uploading}>
                                                <X size={14} />
                                            </button>
                                        </div>
                                    {/each}
                                </div>
                            {/if}

                            <!-- Group: References -->
                            {#if documentFiles.length > 0}
                                <div>
                                    <div class="cozy-label mb-2 opacity-75" style="font-size: 0.65rem;">Reference Documents</div>
                                    {#each documentFiles as file, i}
                                        <div class="file-item reference">
                                            <div class="d-flex align-items-center gap-3 overflow-hidden">
                                                <div style="width: 14px;"></div> <!-- Spacer to align with media grip -->
                                                <FileText size={16} class="text-primary flex-shrink-0" />
                                                <span class="text-truncate small fw-bold" title={file.name}>{file.name}</span>
                                            </div>
                                            <button class="btn btn-link btn-sm text-danger p-0 border-0 shadow-none ms-2" onclick={() => removeDocument(i)} disabled={uploading}>
                                                <X size={14} />
                                            </button>
                                        </div>
                                    {/each}
                                </div>
                            {/if}
                        </div>
                    {:else}
                        <div class="empty-state-files text-center py-5 opacity-25">
                            <Info size={48} class="mb-3 mx-auto" />
                            <p class="fw-medium mb-0">No files selected yet.</p>
                        </div>
                    {/if}
                </div>
            </div>
        </div>
    </div>

    <!-- Action Section -->
    <div class="mt-5 pb-5">
        {#if uploading}
            <div class="text-center">
                <div class="village-spinner mx-auto mb-3"></div>
                <p class="text-muted fw-bold small uppercase letter-spacing-05">{status}</p>
            </div>
        {:else}
            <div class="d-flex flex-column align-items-center gap-3">
                <button 
                    class="btn btn-success btn-lg px-5 rounded-0" 
                    onclick={handleUpload}
                    disabled={!title || (mediaFiles.length === 0 && documentFiles.length === 0)}
                >
                    <Upload size={18} />
                    Start Processing Lesson
                </button>
                <div class="d-flex align-items-center text-muted gap-2 small opacity-75">
                    <Info size={14} />
                    <span>Multiple files will be combined into a single learning experience.</span>
                </div>
            </div>
        {/if}
    </div>
{/if}

<style lang="scss">
    .dropzone-refined {
        border: 1px solid var(--gray-300);
        background: #fff;
        padding: 60px 20px;
        transition: all 0.2s ease;
        position: relative;
        cursor: pointer;

        &:hover {
            border-color: var(--gray-400);
            background: #fafaf9;
        }

        &.is-dragging {
            border-color: var(--orange);
            background: #fffafa;
            outline: 2px solid var(--orange);
            outline-offset: -4px;
        }

        .dropzone-label-refined {
            cursor: pointer;
            display: flex;
            flex-direction: column;
            align-items: center;
            width: 100%;
            margin: 0;
        }
    }

    .icon-stack {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 80px;
        height: 80px;
        background: var(--cream);
        border: 1px solid var(--gray-200);
        margin-bottom: 1rem;
    }

    .file-item {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 12px 16px;
        background: #fff;
        border: 1px solid var(--gray-300);
        margin-bottom: 8px;
        transition: all 0.2s ease;

        &:hover {
            border-color: var(--gray-400);
            background: #fafaf9;
        }

        &.recording {
            cursor: grab;
            border-left: 4px solid var(--orange);
            &:active { cursor: grabbing; }
        }

        &.reference {
            border-left: 4px solid var(--gray-900);
        }
    }

    .uppercase { text-transform: uppercase; }
    .letter-spacing-05 { letter-spacing: 0.05em; }

    textarea:focus {
        outline: none;
        box-shadow: none;
    }
</style>
