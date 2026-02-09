<script lang="ts">
    import { onMount } from 'svelte';
    import { api } from '$lib/api/client';
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';
    import Tile from '$lib/components/Tile.svelte';
    import { Plus } from 'lucide-svelte';

    let exams = $state<any[]>([]);
    let loading = $state(true);
    let newExamTitle = $state('');
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
            await api.createExam({ title: newExamTitle });
            newExamTitle = '';
            showCreate = false;
            await loadExams();
        } catch (e) {
            alert(e);
        }
    }

    onMount(loadExams);
</script>

<Breadcrumb items={[{ label: 'My Studies', active: true }]} />

<h2>My Studies</h2>

<div class="mb-4">
    <button class="btn btn-primary" onclick={() => showCreate = !showCreate}>
        Add New Subject
    </button>
</div>

{#if showCreate}
    <div class="well mb-4">
        <h4>Create a New Subject</h4>
        <form onsubmit={(e) => { e.preventDefault(); createExam(); }} class="row g-3">
            <div class="col-auto flex-grow-1">
                <input type="text" class="form-control" placeholder="Subject Title (e.g. History, Science)" bind:value={newExamTitle} required />
            </div>
            <div class="col-auto">
                <button type="submit" class="btn btn-success">Create</button>
            </div>
        </form>
    </div>
{/if}

{#if loading}
    <div class="text-center p-5">
        <div class="village-spinner mx-auto"></div>
    </div>
{:else if exams.length === 0}
    <p class="text-center p-5">You haven't added any subjects yet. Click the button above to start your first one.</p>
{:else}
    <div class="linkTiles tileSizeMd">
        {#each exams as exam}
            <Tile href="/exams/{exam.id}" icon="科" title={exam.title}>
                {exam.description || 'Access your lectures and study tools for this subject.'}
            </Tile>
        {/each}
    </div>
{/if}
