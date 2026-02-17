<script lang="ts">
  import type { ComponentType } from "svelte";

  interface Props {
    icon: ComponentType<any>;
    size?: number;
    color?: string;
    background?: boolean;
    bgColor?: string;
    class?: string;
  }

  let {
    icon: Icon,
    size = 20,
    color,
    background = false,
    bgColor = "var(--cream)",
    class: className = "",
  }: Props = $props();
</script>

<span class="icon-wrapper {className}" class:has-background={background}>
  {#if background}
    <span class="icon-background" style="background-color: {bgColor};">
      <Icon {size} {color} />
    </span>
  {:else}
    <Icon {size} {color} />
  {/if}
</span>

<style lang="scss">
  .icon-wrapper {
    display: inline-flex;
    align-items: center;
    justify-content: center;

    &.has-background {
      .icon-background {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 40px;
        height: 40px;
        border: 1px solid var(--gray-300);
      }
    }
  }
</style>
