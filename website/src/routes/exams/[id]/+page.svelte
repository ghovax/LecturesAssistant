<script lang="ts">
    import { onMount, onDestroy } from 'svelte';
    import { page } from '$app/state';
    import { browser } from '$app/environment';
    import { api } from '$lib/api/client';
    import { notifications } from '$lib/stores/notifications.svelte';
    import { goto } from '$app/navigation';
    import {
        Breadcrumb,
        ActionTile,
        VerticalTileList,
        EditModal,
        ConfirmModal,
        PageHeader,
        CardContainer,
        LoadingState,
        EmptyState
    } from '$lib';
    import { Plus, MessageCircle, FileText, Trash2, Edit3 } from 'lucide-svelte';

    let examId = $derived(page.params.id);
    let exam = $state<any>(null);
    let lectures = $state<any[]>([]);
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

        if (socket) {
            socket.close();
        }

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
                    loadData();
                }
            }
        };

        socket.onclose = () => {
            setTimeout(setupWebSocket, 5000);
        };
    }

    async function loadData() {
        loading = true;
        try {
            const [examData, lecturesData, sessionsData] = await Promise.all([
                api.getExam(examId!),
                api.listLectures(examId!),
                api.request('GET', `/chat/sessions?exam_id=${examId}`)
            ]);
            exam = examData;
            lectures = lecturesData ?? [];
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

    <PageHeader title={exam.title} description={exam.description}>
        <div class="d-flex align-items-center gap-3">
            <button
                class="btn btn-link btn-sm text-muted p-0 border-0 shadow-none d-flex align-items-center"
                onclick={() => showEditModal = true}
                title="Edit Subject"
            >
                <Edit3 size={16} />
            </button>
            <a href="/exams/{examId}/lectures/new" class="btn btn-primary rounded-0">
                <Plus size={16} /> Add Lesson
            </a>
            <button class="btn btn-success rounded-0" onclick={createChat}>
                <Plus size={16} /> New Chat
            </button>
        </div>
    </PageHeader>

    <div class="container-fluid p-0">
        <div class="row g-4">
            <!-- Sidebar for Study Chats -->
            <div class="col-lg-auto order-md-2">
                <CardContainer title="Study Chats" fitContent>
                    {#if chatSessions.length > 0}
                        <VerticalTileList direction="vertical">
                            {#each chatSessions as session}
                                <ActionTile
                                    href="/exams/{examId}/chat/{session.id}"
                                    title={session.title || 'Untitled Chat'}
                                    cost={session.estimated_cost}
                                >
                                    {#snippet description()}
                                        Opened {new Date(session.created_at).toLocaleDateString(undefined, { day: 'numeric', month: 'long', year: 'numeric' })}
                                    {/snippet}

                                    {#snippet actions()}
                                        <button
                                            class="btn btn-link text-danger p-0 border-0 shadow-none"
                                            onclick={(e) => { e.preventDefault(); e.stopPropagation(); deleteChat(session.id); }}
                                            title="Delete Conversation"
                                            aria-label="Delete Conversation"
                                        >
                                            <Trash2 size={16} />
                                        </button>
                                    {/snippet}
                                </ActionTile>
                            {/each}
                        </VerticalTileList>
                    {:else}
                        <EmptyState
                            icon={MessageCircle}
                            iconSize={32}
                            title="No chats yet"
                            description="Ask questions across all your lessons."
                        >
                            {#snippet action()}
                                <button class="btn btn-success rounded-0" onclick={createChat}>
                                    <Plus size={14} /> Start New Chat
                                </button>
                            {/snippet}
                        </EmptyState>
                    {/if}
                </CardContainer>
            </div>

            <!-- Main Content for Lessons -->
            <div class="col order-md-1">
                <CardContainer title="Lessons" fitContent>
                    {#if lectures.length > 0}
                        <VerticalTileList>
                            {#each lectures as lecture}
                                <ActionTile
                                    href="/exams/{examId}/lectures/{lecture.id}"
                                    title={lecture.title}
                                    cost={lecture.estimated_cost}
                                >
                                    {#snippet description()}
                                        {lecture.description || 'Access lesson materials and study aids.'}
                                    {/snippet}

                                    {#snippet actions()}
                                        <button
                                            class="btn btn-link text-danger p-0 border-0 shadow-none"
                                            onclick={(e) => { e.preventDefault(); e.stopPropagation(); deleteLecture(lecture.id); }}
                                            title="Delete Lesson"
                                            aria-label="Delete Lesson"
                                        >
                                            <Trash2 size={16} />
                                        </button>
                                    {/snippet}
                                </ActionTile>
                            {/each}
                        </VerticalTileList>
                    {:else}
                        <EmptyState
                            icon={FileText}
                            iconSize={48}
                            title="No lessons yet"
                            description="Add your first lesson by uploading a recording or a PDF document."
                        >
                            {#snippet action()}
                                <a href="/exams/{examId}/lectures/new" class="btn btn-primary rounded-0">
                                    <Plus size={14} /> Add Your First Lesson
                                </a>
                            {/snippet}
                        </EmptyState>
                    {/if}
                </CardContainer>
            </div>
        </div>
    </div>
{:else if loading}
    <LoadingState message="Loading project details..." />
{/if}
