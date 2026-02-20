<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "$lib/api/client";
  import { notifications } from "$lib/stores/notifications.svelte";
  import {
    Breadcrumb,
    ActionTile,
    VerticalTileList,
    ConfirmModal,
    CardContainer,
    PageHeader,
    FormField,
    EmptyState,
    LoadingState,
    Modal,
  } from "$lib";
  import { Plus, Trash2 } from "lucide-svelte";

  let exams = $state<any[]>([]);
  let loading = $state(true);
  let newExamTitle = $state("");
  let newExamLanguage = $state("");
  let showCreateModal = $state(false);
  let defaultLanguage = $state("");

  // Confirmation Modal State
  let confirmModal = $state({
    isOpen: false,
    title: "",
    message: "",
    onConfirm: () => {},
    isDanger: false,
  });

  function showConfirm(options: {
    title: string;
    message: string;
    onConfirm: () => void;
    isDanger?: boolean;
  }) {
    confirmModal = {
      isOpen: true,
      title: options.title,
      message: options.message,
      onConfirm: () => {
        options.onConfirm();
        confirmModal.isOpen = false;
      },
      isDanger: options.isDanger ?? false,
    };
  }

  async function loadExams() {
    loading = true;
    try {
      const [data, settings] = await Promise.all([
        api.listExams(),
        api.getSettings(),
      ]);
      exams = data ?? [];
      defaultLanguage = settings?.llm?.language || "en-US";
    } catch (e) {
      console.error(e);
    } finally {
      loading = false;
    }
  }

  async function deleteExam(id: string) {
    showConfirm({
      title: "Delete Subject",
      message:
        "Are you sure you want to delete this subject? All lessons and study materials within it will be permanently removed.",
      isDanger: true,
      onConfirm: async () => {
        try {
          await api.request("DELETE", "/exams", { exam_id: id });
          await loadExams();
          notifications.success("The subject has been removed.");
        } catch (e: any) {
          notifications.error(e.message || e);
        }
      },
    });
  }

  let creating = $state(false);

  function openCreateModal() {
    newExamLanguage = defaultLanguage;
    showCreateModal = true;
  }

  async function createExam() {
    if (!newExamTitle || creating) return;
    creating = true;
    try {
      await api.createExam({
        title: newExamTitle,
        language: newExamLanguage || defaultLanguage || undefined,
      });
      newExamTitle = "";
      newExamLanguage = "";
      showCreateModal = false;
      await loadExams();
      notifications.success("Your new subject has been added.");
    } catch (e: any) {
      notifications.error(e.message || e);
    } finally {
      creating = false;
    }
  }

  onMount(loadExams);

  const languageOptions = [
    { value: "en-US", label: "English (US)" },
    { value: "it-IT", label: "Italian" },
    { value: "ja-JP", label: "Japanese" },
    { value: "es-ES", label: "Spanish" },
    { value: "fr-FR", label: "French" },
    { value: "de-DE", label: "German" },
    { value: "tr-TR", label: "Turkish" },
    { value: "zh-CN", label: "Chinese (Simplified)" },
    { value: "pt-BR", label: "Portuguese (Brazilian)" },
  ];
</script>

<Breadcrumb items={[{ label: "My Studies", active: true }]} />

<ConfirmModal
  isOpen={confirmModal.isOpen}
  title={confirmModal.title}
  message={confirmModal.message}
  isDanger={confirmModal.isDanger}
  onConfirm={confirmModal.onConfirm}
  onCancel={() => (confirmModal.isOpen = false)}
/>

<PageHeader
  title="My Studies"
  description="Access your personal study hub and manage all subjects."
>
  <button
    class="btn btn-primary"
    onclick={openCreateModal}
  >
    <Plus size={16} /> Add Subject
  </button>
</PageHeader>

{#if !loading}
  <CardContainer title="Workspace" fitContent>
    {#if exams.length > 0}
      <VerticalTileList>
        {#each exams as exam}
          <ActionTile
            href="/exams/{exam.id}"
            title={exam.title}
            cost={exam.estimated_cost}
          >
            {#snippet description()}
              {exam.description || "Access your lessons and study materials."}
            {/snippet}

            {#snippet actions()}
              <button
                class="btn btn-link text-danger p-0 border-0 shadow-none"
                onclick={(e) => {
                  e.preventDefault();
                  e.stopPropagation();
                  deleteExam(exam.id);
                }}
                title="Delete Subject"
                aria-label="Delete Subject"
              >
                <Trash2 size={16} />
              </button>
            {/snippet}
          </ActionTile>
        {/each}
      </VerticalTileList>
    {:else}
      <EmptyState
        icon={Plus}
        title="Welcome to your Study Hub"
        description="Get started by creating your first subject. You can then add lessons, upload recordings, and generate AI-powered study guides."
      >
        {#snippet action()}
          <button
            class="btn btn-success"
            onclick={openCreateModal}
          >
            Create My First Subject
          </button>
        {/snippet}
      </EmptyState>
    {/if}
  </CardContainer>
{/if}

<Modal
  title="Create a New Subject"
  isOpen={showCreateModal}
  onClose={() => (showCreateModal = false)}
  maxWidth="550px"
>
  <form
    onsubmit={(e) => {
      e.preventDefault();
      createExam();
    }}
  >
    <FormField
      label="Subject Name"
      id="examTitle"
      type="text"
      bind:value={newExamTitle}
      placeholder="e.g. History, Science, Mathematics..."
      required
    />

    <FormField
      label="Language (Optional)"
      id="examLanguage"
      type="select"
      bind:value={newExamLanguage}
      options={languageOptions}
      helpText="Lectures will inherit this language for transcription and document processing."
    />

    <div class="d-flex justify-content-end mt-4">
      <button
        type="submit"
        class="btn btn-success px-4"
        disabled={creating}
      >
        {#if creating}
          <span class="spinner-border spinner-border-sm me-2" role="status"></span>
        {/if}
        Create Subject
      </button>
    </div>
  </form>
</Modal>

{#if loading && exams.length === 0}
  <LoadingState message="Loading your studies..." />
{/if}
