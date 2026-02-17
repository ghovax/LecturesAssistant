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
  class="cozy-form-textarea {className}"
  {value}
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

<style lang="scss">
  .cozy-form-textarea {
    border-radius: 0 !important;
    border: 1px solid var(--gray-300) !important;
    font-family: var(--font-primary) !important;
    font-size: 0.8rem !important;
    padding: 6px 10px !important;
    background: #fff !important;
    transition: all 0.2s ease !important;
    min-height: 2rem !important;
    height: auto !important;
    color: var(--gray-800) !important;
    line-height: 1.4 !important;
    resize: vertical !important;

    &:focus {
      border-color: var(--orange) !important;
      box-shadow: none !important;
      background: #fafaf9 !important;
      outline: none !important;
    }

    &:disabled {
      background: var(--gray-100) !important;
      cursor: not-allowed;
      opacity: 0.7;
    }

    &::placeholder {
      color: var(--gray-400);
    }
  }
</style>
