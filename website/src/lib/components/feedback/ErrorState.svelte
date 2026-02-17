<script lang="ts">
  import type { Snippet } from "svelte";
  import type { ComponentType } from "svelte";
  import { AlertTriangle } from "lucide-svelte";

  interface Props {
    icon?: ComponentType<any>;
    iconSize?: number;
    title?: string;
    message: string;
    action?: Snippet;
    class?: string;
  }

  let {
    icon: Icon = AlertTriangle,
    iconSize = 48,
    title = "Error",
    message,
    action,
    class: className = "",
  }: Props = $props();
</script>

<div class="error-state {className}">
  <div class="error-state-icon">
    <Icon size={iconSize} />
  </div>
  {#if title}
    <h3 class="error-state-title">{title}</h3>
  {/if}
  <p class="error-state-message">{message}</p>
  {#if action}
    <div class="error-state-action">
      {@render action()}
    </div>
  {/if}
</div>

<style lang="scss">
  .error-state {
    padding: 5rem 1.5rem;
    text-align: center;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    width: 100%;
  }

  .error-state-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    margin-bottom: 1.5rem;
    color: var(--danger);
    opacity: 0.25;
  }

  .error-state-title {
    font-size: 1rem;
    font-weight: 600;
    color: var(--gray-900);
    margin: 0 0 0.5rem 0;
    line-height: 1.2;
  }

  .error-state-message {
    font-size: 0.85rem;
    color: var(--gray-600);
    line-height: 1.5;
    margin: 0 0 1.5rem 0;
    max-width: 400px;
  }

  .error-state-action {
    display: flex;
    gap: 0.75rem;
  }
</style>
