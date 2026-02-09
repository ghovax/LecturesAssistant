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
            Add a new course or subject to your hub.
        {/snippet}
    </Tile>
    {#each exams as exam}
        <Tile href="/exams/{exam.id}" icon="科" title={exam.title}>
            {#snippet description()}
                {exam.description || 'Access your lectures and study tools for this subject.'}
            {/snippet}
        </Tile>
    {/each}
</div>

{#if showCreate}
    <div class="well mb-4 bg-white border shadow-sm p-4">
        <h4 class="mt-0">Create a New Subject</h4>
        <form onsubmit={(e) => { e.preventDefault(); createExam(); }}>
            <div class="input-group dictionary-style">
                <input 
                    type="text" 
                    class="form-control" 
                    placeholder="Enter Subject Title (e.g. History, Science, Mathematics)..." 
                    bind:value={newExamTitle} 
                    required 
                />
                <button type="submit" class="btn btn-primary">
                    <span class="glyphicon m-0"><Plus size={18} /></span>
                </button>
            </div>
        </form>
    </div>
{/if}

{#if loading && exams.length === 0}
    <div class="text-center p-5">
        <div class="village-spinner mx-auto"></div>
    </div>
{/if}
