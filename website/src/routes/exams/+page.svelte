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

<form onsubmit={(e) => { e.preventDefault(); createExam(); }} class="mb-4">
    <div class="input-group dictionary-style mb-3">
        <input 
            type="text" 
            class="form-control" 
            placeholder="Add New Subject (e.g. History, Science, Mathematics)..." 
            bind:value={newExamTitle} 
            required 
        />
        <button type="submit" class="btn btn-primary">
            <span class="glyphicon m-0"><Plus size={18} /></span>
        </button>
    </div>
</form>

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
                {#snippet description()}
                    {exam.description || 'Access your lectures and study tools for this subject.'}
                {/snippet}
            </Tile>
        {/each}
    </div>
{/if}
