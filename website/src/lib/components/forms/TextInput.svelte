<script lang="ts">
  interface Props {
    id?: string;
    value?: string;
    placeholder?: string;
    required?: boolean;
    disabled?: boolean;
    readonly?: boolean;
    type?: "text" | "email" | "password" | "number" | "search" | "url" | "tel";
    class?: string;
    onchange?: (e: Event) => void;
    oninput?: (e: Event) => void;
    onfocus?: (e: FocusEvent) => void;
    onblur?: (e: FocusEvent) => void;
  }

  let {
    id,
    value = $bindable(""),
    placeholder,
    required = false,
    disabled = false,
    readonly = false,
    type = "text",
    class: className = "",
    onchange,
    oninput,
    onfocus,
    onblur,
  }: Props = $props();
</script>

<input
  {id}
  {type}
  class="form-control cozy-input {className}"
  bind:value
  {placeholder}
  {required}
  {disabled}
  {readonly}
  {onchange}
  {oninput}
  {onfocus}
  {onblur}
/>
