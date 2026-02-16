<script lang="ts">
    import { onMount, onDestroy } from 'svelte';
    import { page } from '$app/state';
    import { browser } from '$app/environment';
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
    let socket: WebSocket | null = null;

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

    function setupWebSocket() {
        if (!browser || !examId || examId === 'undefined') return;
        
        const token = localStorage.getItem('session_token');
        const baseUrl = api.getBaseUrl().replace('http', 'ws');
        socket = new WebSocket(`${baseUrl}/socket?session_token=${token}`);
        
        socket.onopen = () => {
            if (examId && examId !== 'undefined') {
                socket?.send(JSON.stringify({
                    type: 'subscribe',
                    channel: `course:${examId}`
                }));
            }
        };

        socket.onmessage = (event) => {
            const data = JSON.parse(event.data);
            if (data.type === 'job:progress') {
                const update = data.payload;
                if (update.status === 'COMPLETED') {
                    // Refresh data if a job finishes
                    loadData();
                }
            }
        };

        socket.onclose = () => {
            setTimeout(setupWebSocket, 5000);
        };
    }

    async function loadData() {
        // ... (rest of loadData)
        loading = true;
        try {
            const [examData, lecturesData, toolsData, sessionsData] = await Promise.all([
                api.getExam(examId!),
                api.listLectures(examId!),
                api.listTools(examId!),
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
                    await api.deleteLecture(id, examId!);
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
            loadData().then(() => {
                if (browser) setupWebSocket();
            });
        }
    });

    onDestroy(() => {
        socket?.close();
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

    <header class="page-header">
        <div class="d-flex justify-content-between align-items-center mb-2">
            <div class="d-flex align-items-center gap-3">
                <h1 class="page-title m-0">{exam.title}</h1>
            </div>
            <div class="d-flex align-items-center gap-3">
                <button class="btn btn-link btn-sm text-muted p-0 border-0 shadow-none d-flex align-items-center" onclick={() => showEditModal = true} title="Edit Subject">
                    <Edit3 size={16} />
                </button>
                <a href="/exams/{examId}/lectures/new" class="btn btn-primary rounded-0">
                    <Plus size={16} /> Add Lesson
                </a>
                <button class="btn btn-success rounded-0" onclick={createChat}>
                    <Plus size={16} /> New Chat
                </button>
            </div>
        </div>
        {#if exam.description}
            <p class="page-description text-muted">{exam.description}</p>
        {/if}
    </header>

    <div class="container-fluid p-0">
        <div class="row g-4">
            <!-- Sidebar / Tile Style for Tools & Chats -->
            <div class="col-lg-auto order-md-2">
                <div class="bg-white border mb-4" style="width: fit-content;">
                    <div class="standard-header">
                        <div class="header-title">
                            <span class="header-text">Study Chats</span>
                        </div>
                    </div>
                    <div class="linkTiles flex-column">
                        {#each chatSessions as session}
                            <Tile href="/exams/{examId}/chat/{session.id}" icon="" title={session.title || 'Untitled Chat'} cost={session.estimated_cost}>
                                {#snippet description()}
                                    Opened {new Date(session.created_at).toLocaleDateString(undefined, { day: 'numeric', month: 'long', year: 'numeric' })}
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
                        {:else}
                            <div class="p-4 text-center text-muted">
                                <MessageCircle size={32} class="mb-3 opacity-25" />
                                <p class="small mb-3">Ask questions across all your lessons.</p>
                                <button class="btn btn-success rounded-0" onclick={createChat}>
                                    <Plus size={14} /> Start New Chat
                                </button>
                            </div>
                        {/each}
                    </div>
                </div>
            </div>

            <!-- Main Content / Tile Style for Lessons -->
            <div class="col order-md-1">
                <div class="bg-white border mb-3" style="width: fit-content; max-width: 100%;">
                    <div class="standard-header">
                        <div class="header-title">
                            <span class="header-text">Lessons</span>
                        </div>
                    </div>

                    <div class="linkTiles">
                        {#each lectures as lecture}
                            <Tile href="/exams/{examId}/lectures/{lecture.id}" icon="" title={lecture.title} cost={lecture.estimated_cost}>
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
                        {:else}
                            <div class="p-5 text-center text-muted w-100">
                                <FileText size={48} class="mb-3 opacity-25" />
                                <h3 class="text-dark h6 mb-2">No lessons yet</h3>
                                <p class="small mb-4">Add your first lesson by uploading a recording or a PDF document.</p>
                                <a href="/exams/{examId}/lectures/new" class="btn btn-primary rounded-0">
                                    <Plus size={14} /> Add Your First Lesson
                                </a>
                            </div>
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

<style lang="scss">
    .page-description {
        font-family: 'Manrope', sans-serif;
        font-size: 0.85rem;
        line-height: 1.6;
        max-width: 600px;
        margin: 0;
    }

    .linkTiles {
        display: flex;
        flex-wrap: wrap;
        gap: 0;
        background: transparent;
        overflow: hidden;
        
        &.flex-column {
            flex-direction: column;
            overflow: visible;
        }

        :global(.tile-wrapper) {
            width: 250px;
        }
    }
</style>
