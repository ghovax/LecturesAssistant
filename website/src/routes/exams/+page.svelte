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

<div class="bg-white border mb-5">
    <div class="standard-header">
        <div class="header-title">
            <span class="header-glyph" lang="ja">科</span>
            <span class="header-text">My Studies</span>
        </div>
    </div>

    <div class="linkTiles tileSizeMd p-2">
        <Tile href="javascript:void(0)" icon="新" title="New Subject" onclick={() => showCreate = !showCreate}>
            {#snippet description()}
                Add a new subject to your hub.
            {/snippet}
        </Tile>
        {#each exams as exam}
            <Tile href="/exams/{exam.id}" icon="科" title={exam.title}>
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
    <div class="bg-white border mb-5 shadow-none">
        <div class="standard-header">
            <div class="header-title">
                <span class="header-glyph" lang="ja">新</span>
                <span class="header-text">Create a New Subject</span>
            </div>
        </div>
        <div class="p-4">
            <form onsubmit={(e) => { e.preventDefault(); createExam(); }}>
                <div class="mb-4">
                    <label for="examTitle" class="form-label fw-bold small text-muted text-uppercase mb-2" style="font-size: 0.7rem; letter-spacing: 0.05em;">Subject Name</label>
                    <input
                        id="examTitle"
                        type="text"
                        class="form-control rounded-0 border shadow-none"
                        placeholder="e.g. History, Science, Mathematics..."
                        bind:value={newExamTitle}
                        required
                    />
                </div>
                <div class="mb-5">
                    <label for="examLanguage" class="form-label fw-bold small text-muted text-uppercase mb-2" style="font-size: 0.7rem; letter-spacing: 0.05em;">Language (Optional)</label>
                    <select id="examLanguage" class="form-select rounded-0 border shadow-none" bind:value={newExamLanguage}>
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
                <button type="submit" class="btn btn-primary px-5 rounded-0" disabled={creating}>
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
