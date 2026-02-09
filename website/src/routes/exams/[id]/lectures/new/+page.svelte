<script lang="ts">
    import { onMount } from 'svelte';
    import { page } from '$app/state';
    import { api } from '$lib/api/client';
    import { goto } from '$app/navigation';
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';
    import { Upload, File, Video, CheckCircle2 } from 'lucide-svelte';

    let examId = $derived(page.params.id);
    let exam = $state<any>(null);
    let title = $state('');
    let description = $state('');
    let mediaFile = $state<File | null>(null);
    let documentFile = $state<File | null>(null);
    let uploading = $state(false);
    let progress = $state(0);
    let status = $state('');

    async function handleUpload() {
        if (!title || (!mediaFile && !documentFile)) return;
        
        uploading = true;
        progress = 0;
        
        try {
            const formData = new FormData();
            formData.append('exam_id', examId);
            formData.append('title', title);
            formData.append('description', description);
            
            if (mediaFile) formData.append('media', mediaFile);
            if (documentFile) formData.append('documents', documentFile);

            status = 'Uploading files...';
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
    <Breadcrumb items={[{ label: 'Exams', href: '/exams' }, { label: exam.title, href: `/exams/${examId}` }, { label: 'New Lecture', active: true }]} />

    <h1>Add New Lecture</h1>

    <div class="row">
        <div class="col-md-8">
            <div class="well">
                <form onsubmit={(e) => { e.preventDefault(); handleUpload(); }}>
                    <div class="mb-4">
                        <label for="lectureTitle" class="form-label fw-bold">Lecture Title</label>
                        <input type="text" id="lectureTitle" class="form-control form-control-lg" placeholder="e.g. Chapter 1: Introduction to Cytology" bind:value={title} required />
                    </div>

                    <div class="mb-4">
                        <label for="lectureDescription" class="form-label fw-bold">Description (Optional)</label>
                        <textarea id="lectureDescription" class="form-control" rows="2" bind:value={description}></textarea>
                    </div>

                    <div class="row">
                        <div class="col-md-6 mb-4">
                            <label for="media" class="form-label d-block fw-bold">Lecture Recording</label>
                            <div class="upload-box {mediaFile ? 'has-file' : ''}">
                                <input type="file" id="media" accept="video/*,audio/*" onchange={(e) => mediaFile = e.currentTarget.files?.[0] || null} />
                                <label for="media">
                                    {#if mediaFile}
                                        <CheckCircle2 size={32} class="text-success mb-2" />
                                        <div class="small text-truncate px-2">{mediaFile.name}</div>
                                    {:else}
                                        <Video size={32} class="text-muted mb-2" />
                                        <div>Click to upload audio/video</div>
                                    {/if}
                                </label>
                            </div>
                        </div>

                        <div class="col-md-6 mb-4">
                            <label for="docs" class="form-label d-block fw-bold">Reference Document</label>
                            <div class="upload-box {documentFile ? 'has-file' : ''}">
                                <input type="file" id="docs" accept=".pdf,.pptx,.docx" onchange={(e) => documentFile = e.currentTarget.files?.[0] || null} />
                                <label for="docs">
                                    {#if documentFile}
                                        <CheckCircle2 size={32} class="text-success mb-2" />
                                        <div class="small text-truncate px-2">{documentFile.name}</div>
                                    {:else}
                                        <File size={32} class="text-muted mb-2" />
                                        <div>Click to upload PDF/Slides</div>
                                    {/if}
                                </label>
                            </div>
                        </div>
                    </div>

                    {#if uploading}
                        <div class="mt-4">
                            <div class="progress mb-2" style="height: 10px;">
                                <div class="progress-bar progress-bar-striped progress-bar-animated bg-success" style="width: 100%"></div>
                            </div>
                            <p class="text-center small text-muted">{status}</p>
                        </div>
                    {/if}

                    <div class="text-center mt-4">
                        <button type="submit" class="btn btn-success btn-lg px-5" disabled={uploading}>
                            {uploading ? 'Processing...' : 'Create Lecture'}
                        </button>
                    </div>
                </form>
            </div>
        </div>
        
        <div class="col-md-4">
            <h4>Getting Started</h4>
            <p class="text-muted">Upload your lecture recordings and any relevant slides or PDFs. The system will automatically:</p>
            <ul class="text-muted">
                <li>Create a high-fidelity transcript</li>
                <li>Process your slides for study context</li>
                <li>Prepare your personalized study assistant</li>
                <li>Ready your materials for study tool generation</li>
            </ul>
        </div>
    </div>
{/if}

<style>
    .upload-box {
        border: 2px dashed #dee2e6;
        border-radius: 0.5rem;
        text-align: center;
        padding: 2rem 1rem;
        background: #fff;
        cursor: pointer;
        transition: all 0.2s;
        position: relative;
    }

    .upload-box:hover {
        border-color: var(--primary-color);
        background: #f8f9fa;
    }

    .upload-box.has-file {
        border-color: var(--success-color);
        background: #f0fff4;
    }

    .upload-box input {
        position: absolute;
        width: 100%;
        height: 100%;
        top: 0;
        left: 0;
        opacity: 0;
        cursor: pointer;
    }
</style>
