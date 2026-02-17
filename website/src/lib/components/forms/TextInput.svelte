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
  class="cozy-form-input {className}"
  {value}
  {placeholder}
  {required}
  {disabled}
  {readonly}
  {onchange}
  {oninput}
  {onfocus}
  {onblur}
/>

<style lang="scss">
  .cozy-form-input {
    border-radius: 0 !important;
    border: 1px solid var(--gray-300) !important;
    font-family: var(--font-primary) !important;
    font-size: 0.85rem !important;
    padding: 8px 12px !important;
    background: #fff !important;
    transition: all 0.2s ease !important;
    height: 2.25rem !important;
    color: var(--gray-800) !important;
    line-height: 1.5 !important;

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

    /* Hide number input spinners */
    &[type="number"] {
      appearance: none;
      &::-webkit-outer-spin-button,
      &::-webkit-inner-spin-button {
        -webkit-appearance: none;
        margin: 0;
      }
    }
  }
</style>
