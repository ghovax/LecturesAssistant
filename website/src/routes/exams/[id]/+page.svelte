<script lang="ts">
    import { onMount } from 'svelte';
    import { page } from '$app/state';
    import { api } from '$lib/api/client';
    import { notifications } from '$lib/stores/notifications.svelte';
    import { goto } from '$app/navigation';
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';
    import Tile from '$lib/components/Tile.svelte';
    import { Plus, MessageCircle, FileText, Video, Trash2, ExternalLink } from 'lucide-svelte';

    let examId = $derived(page.params.id);
    let exam = $state<any>(null);
    let lectures = $state<any[]>([]);
    let tools = $state<any[]>([]);
    let chatSessions = $state<any[]>([]);
    let loading = $state(true);

    async function loadData() {
        loading = true;
        try {
            const [examData, lecturesData, toolsData, sessionsData] = await Promise.all([
                api.getExam(examId),
                api.listLectures(examId),
                api.listTools(examId),
                api.request('GET', `/chat/sessions?exam_id=${examId}`)
            ]);
            exam = examData;
            lectures = lecturesData ?? [];
            tools = toolsData ?? [];
            chatSessions = sessionsData ?? [];
        } catch (e) {
            console.error(e);
        } finally {
            loading = false;
        }
    }

    async function createChat() {
        try {
            const session = await api.request('POST', '/chat/sessions', { 
                exam_id: examId, 
                title: `Study Session ${chatSessions.length + 1}` 
            });
            goto(`/exams/${examId}/chat/${session.id}`);
        } catch (e: any) {
            notifications.error(e.message || e);
        }
    }

    async function deleteLecture(id: string) {
        if (!confirm('Are you sure you want to delete this lecture?')) return;
        try {
            await api.deleteLecture(id, examId);
            await loadData();
            notifications.success('Lecture deleted successfully.');
        } catch (e: any) {
            notifications.error(e.message || e);
        }
    }

    onMount(loadData);
</script>

{#if exam}
    <Breadcrumb items={[{ label: 'My Studies', href: '/exams' }, { label: exam.title, active: true }]} />

    <div class="d-flex justify-content-between align-items-center mb-3">
        <h2>{exam.title}</h2>
        <div class="d-flex gap-2">
            <a href="/exams/{examId}/lectures/new" class="btn btn-primary">
                <span class="glyphicon me-1"><Plus size={16} /></span> Add Lecture
            </a>
        </div>
    </div>

    <div class="container-fluid p-0">
        <div class="row">
            <!-- Sidebar / Tile Style for Tools & Chats -->
            <div class="col-lg-3 col-md-4 order-md-2">
                <div class="mb-2">
                    <h3 class="m-0">Study Chats</h3>
                </div>
                <div class="linkTiles tileSizeMd mb-4">
                    <Tile href="javascript:void(0)" icon="新" title="New Chat" onclick={(e) => { e.preventDefault(); createChat(); }}>
                        {#snippet description()}
                            Start a fresh study session.
                        {/snippet}
                    </Tile>
                    {#each chatSessions as session}
                        <Tile href="/exams/{examId}/chat/{session.id}" icon="談" title={session.title || 'Untitled Chat'}>
                            {#snippet description()}
                                Created {new Date(session.created_at).toLocaleDateString()}
                            {/snippet}
                        </Tile>
                    {/each}
                </div>

                {#if tools.length > 0}
                    <h3>Study Tools</h3>
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
                <div class="mb-3">
                    <h3>Lectures</h3>
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
