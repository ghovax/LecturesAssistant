<script lang="ts">
    import Modal from './Modal.svelte';

    interface Props {
        title: string;
        initialTitle: string;
        initialDescription: string;
        onConfirm: (title: string, description: string) => void;
        onCancel: () => void;
    }

    let { title, initialTitle, initialDescription, onConfirm, onCancel }: Props = $props();

    let newTitle = $state(initialTitle);
    let newDescription = $state(initialDescription);
</script>

<Modal 
    {title} 
    isOpen={true} 
    onClose={onCancel}
>
    <div class="mb-4">
        <label class="form-label" for="edit-title">Title</label>
        <input 
            id="edit-title"
            type="text" 
            class="form-control rounded-0 border shadow-none" 
            bind:value={newTitle}
            autofocus
        />
    </div>

    <div class="mb-0">
        <label class="form-label" for="edit-desc">Description</label>
        <textarea 
            id="edit-desc"
            class="form-control rounded-0 border shadow-none" 
            rows="4" 
            bind:value={newDescription}
        ></textarea>
    </div>

    {#snippet footer()}
        <button class="btn btn-success w-100 rounded-0" onclick={() => onConfirm(newTitle, newDescription)}>
            Save Changes
        </button>
    {/snippet}
</Modal>
