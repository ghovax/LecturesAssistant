<script lang="ts">
    import { onMount } from 'svelte';
    import { api } from '$lib/api/client';
    import { notifications } from '$lib/stores/notifications.svelte';
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';
    import Tile from '$lib/components/Tile.svelte';
    import ConfirmModal from '$lib/components/ConfirmModal.svelte';
    import { Plus, Trash2 } from 'lucide-svelte';

    let exams = $state<any[]>([]);
    let loading = $state(true);
    let newExamTitle = $state('');
    let newExamLanguage = $state('');
    let showCreate = $state(false);

    // Confirmation Modal State
    let confirmModal = $state({
        isOpen: false,
        title: '',
        message: '',
        onConfirm: () => {},
        isDanger: false
    });

    function showConfirm(options: { title: string, message: string, onConfirm: () => void, isDanger?: boolean }) {
        confirmModal = {
            isOpen: true,
            title: options.title,
            message: options.message,
            onConfirm: () => {
                options.onConfirm();
                confirmModal.isOpen = false;
            },
            isDanger: options.isDanger ?? false
        };
    }

    async function loadExams() {
        loading = true;
        try {
            const data = await api.listExams();
            exams = data ?? [];
        } catch (e) {
            console.error(e);
        } finally {
            loading = false;
        }
    }

    async function deleteExam(id: string) {
        showConfirm({
            title: 'Delete Subject',
            message: 'Are you sure you want to delete this subject? All lessons and study materials within it will be permanently removed.',
            isDanger: true,
            onConfirm: async () => {
                try {
                    await api.request('DELETE', '/exams', { exam_id: id });
                    await loadExams();
                    notifications.success('The subject has been removed.');
                } catch (e: any) {
                    notifications.error(e.message || e);
                }
            }
        });
    }

    let creating = $state(false);

    async function createExam() {
        if (!newExamTitle || creating) return;
        creating = true;
        try {
            await api.createExam({
                title: newExamTitle,
                language: newExamLanguage || undefined
            });
            newExamTitle = '';
            newExamLanguage = '';
            showCreate = false;
            await loadExams();
            notifications.success('Your new subject has been added.');
        } catch (e: any) {
            notifications.error(e.message || e);
        } finally {
            creating = false;
        }
    }

    onMount(loadExams);
</script>

<Breadcrumb items={[{ label: 'My Studies', active: true }]} />

<ConfirmModal 
    isOpen={confirmModal.isOpen}
    title={confirmModal.title}
    message={confirmModal.message}
    isDanger={confirmModal.isDanger}
    onConfirm={confirmModal.onConfirm}
    onCancel={() => confirmModal.isOpen = false}
/>

<header class="page-header">
    <div class="d-flex justify-content-between align-items-center">
        <h1 class="page-title m-0">My Studies</h1>
        <button class="btn btn-primary rounded-0" onclick={() => showCreate = !showCreate}>
            <Plus size={16} /> Add Subject
        </button>
    </div>
</header>

<div class="bg-white border mb-3">
    <div class="standard-header">
        <div class="header-title">
            <span class="header-text">Workspace</span>
        </div>
    </div>

    <div class="linkTiles">
        {#each exams as exam}
            <Tile href="/exams/{exam.id}" icon="" title={exam.title} cost={exam.estimated_cost}>
                {#snippet description()}
                    {exam.description || 'Access your lessons and study materials.'}
                {/snippet}

                {#snippet actions()}
                    <button 
                        class="btn btn-link text-danger p-0 border-0 shadow-none" 
                        onclick={(e) => { e.preventDefault(); e.stopPropagation(); deleteExam(exam.id); }}
                        title="Delete Subject"
                    >
                        <Trash2 size={16} />
                    </button>
                {/snippet}
            </Tile>
        {/each}
    </div>
</div>

{#if showCreate}
    <div class="bg-white border mb-3 shadow-none">
        <div class="standard-header">
            <div class="header-title">
                <span class="header-text">Create a New Subject</span>
            </div>
        </div>
        <div class="p-4">
            <form onsubmit={(e) => { e.preventDefault(); createExam(); }}>
                <div class="mb-4">
                    <label for="examTitle" class="cozy-label">Subject Name</label>
                    <input
                        id="examTitle"
                        type="text"
                        class="form-control cozy-input"
                        placeholder="e.g. History, Science, Mathematics..."
                        bind:value={newExamTitle}
                        required
                    />
                </div>
                <div class="mb-4">
                    <label for="examLanguage" class="cozy-label">Language (Optional)</label>
                    <select id="examLanguage" class="form-select cozy-input" bind:value={newExamLanguage}>
                        <option value="">Default (from settings)</option>
                        <option value="en-US">English (US)</option>
                        <option value="it-IT">Italian</option>
                        <option value="ja-JP">Japanese</option>
                        <option value="es-ES">Spanish</option>
                        <option value="fr-FR">French</option>
                        <option value="de-DE">German</option>
                        <option value="zh-CN">Chinese (Simplified)</option>
                        <option value="pt-BR">Portuguese (Brazilian)</option>
                    </select>
                    <div class="form-text small">Lectures will inherit this language for transcription and document processing.</div>
                </div>
                <button type="submit" class="btn btn-success btn-sm px-4 rounded-0" disabled={creating}>
                    {#if creating}
                        <span class="spinner-border spinner-border-sm me-2" role="status"></span>
                    {/if}
                    Create Subject
                </button>
            </form>
        </div>
    </div>
{/if}

{#if loading && exams.length === 0}
    <div class="p-5 text-center">
        <div class="d-flex flex-column align-items-center gap-3">
            <div class="village-spinner mx-auto" role="status"></div>
            <p class="text-muted mb-0">Loading your studies...</p>
        </div>
    </div>
{/if}

<style lang="scss">
    .linkTiles {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
        gap: 0;
        background: transparent;
        overflow: hidden;
        
        :global(.tile-wrapper) {
            width: 100%;
            
            :global(a), :global(button) {
                width: 100%;
            }
        }
    }
</style>
