<script lang="ts">
    import { onMount } from 'svelte';
    import { page } from '$app/state';
    import { api } from '$lib/api/client';
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';
    import Tile from '$lib/components/Tile.svelte';
    import { Plus, MessageCircle, FileText, Video, Trash2, ExternalLink } from 'lucide-svelte';

    let examId = $derived(page.params.id);
    let exam = $state<any>(null);
    let lectures = $state<any[]>([]);
    let tools = $state<any[]>([]);
    let loading = $state(true);

    async function loadData() {
        loading = true;
        try {
            const [examData, lecturesData, toolsData] = await Promise.all([
                api.getExam(examId),
                api.listLectures(examId),
                api.listTools(examId)
            ]);
            exam = examData;
            lectures = lecturesData ?? [];
            tools = toolsData ?? [];
        } catch (e) {
            console.error(e);
        } finally {
            loading = false;
        }
    }

    async function deleteLecture(id: string) {
        if (!confirm('Are you sure you want to delete this lecture?')) return;
        try {
            await api.deleteLecture(id, examId);
            await loadData();
        } catch (e) {
            alert(e);
        }
    }

    onMount(loadData);
</script>

{#if exam}
    <Breadcrumb items={[{ label: 'My Studies', href: '/exams' }, { label: exam.title, active: true }]} />

    <div class="d-flex justify-content-between align-items-center mb-3">
        <h2>{exam.title}</h2>
        <a href="/exams/{examId}/chat" class="btn btn-primary">
            <span class="glyphicon me-1"><MessageCircle size={16} /></span> Study Chat
        </a>
    </div>

    <div class="container-fluid p-0">
        <div class="row">
            <!-- Sidebar / Tile Style for Tools -->
            <div class="col-lg-3 col-md-4 order-md-2">
                <h3>Study Tools</h3>
                {#if tools.length === 0}
                    <div class="well text-center p-3">
                        <p class="small text-muted m-0">No tools generated yet.</p>
                    </div>
                {:else}
                    <div class="linkTiles tileSizeMd">
                        {#each tools as tool}
                            <Tile href="/exams/{examId}/tools/{tool.id}" 
                                  icon={tool.type === 'guide' ? '案' : (tool.type === 'flashcard' ? '札' : '問')} 
                                  title={tool.title}>
                                {#snippet description()}
                                    <span class="text-uppercase">{tool.type}</span>
                                {/snippet}
                            </Tile>
                        {/each}
                    </div>
                {/if}
            </div>

            <!-- Main Content / Tile Style for Lectures -->
            <div class="col-lg-9 col-md-8 order-md-1">
                <div class="d-flex justify-content-between align-items-center mb-3">
                    <h3>Lectures</h3>
                    <a href="/exams/{examId}/lectures/new" class="btn btn-success">
                        <span class="glyphicon me-1"><Plus size={16} /></span> Add Lecture
                    </a>
                </div>

                {#if lectures.length === 0}
                    <div class="well text-center p-4">
                        <p>No lectures added yet. Click "Add Lecture" to begin.</p>
                    </div>
                {:else}
                    <div class="linkTiles tileSizeMd">
                        {#each lectures as lecture}
                            <Tile href="/exams/{examId}/lectures/{lecture.id}" icon={lecture.status === 'ready' ? '講' : '作'} title={lecture.title}>
                                {#snippet description()}
                                    {lecture.description || 'No description provided.'}
                                {/snippet}
                                
                                <button 
                                    class="btn btn-link text-danger p-0 position-absolute" 
                                    style="top: 0.5rem; right: 0.5rem; z-index: 10;"
                                    onclick={(e) => { e.preventDefault(); e.stopPropagation(); deleteLecture(lecture.id); }}
                                    title="Delete Lecture"
                                >
                                    <span class="glyphicon m-0"><Trash2 size={14} /></span>
                                </button>
                            </Tile>
                        {/each}
                    </div>
                {/if}
            </div>
        </div>
    </div>
{:else if loading}
    <div class="text-center p-5">
        <div class="village-spinner mx-auto"></div>
    </div>
{/if}
