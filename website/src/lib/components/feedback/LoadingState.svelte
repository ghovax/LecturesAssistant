<script lang="ts">
  interface Props {
    message?: string;
    size?: "sm" | "md" | "lg";
    class?: string;
  }

  let {
    message = "Loading...",
    size = "md",
    class: className = "",
  }: Props = $props();

  const sizeClasses = {
    sm: "spinner-sm",
    md: "spinner-md",
    lg: "spinner-lg",
  };
</script>

<div class="loading-state {className}">
  <div class="loading-spinner">
    <div class="village-spinner {sizeClasses[size]}" role="status"></div>
  </div>
  {#if message}
    <p class="loading-message">{message}</p>
  {/if}
</div>

<style lang="scss">
  .loading-state {
    padding: 5rem 1.5rem;
    text-align: center;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    width: 100%;
  }

  .loading-spinner {
    display: flex;
    align-items: center;
    justify-content: center;
    margin-bottom: 1rem;
  }

  .spinner-sm {
    width: 1rem;
    height: 1rem;
  }

  .spinner-md {
    width: 1.25rem;
    height: 1.25rem;
  }

  .spinner-lg {
    width: 2rem;
    height: 2rem;
  }

  .loading-message {
    font-size: 1rem;
    color: var(--gray-500);
    margin: 0;
    font-weight: 500;
  }
</style>
