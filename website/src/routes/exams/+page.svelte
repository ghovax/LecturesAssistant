<script lang="ts">
    import { onMount } from 'svelte';
    import { api } from '$lib/api/client';
    import { notifications } from '$lib/stores/notifications.svelte';
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';
    import Tile from '$lib/components/Tile.svelte';
    import { Plus } from 'lucide-svelte';

    let exams = $state<any[]>([]);
    let loading = $state(true);
    let newExamTitle = $state('');
    let newExamLanguage = $state('');
    let showCreate = $state(false);

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

    async function createExam() {
        if (!newExamTitle) return;
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
        }
    }

    onMount(loadExams);
</script>

<Breadcrumb items={[{ label: 'My Studies', active: true }]} />

<h2>My Studies</h2>

<div class="linkTiles tileSizeMd mb-4">
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
        </Tile>
    {/each}
    
</div>

{#if showCreate}
    <div class="well mb-4 bg-white border shadow-sm p-4">
        <h4 class="mt-0">Create a New Subject</h4>
        <form onsubmit={(e) => { e.preventDefault(); createExam(); }}>
            <div class="mb-3">
                <label for="examTitle" class="form-label small fw-bold">Subject Name</label>
                <input
                    id="examTitle"
                    type="text"
                    class="form-control"
                    placeholder="e.g. History, Science, Mathematics..."
                    bind:value={newExamTitle}
                    required
                />
            </div>
            <div class="mb-3">
                <label for="examLanguage" class="form-label small fw-bold">Language (Optional)</label>
                <select id="examLanguage" class="form-select" bind:value={newExamLanguage}>
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
            <button type="submit" class="btn btn-primary">
                <span class="glyphicon me-1"><Plus size={18} /></span>
                Create Subject
            </button>
        </form>
    </div>
{/if}

{#if loading && exams.length === 0}
    <div class="text-center p-5">
        <div class="village-spinner mx-auto"></div>
    </div>
{/if}
