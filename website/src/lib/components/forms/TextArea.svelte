<script lang="ts">
  interface Props {
    id?: string;
    value?: string;
    placeholder?: string;
    required?: boolean;
    disabled?: boolean;
    readonly?: boolean;
    rows?: number;
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
    rows = 4,
    class: className = "",
    onchange,
    oninput,
    onfocus,
    onblur,
  }: Props = $props();
</script>

<textarea
  {id}
  class="form-control cozy-input {className}"
  bind:value
  {placeholder}
  {required}
  {disabled}
  {readonly}
  {rows}
  {onchange}
  {oninput}
  {onfocus}
  {onblur}
></textarea>
