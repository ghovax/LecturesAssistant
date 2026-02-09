<script lang="ts">
    import { onMount } from 'svelte';
    import { page } from '$app/state';
    import { api } from '$lib/api/client';
    import { goto } from '$app/navigation';
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';
    import { Upload, File, Video, CheckCircle2, Search, Info, Trash2, X } from 'lucide-svelte';

    let examId = $derived(page.params.id);
    let exam = $state<any>(null);
    let title = $state('');
    let description = $state('');
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
            
            mediaFiles.forEach(file => formData.append('media', file));
            documentFiles.forEach(file => formData.append('documents', file));

            status = 'Processing upload...';
            await api.createLecture(formData);
            
            status = 'Success! Redirecting...';
            goto(`/exams/${examId}`);
        } catch (e: any) {
            alert(e.message);
            uploading = false;
        }
    }

    onMount(async () => {
        exam = await api.getExam(examId);
    });
</script>

{#if exam}
    <Breadcrumb items={[{ label: 'My Studies', href: '/exams' }, { label: exam.title, href: `/exams/${examId}` }, { label: 'Add Lecture', active: true }]} />

    <h2>Add New Lecture</h2>

    <!-- Prominent Search-style Title Input -->
    <form onsubmit={(e) => { e.preventDefault(); handleUpload(); }} class="mb-4">
        <div class="input-group dictionary-style mb-3">
            <input 
                type="text" 
                class="form-control" 
                placeholder="Enter Lecture Title (e.g., Cellular Respiration)..." 
                bind:value={title}
                required
                disabled={uploading}
            />
            <button class="btn btn-primary" type="submit" disabled={uploading || !title || (mediaFiles.length === 0 && documentFiles.length === 0)}>
                <span class="glyphicon m-0"><Upload size={18} /></span>
            </button>
        </div>
    </form>

    <div class="container-fluid p-0">
        <div class="row">
            <!-- Main Content: Description and Files -->
            <div class="col-lg-9 col-md-8 order-md-1">
                <h3>Details & Materials</h3>
                
                <div class="wordBrief mb-4">
                    <div class="bg-light border-start border-4 border-primary p-3 shadow-sm">
                        <div class="small fw-bold text-muted text-uppercase mb-2" style="font-size: 0.7rem; letter-spacing: 0.1em;">Description</div>
                        <textarea 
                            class="form-control bg-transparent border-0 p-0 shadow-none" 
                            rows="3" 
                            placeholder="Add an optional summary of the lecture content..."
                            bind:value={description}
                            disabled={uploading}
                            style="font-size: 1.1rem; line-height: 1.5; resize: none;"
                        ></textarea>
                    </div>
                </div>

                                <div class="row">

                                    <!-- Media Upload -->

                                    <div class="col-lg-6 mb-4">

                                        <div class="char-results">

                                            <div class="well bg-white p-4 border {mediaFiles.length > 0 ? 'border-success' : ''}">

                                                <div class="row align-items-center">

                                                    <div lang="ja" class="col-3 text-center" style="font-size: 2.5rem; line-height: 1; color: #ccc;">

                                                        新

                                                    </div>

                                                    <div class="col-9">

                                                        <h4 class="mt-0 border-0 pt-0">Recordings</h4>

                                                        <p class="small text-muted mb-2">Video or Audio (MP4, MP3, etc.)</p>

                                                        <input 

                                                            type="file" 

                                                            id="media" 

                                                            class="d-none" 

                                                            accept="video/*,audio/*" 

                                                            multiple

                                                            onchange={addMediaFiles} 

                                                            disabled={uploading}

                                                        />

                                                        <label for="media" class="btn btn-outline-secondary btn-sm">

                                                            Select Files

                                                        </label>

                                                    </div>

                                                </div>

                                                

                                                {#if mediaFiles.length > 0}

                                                    <div class="mt-3 border-top pt-2">

                                                        {#each mediaFiles as file, i}

                                                            <div class="d-flex justify-content-between align-items-center mb-1 small bg-light p-1">

                                                                <span class="text-truncate me-2 fw-bold" title={file.name}>{file.name}</span>

                                                                <button class="btn btn-link btn-sm text-danger p-0" onclick={() => removeMedia(i)} disabled={uploading}>

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

                                        <div class="char-results">

                                            <div class="well bg-white p-4 border {documentFiles.length > 0 ? 'border-success' : ''}">

                                                <div class="row align-items-center">

                                                    <div lang="ja" class="col-3 text-center" style="font-size: 2.5rem; line-height: 1; color: #ccc;">

                                                        資

                                                    </div>

                                                    <div class="col-9">

                                                        <h4 class="mt-0 border-0 pt-0">References</h4>

                                                        <p class="small text-muted mb-2">Slides or PDFs (PDF, PPTX, etc.)</p>

                                                        <input 

                                                            type="file" 

                                                            id="docs" 

                                                            class="d-none" 

                                                            accept=".pdf,.pptx,.docx" 

                                                            multiple

                                                            onchange={addDocumentFiles} 

                                                            disabled={uploading}

                                                        />

                                                        <label for="docs" class="btn btn-outline-secondary btn-sm">

                                                            Select Files

                                                        </label>

                                                    </div>

                                                </div>

                

                                                {#if documentFiles.length > 0}

                                                    <div class="mt-3 border-top pt-2">

                                                        {#each documentFiles as file, i}

                                                            <div class="d-flex justify-content-between align-items-center mb-1 small bg-light p-1">

                                                                <span class="text-truncate me-2 fw-bold" title={file.name}>{file.name}</span>

                                                                <button class="btn btn-link btn-sm text-danger p-0" onclick={() => removeDocument(i)} disabled={uploading}>

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
                    <div class="well bg-light text-center p-4 shadow-sm border-success">
                        <div class="village-spinner mx-auto mb-3"></div>
                        <div class="fw-bold text-uppercase small text-success">{status}</div>
                    </div>
                {/if}
            </div>

            <!-- Sidebar: Instructions -->
            <div class="col-lg-3 col-md-4 order-md-2">
                <h3>Instructions</h3>
                <div class="well bg-light small">
                    <p><strong>Step 1:</strong> Enter a descriptive title for this lecture session.</p>
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
