<script lang="ts">
    import Modal from './Modal.svelte';

    interface Props {
        title?: string;
        message: string;
        confirmText?: string;
        cancelText?: string;
        isOpen: boolean;
        onConfirm: () => void;
        onCancel: () => void;
        isDanger?: boolean;
    }

    let { 
        title = 'Confirmation', 
        message, 
        confirmText = 'Confirm', 
        cancelText = 'Cancel', 
        isOpen, 
        onConfirm, 
        onCancel,
        isDanger = false
    }: Props = $props();
</script>

<Modal {title} {isOpen} onClose={onCancel}>
    <div class="py-2">
        <p class="mb-0" style="font-size: 0.85rem; line-height: 1.5;">{message}</p>
    </div>

    {#snippet footer()}
        <div class="d-flex gap-2 justify-content-end">
            <button class="btn btn-white" onclick={onCancel}>
                {cancelText}
            </button>
            <button class="btn {isDanger ? 'btn-danger' : 'btn-primary'}" onclick={onConfirm}>
                {confirmText}
            </button>
        </div>
    {/snippet}
</Modal>
