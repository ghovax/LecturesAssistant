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
        <div class="d-flex gap-3">
            <button class="btn btn-white bg-white text-dark border flex-grow-1 rounded-0" onclick={onCancel}>
                {cancelText}
            </button>
            <button class="btn {isDanger ? 'btn-danger' : 'btn-primary'} flex-grow-1 rounded-0" onclick={onConfirm}>
                {confirmText}
            </button>
        </div>
    {/snippet}
</Modal>
