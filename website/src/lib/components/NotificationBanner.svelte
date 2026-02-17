<script lang="ts">
  import { notifications } from "$lib/stores/notifications.svelte";
  import { X, CheckCircle2, AlertCircle, Info } from "lucide-svelte";
</script>

<div class="notification-container">
  {#each notifications.notifications as n (n.id)}
    <div class="notification-banner {n.type} shadow-lg" role="alert">
      <div class="d-flex align-items-center gap-3">
        <div class="icon d-flex align-items-center">
          {#if n.type === "success"}
            <CheckCircle2 size={18} />
          {:else if n.type === "error"}
            <AlertCircle size={18} />
          {:else}
            <Info size={18} />
          {/if}
        </div>
        <div class="message flex-grow-1 fw-bold small">
          {n.message}
          {#if n.action}
            <button
              class="btn btn-link btn-sm p-0 ms-2 text-decoration-underline fw-bold"
              style="color: inherit; font-size: inherit;"
              onclick={() => {
                n.action?.callback();
                notifications.remove(n.id);
              }}
            >
              {n.action.label}
            </button>
          {/if}
        </div>
        <button
          class="btn-close-custom d-flex align-items-center"
          onclick={() => notifications.remove(n.id)}
        >
          <X size={14} />
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
    max-width: 350px;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    pointer-events: none;
  }

  .notification-banner {
    pointer-events: auto;
    padding: 0.75rem 1rem;
    border-radius: 0; /* Kakimashou style: no rounded corners */
    border-left: 0.25rem solid transparent;
    background: #fff;
    color: #333;
    line-height: 1.2;
    border: none !important;
    box-shadow:
      0 10px 30px rgba(0, 0, 0, 0.15),
      0 4px 10px rgba(0, 0, 0, 0.05);

    &.success {
      border-left-color: #568f27;
      .icon {
        color: #568f27;
      }
    }

    &.error {
      border-left-color: #c9302c;
      .icon {
        color: #c9302c;
      }
    }

    &.info {
      border-left-color: #31b0d5;
      .icon {
        color: #31b0d5;
      }
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
