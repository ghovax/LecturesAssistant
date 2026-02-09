<script lang="ts">
    import { onMount } from 'svelte';
    import { page } from '$app/state';
    import { api } from '$lib/api/client';
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';
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

    <div class="row mb-5">
        <div class="col-md-8">
            <h1 class="characterHeading">{exam.title}</h1>
            <p class="lead">{exam.description || 'Access your learning materials for this subject.'}</p>
        </div>
        <div class="col-md-4 text-end">
            <a href="/exams/{examId}/chat" class="btn btn-success btn-lg w-100 mb-2">
                <MessageCircle size={20} class="me-2" /> Open Study Chat
            </a>
        </div>
    </div>

    <div class="row">
        <div class="col-md-7">
            <div class="d-flex justify-content-between align-items-center mb-3">
                <h3>Lessons</h3>
                <a href="/exams/{examId}/lectures/new" class="btn btn-outline-primary btn-sm">
                    <Plus size={16} /> Add Lesson
                </a>
            </div>

            {#if lectures.length === 0}
                <div class="well text-center p-4">
                    <p>No lessons added yet.</p>
                </div>
            {:else}
                <div class="list-group">
                    {#each lectures as lecture}
                        <div class="list-group-item d-flex justify-content-between align-items-center">
                            <div>
                                <h5 class="mb-1">{lecture.title}</h5>
                                <small class="text-muted">Status: 
                                    <span class="badge {lecture.status === 'ready' ? 'bg-success' : 'bg-warning'}">
                                        {lecture.status === 'ready' ? 'Ready' : 'Preparing...'}
                                    </span>
                                </small>
                            </div>
                            <div class="btn-group">
                                <a href="/exams/{examId}/lectures/{lecture.id}" class="btn btn-sm btn-outline-secondary">Open</a>
                                <button class="btn btn-sm btn-outline-danger" onclick={() => deleteLecture(lecture.id)}>
                                    <Trash2 size={14} />
                                </button>
                            </div>
                        </div>
                    {/each}
                </div>
            {/if}
        </div>

        <div class="col-md-5">
            <div class="d-flex justify-content-between align-items-center mb-3">
                <h3>Study Kits</h3>
            </div>

            {#if tools.length === 0}
                <div class="well text-center p-4">
                    <p>No study kits generated yet. Prepare a lesson to begin.</p>
                </div>
            {:else}
                <div class="linkTiles tileSizeMd" style="grid-template-columns: 1fr;">
                    {#each tools as tool}
                        <a href="/exams/{examId}/tools/{tool.id}">
                            <div style="font-size: 1.5rem;">
                                {#if tool.type === 'guide'}📝{:else if tool.type === 'flashcard'}🗂️{:else}❓{/if}
                            </div>
                            <p><strong>{tool.title}</strong></p>
                            <small class="text-muted text-uppercase">{tool.type}</small>
                        </a>
                    {/each}
                </div>
            {/if}
        </div>
    </div>
{:else if loading}
    <div class="text-center p-5">
        <div class="village-spinner mx-auto"></div>
    </div>
{/if}
