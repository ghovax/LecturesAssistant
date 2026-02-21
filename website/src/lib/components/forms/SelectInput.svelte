<script lang="ts">
  interface Option {
    value: string;
    label: string;
    disabled?: boolean;
  }

  interface Props {
    id?: string;
    value?: string;
    options: Option[];
    required?: boolean;
    disabled?: boolean;
    class?: string;
    onchange?: (e: Event) => void;
    onfocus?: (e: FocusEvent) => void;
    onblur?: (e: FocusEvent) => void;
  }

  let {
    id,
    value = $bindable(""),
    options = [],
    required = false,
    disabled = false,
    class: className = "",
    onchange,
    onfocus,
    onblur,
  }: Props = $props();
</script>

<select
  {id}
  class="form-select custom-input {className}"
  bind:value
  {required}
  {disabled}
  {onchange}
  {onfocus}
  {onblur}
>
  {#each options as option}
    <option value={option.value} disabled={option.disabled}>
      {option.label}
    </option>
  {/each}
</select>
