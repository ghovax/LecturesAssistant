<script lang="ts">
  import type { Snippet } from "svelte";
  import ActionTile from "./navigation/ActionTile.svelte";

  interface Props {
    href?: string;
    icon?: string;
    title: string;
    monospaceTitle?: boolean;
    description?: Snippet;
    children?: Snippet;
    actions?: Snippet;
    onclick?: (e: MouseEvent) => void;
    class?: string;
    disabled?: boolean;
    cost?: number;
    width?: string;
    height?: string;
  }

  let {
    href,
    icon,
    title,
    monospaceTitle = false,
    description,
    children,
    actions,
    onclick,
    class: className = "",
    disabled = false,
    cost,
    width = "250px",
    height = "150px",
  }: Props = $props();

  // Note: icon prop is deprecated, use actions slot for icons
  // monospaceTitle is deprecated - titles are no longer monospace by default
</script>

<!-- Tile is now a wrapper around ActionTile for backward compatibility -->
<ActionTile
  {href}
  {title}
  {monospaceTitle}
  {description}
  {children}
  {actions}
  {onclick}
  {disabled}
  {cost}
  {width}
  {height}
  class={className}
/>
<!-- Tile is now a thin wrapper around ActionTile - styles are handled by ActionTile -->
