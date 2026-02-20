<script lang="ts">
  import type { Snippet } from "svelte";
  import FormLabel from "./FormLabel.svelte";
  import TextInput from "./TextInput.svelte";
  import TextArea from "./TextArea.svelte";
  import SelectInput from "./SelectInput.svelte";

  interface Props {
    label: string;
    id?: string;
    type?: "text" | "textarea" | "select" | "number" | "email" | "password";
    value?: any;
    placeholder?: string;
    required?: boolean;
    disabled?: boolean;
    readonly?: boolean;
    rows?: number;
    options?: Array<{ value: string; label: string; disabled?: boolean }>;
    helpText?: string;
    class?: string;
    error?: string;
  }

  let {
    label,
    id,
    type = "text",
    value = $bindable(""),
    placeholder,
    required = false,
    disabled = false,
    readonly = false,
    rows = 4,
    options = [],
    helpText,
    class: className = "",
    error,
  }: Props = $props();

  let inputId = $derived(id || `input-${Math.random().toString(36).slice(2)}`);
  let classes = $derived(
    `form-field ${className || ""} ${error ? "has-error" : ""}`.trim(),
  );
</script>

<div class={classes}>
  <FormLabel htmlFor={inputId}>
    {#snippet children()}
      {label}
    {/snippet}
  </FormLabel>

  {#if type === "textarea"}
    <TextArea
      id={inputId}
      bind:value
      {placeholder}
      {required}
      {disabled}
      {readonly}
      {rows}
    />
  {:else if type === "select"}
    <SelectInput id={inputId} bind:value {options} {required} {disabled} />
  {:else}
    <TextInput
      id={inputId}
      bind:value
      {type}
      {placeholder}
      {required}
      {disabled}
      {readonly}
    />
  {/if}

  {#if helpText && !error}
    <div class="form-help-text">{helpText}</div>
  {/if}

  {#if error}
    <div class="form-error-text">{error}</div>
  {/if}
</div>
