<script lang="ts">
    import { notifications } from '$lib/stores/notifications.svelte';
    import { X, CheckCircle2, AlertCircle, Info } from 'lucide-svelte';
</script>

<div class="notification-container">
    {#each notifications.notifications as n (n.id)}
        <div class="notification-banner {n.type} shadow-lg" role="alert">
            <div class="d-flex align-items-start gap-3">
                <div class="icon mt-1">
                    {#if n.type === 'success'}
                        <CheckCircle2 size={20} />
                    {:else if n.type === 'error'}
                        <AlertCircle size={20} />
                    {:else}
                        <Info size={20} />
                    {/if}
                </div>
                <div class="message flex-grow-1">
                    {n.message}
                </div>
                <button class="btn-close-custom" onclick={() => notifications.remove(n.id)}>
                    <X size={16} />
                </button>
            </div>
        </div>
    {/each}
</div>

<style lang="scss">
    .notification-container {
        position: fixed;
        top: 4.5rem; /* Just below the navbar (3.125rem + margin) */
        right: 1.5rem;
        z-index: 9999;
        width: 100%;
        max-width: 400px;
        display: flex;
        flex-direction: column;
        gap: 0.75rem;
        pointer-events: none;
    }

    .notification-banner {
        pointer-events: auto;
        padding: 1rem 1.25rem;
        border-radius: 0; /* Kakimashou style: no rounded corners */
        border-left: 0.25rem solid transparent;
        background: #fff;
        color: #333;
        font-size: 0.95rem;
        line-height: 1.4;
        border: 1px solid #ddd;
        
        &.success {
            border-left-color: #568f27;
            .icon { color: #568f27; }
        }
        
        &.error {
            border-left-color: #c9302c;
            .icon { color: #c9302c; }
        }
        
        &.info {
            border-left-color: #31b0d5;
            .icon { color: #31b0d5; }
        }
    }

    .btn-close-custom {
        background: transparent;
        border: none;
        padding: 0;
        color: #999;
        cursor: pointer;
        &:hover {
            color: #333;
        }
    }
</style>
