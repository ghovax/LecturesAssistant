<script lang="ts">
  import type { Snippet } from "svelte";

  interface Props {
    htmlFor?: string;
    class?: string;
    children?: Snippet;
  }

  let { htmlFor, class: className = "", children }: Props = $props();
</script>

<label for={htmlFor} class="form-label {className}">
  {#if children}
    {@render children()}
  {/if}
</label>
