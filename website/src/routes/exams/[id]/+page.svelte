<script lang="ts">
    import { onMount } from 'svelte';
    import { page } from '$app/state';
    import { api } from '$lib/api/client';
    import { notifications } from '$lib/stores/notifications.svelte';
    import { goto } from '$app/navigation';
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';
    import Tile from '$lib/components/Tile.svelte';
    import EditModal from '$lib/components/EditModal.svelte';
    import ConfirmModal from '$lib/components/ConfirmModal.svelte';
    import { Plus, MessageCircle, FileText, Video, Trash2, ExternalLink, Edit3 } from 'lucide-svelte';

    let examId = $derived(page.params.id);
    let exam = $state<any>(null);
    let lectures = $state<any[]>([]);
    let tools = $state<any[]>([]);
    let chatSessions = $state<any[]>([]);
    let loading = $state(true);
    let showEditModal = $state(false);

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
                title: `Conversation ${chatSessions.length + 1}` 
            });
            goto(`/exams/${examId}/chat/${session.id}`);
        } catch (e: any) {
            notifications.error(e.message || e);
        }
    }

    async function deleteChat(id: string) {
        showConfirm({
            title: 'Delete Chat',
            message: 'Are you sure you want to delete this conversation? All messages will be permanently removed.',
            isDanger: true,
            onConfirm: async () => {
                try {
                    await api.request('DELETE', '/chat/sessions', { session_id: id, exam_id: examId });
                    await loadData();
                    notifications.success('The conversation has been removed.');
                } catch (e: any) {
                    notifications.error(e.message || e);
                }
            }
        });
    }

    async function deleteLecture(id: string) {
        showConfirm({
            title: 'Delete Lesson',
            message: 'Are you sure you want to delete this lesson? This action cannot be undone.',
            isDanger: true,
            onConfirm: async () => {
                try {
                    await api.deleteLecture(id, examId);
                    await loadData();
                    notifications.success('The lesson has been removed.');
                } catch (e: any) {
                    notifications.error(e.message || e);
                }
            }
        });
    }

    async function handleEditConfirm(newTitle: string, newDesc: string) {
        if (!newTitle) return;
        try {
            await api.request('PATCH', '/exams', {
                exam_id: examId,
                title: newTitle,
                description: newDesc
            });
            exam.title = newTitle;
            exam.description = newDesc;
            showEditModal = false;
            notifications.success('Project updated.');
        } catch (e: any) {
            notifications.error('Failed to update: ' + (e.message || e));
        }
    }

    $effect(() => {
        if (examId) {
            loadData();
        }
    });
</script>

{#if showEditModal && exam}
    <EditModal 
        title="Edit Project" 
        initialTitle={exam.title} 
        initialDescription={exam.description || ''} 
        onConfirm={handleEditConfirm} 
        onCancel={() => showEditModal = false} 
    />
{/if}

<ConfirmModal 
    isOpen={confirmModal.isOpen}
    title={confirmModal.title}
    message={confirmModal.message}
    isDanger={confirmModal.isDanger}
    onConfirm={confirmModal.onConfirm}
    onCancel={() => confirmModal.isOpen = false}
/>

{#if exam}
    <Breadcrumb items={[{ label: 'My Studies', href: '/exams' }, { label: exam.title, active: true }]} />

    <div class="bg-white border mb-4">
        <div class="standard-header">
            <div class="header-title">
                <span class="header-glyph" lang="ja">科</span>
                <span class="header-text">{exam.title}</span>
            </div>
            <button class="btn btn-link btn-sm text-muted p-0 border-0 shadow-none d-flex align-items-center" onclick={() => showEditModal = true} title="Edit Subject">
                <Edit3 size={18} />
            </button>
        </div>
        {#if exam.description}
            <div class="p-4 prose bg-light bg-opacity-10" style="font-size: 1.1rem;">
                <p class="mb-0">{exam.description}</p>
            </div>
        {/if}
    </div>

    <div class="container-fluid p-0">
        <div class="row">
            <!-- Sidebar / Tile Style for Tools & Chats -->
            <div class="col-lg-3 col-md-4 order-md-2">
                <div class="bg-white border mb-4">
                    <div class="standard-header">
                        <div class="header-title">
                            <span class="header-glyph" lang="ja">談</span>
                            <span class="header-text">Study Chats</span>
                        </div>
                    </div>
                    <div class="linkTiles flex-column p-3">
                        <Tile href="javascript:void(0)" icon="談" title="New Chat" onclick={(e) => { e.preventDefault(); createChat(); }}>
                            {#snippet description()}
                                Start a fresh conversation with me about your studies.
                            {/snippet}
                        </Tile>
                        {#each chatSessions as session}
                            <Tile href="/exams/{examId}/chat/{session.id}" icon="談" title={session.title || 'Untitled Chat'}>
                                {#snippet description()}
                                    Opened {new Date(session.created_at).toLocaleDateString()}
                                {/snippet}

                                {#snippet actions()}
                                    <button 
                                        class="btn btn-link text-danger p-0 border-0 shadow-none" 
                                        onclick={(e) => { e.preventDefault(); e.stopPropagation(); deleteChat(session.id); }}
                                        title="Delete Conversation"
                                    >
                                        <Trash2 size={16} />
                                    </button>
                                {/snippet}
                            </Tile>
                        {/each}
                    </div>
                </div>
            </div>

            <!-- Main Content / Tile Style for Lessons -->
            <div class="col-lg-9 col-md-8 order-md-1">
                <div class="bg-white border mb-3">
                    <div class="standard-header">
                        <div class="header-title">
                            <span class="header-glyph" lang="ja">講</span>
                            <span class="header-text">Lessons</span>
                        </div>
                    </div>

                    <div class="linkTiles tileSizeMd p-2">
                        <Tile href="/exams/{examId}/lectures/new" icon="新" title="Add Lesson">
                            {#snippet description()}
                                Upload recordings and reference materials for a new lesson.
                            {/snippet}
                        </Tile>

                        {#each lectures as lecture}
                            <Tile href="/exams/{examId}/lectures/{lecture.id}" icon={lecture.status === 'ready' ? '講' : '作'} title={lecture.title}>
                                {#snippet description()}
                                    {lecture.description || 'Access lesson materials and study aids.'}
                                {/snippet}
                                
                                {#snippet actions()}
                                    <button 
                                        class="btn btn-link text-danger p-0 border-0 shadow-none" 
                                        onclick={(e) => { e.preventDefault(); e.stopPropagation(); deleteLecture(lecture.id); }}
                                        title="Delete Lesson"
                                    >
                                        <Trash2 size={16} />
                                    </button>
                                {/snippet}
                            </Tile>
                        {/each}
                    </div>
                </div>
            </div>
        </div>
    </div>
{:else if loading}
    <div class="p-5 text-center">
        <div class="d-flex flex-column align-items-center gap-3">
            <div class="village-spinner mx-auto" role="status"></div>
            <p class="text-muted mb-0">Loading project details...</p>
        </div>
    </div>
{/if}
