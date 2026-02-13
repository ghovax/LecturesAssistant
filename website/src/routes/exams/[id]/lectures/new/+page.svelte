<script lang="ts">
    import { onMount } from 'svelte';
    import { page } from '$app/state';
    import { api } from '$lib/api/client';
    import { notifications } from '$lib/stores/notifications.svelte';
    import { goto } from '$app/navigation';
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';
    import { Upload, File, Video, CheckCircle2, Search, Info, Trash2, X } from 'lucide-svelte';

    let examId = $derived(page.params.id);
    let exam = $state<any>(null);
    let title = $state('');
    let description = $state('');
    let language = $state('');
    let mediaFiles = $state<File[]>([]);
    let documentFiles = $state<File[]>([]);
    let uploading = $state(false);
    let status = $state('');

    function addMediaFiles(e: Event) {
        const input = e.target as HTMLInputElement;
        if (input.files) {
            mediaFiles = [...mediaFiles, ...Array.from(input.files)];
        }
    }

    function addDocumentFiles(e: Event) {
        const input = e.target as HTMLInputElement;
        if (input.files) {
            documentFiles = [...documentFiles, ...Array.from(input.files)];
        }
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

        // Default to exam language, then settings language
        if (exam?.language) {
            language = exam.language;
        } else if (settings?.llm?.language) {
            language = settings.llm.language;
        }
    });
</script>

{#if exam}
    <Breadcrumb items={[{ label: 'My Studies', href: '/exams' }, { label: exam.title, href: `/exams/${examId}` }, { label: 'Add Lesson', active: true }]} />

    <div class="bg-white border mb-3">
        <div class="standard-header">
            <div class="header-title">
                <span class="header-glyph" lang="ja">新</span>
                <span class="header-text">Add New Lesson</span>
            </div>
        </div>

        <div class="p-4">
            <!-- Prominent Search-style Title Input -->
            <form onsubmit={(e) => { e.preventDefault(); handleUpload(); }} class="mb-4">
                <div class="input-group dictionary-style mb-3">
                    <input 
                        type="text" 
                        class="form-control" 
                        placeholder="Enter Lesson Title (e.g., Cellular Respiration)..." 
                        bind:value={title}
                        required
                        disabled={uploading}
                    />
                    <button class="btn btn-success" type="submit" disabled={uploading || !title || (mediaFiles.length === 0 && documentFiles.length === 0)}>
                        {#if uploading}
                            <span class="spinner-border spinner-border-sm" role="status"></span>
                        {:else}
                            <span class="glyphicon m-0"><Upload size={18} /></span>
                        {/if}
                    </button>
                </div>
            </form>

            <div class="container-fluid p-0">
                <div class="row">
                    <!-- Main Content: Description and Files -->
                    <div class="col-lg-9 col-md-8 order-md-1">
                        <div class="bg-white border mb-4">
                            <div class="standard-header">
                                <div class="header-title">
                                    <span class="header-glyph" lang="ja">解</span>
                                    <span class="header-text">Description</span>
                                </div>
                            </div>
                            <div class="p-4 prose">
                                <textarea
                                    class="form-control bg-transparent border-0 p-0 shadow-none"
                                    rows="3"
                                    placeholder="Add an optional summary of the lesson content..."
                                    bind:value={description}
                                    disabled={uploading}
                                    style="font-size: 1.1rem; line-height: 1.5; resize: none;"
                                ></textarea>
                            </div>
                        </div>

                        <div class="bg-white border mb-4">
                            <div class="standard-header">
                                <div class="header-title">
                                    <span class="header-glyph" lang="ja">言</span>
                                    <span class="header-text">Language</span>
                                </div>
                            </div>
                            <div class="p-4">
                                <select class="form-select" bind:value={language} disabled={uploading}>
                                    <option value="">Default ({exam?.language || 'from settings'})</option>
                                    <option value="en-US">English (US)</option>
                                    <option value="it-IT">Italian</option>
                                    <option value="ja-JP">Japanese</option>
                                    <option value="es-ES">Spanish</option>
                                    <option value="fr-FR">French</option>
                                    <option value="de-DE">German</option>
                                    <option value="zh-CN">Chinese (Simplified)</option>
                                    <option value="pt-BR">Portuguese (Brazilian)</option>
                                </select>
                                <div class="form-text small mt-2">Language for transcription and document processing.</div>
                            </div>
                        </div>

                        <div class="row">
                            <!-- Media Upload -->
                            <div class="col-lg-6 mb-4">
                                <div class="bg-white border h-100">
                                    <div class="standard-header">
                                        <div class="header-title">
                                            <span class="header-glyph" lang="ja">音</span>
                                            <span class="header-text">Recordings</span>
                                        </div>
                                    </div>
                                    <div class="p-4">
                                        <p class="small text-muted mb-3">Video or Audio (MP4, MP3, etc.)</p>
                                        <input 
                                            type="file" 
                                            id="media" 
                                            class="d-none" 
                                            accept="video/*,audio/*" 
                                            multiple
                                            onchange={addMediaFiles} 
                                            disabled={uploading}
                                        />
                                        <label for="media" class="btn btn-outline-secondary btn-sm w-100">
                                            Select Files
                                        </label>
                                        
                                        {#if mediaFiles.length > 0}
                                            <div class="mt-3 border-top pt-2">
                                                {#each mediaFiles as file, i}
                                                    <div class="d-flex justify-content-between align-items-center mb-1 small bg-light p-1">
                                                        <span class="text-truncate me-2 fw-bold" title={file.name}>{file.name}</span>
                                                        <button class="btn btn-link btn-sm text-danger p-0 border-0 shadow-none" onclick={() => removeMedia(i)} disabled={uploading}>
                                                            <span class="glyphicon m-0"><X size={14} /></span>
                                                        </button>
                                                    </div>
                                                {/each}
                                            </div>
                                        {/if}
                                    </div>
                                </div>
                            </div>

                            <!-- Document Upload -->
                            <div class="col-lg-6 mb-4">
                                <div class="bg-white border h-100">
                                    <div class="standard-header">
                                        <div class="header-title">
                                            <span class="header-glyph" lang="ja">資</span>
                                            <span class="header-text">References</span>
                                        </div>
                                    </div>
                                    <div class="p-4">
                                        <p class="small text-muted mb-3">Slides or PDFs (PDF, PPTX, etc.)</p>
                                        <input 
                                            type="file" 
                                            id="docs" 
                                            class="d-none" 
                                            accept=".pdf,.pptx,.docx" 
                                            multiple
                                            onchange={addDocumentFiles} 
                                            disabled={uploading}
                                        />
                                        <label for="docs" class="btn btn-outline-secondary btn-sm w-100">
                                            Select Files
                                        </label>

                                        {#if documentFiles.length > 0}
                                            <div class="mt-3 border-top pt-2">
                                                {#each documentFiles as file, i}
                                                    <div class="d-flex justify-content-between align-items-center mb-1 small bg-light p-1">
                                                        <span class="text-truncate me-2 fw-bold" title={file.name}>{file.name}</span>
                                                        <button class="btn btn-link btn-sm text-danger p-0 border-0 shadow-none" onclick={() => removeDocument(i)} disabled={uploading}>
                                                            <span class="glyphicon m-0"><X size={14} /></span>
                                                        </button>
                                                    </div>
                                                {/each}
                                            </div>
                                        {/if}
                                    </div>
                                </div>
                            </div>
                        </div>

                        {#if uploading}
                            <div class="text-center p-4">
                                <div class="d-flex flex-column align-items-center gap-3">
                                    <div class="village-spinner mx-auto" role="status"></div>
                                    <p class="text-muted mb-0">{status}</p>
                                </div>
                            </div>
                        {/if}
                    </div>

                    <!-- Sidebar: Instructions -->
                    <div class="col-lg-3 col-md-4 order-md-2">
                        <div class="bg-white border mb-4">
                            <div class="standard-header">
                                <div class="header-title">
                                    <span class="header-glyph" lang="ja">説</span>
                                    <span class="header-text">Instructions</span>
                                </div>
                            </div>
                            <div class="p-4 small">
                                <p><strong>Step 1:</strong> Enter a descriptive title for this lesson.</p>
                                <p><strong>Step 2:</strong> Provide any number of recordings or reference documents.</p>
                                <p><strong>Step 3:</strong> Click the upload button in the title bar to begin processing.</p>
                                <hr />
                                <div class="d-flex text-muted">
                                    <Info size={16} class="me-2 flex-shrink-0" />
                                    <div>Multiple files will be combined into a single unified learning experience.</div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
{/if}

<style>
    textarea:focus {
        outline: none;
        box-shadow: none;
    }
    
    label {
        cursor: pointer;
    }

    h4 {
        margin-bottom: 0.25rem;
        font-weight: bold;
    }

    .border-dashed {
        border-style: dashed !important;
    }
</style>