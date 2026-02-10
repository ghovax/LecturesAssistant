<script lang="ts">
    import { X } from 'lucide-svelte';

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

<div class="modal fade show d-block" tabindex="-1" style="background: rgba(0,0,0,0.4); backdrop-filter: blur(2px);">
    <div class="modal-dialog modal-dialog-centered">
        <div class="modal-content border-0 rounded-0 shadow-lg">
            <div class="px-4 py-3 border-bottom d-flex justify-content-between align-items-center bg-white">
                <div class="d-flex align-items-center gap-2">
                    <span class="glyphicon m-0" style="font-size: 1.25rem; color: #568f27;">変</span>
                    <span class="fw-bold" style="letter-spacing: 0.02em; font-size: 1rem;">{title}</span>
                </div>
                <button class="btn btn-link btn-sm text-muted p-0 d-flex align-items-center shadow-none" onclick={onCancel}><X size={20} /></button>
            </div>
            <div class="modal-body p-4 bg-light">
                <div class="mb-4">
                    <label class="form-label small fw-bold text-muted text-uppercase mb-2" style="letter-spacing: 0.05em;">Title</label>
                    <input 
                        type="text" 
                        class="form-control rounded-0 border shadow-none" 
                        bind:value={newTitle}
                        autofocus
                    />
                </div>

                <div class="mb-0">
                    <label class="form-label small fw-bold text-muted text-uppercase mb-2" style="letter-spacing: 0.05em;">Description</label>
                    <textarea 
                        class="form-control rounded-0 border shadow-none" 
                        rows="4" 
                        bind:value={newDescription}
                    ></textarea>
                </div>
            </div>
            <div class="px-4 py-3 bg-white border-top text-center">
                <button class="btn btn-success w-100 rounded-0" onclick={() => onConfirm(newTitle, newDescription)}>
                    Save Changes
                </button>
            </div>
        </div>
    </div>
</div>
