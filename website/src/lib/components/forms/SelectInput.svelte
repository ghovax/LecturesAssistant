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
        value = $bindable(''),
        options = [],
        required = false,
        disabled = false,
        class: className = '',
        onchange,
        onfocus,
        onblur
    }: Props = $props();
</script>

<select
    {id}
    class="cozy-form-select {className}"
    {value}
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

<style lang="scss">
    .cozy-form-select {
        border-radius: 0 !important;
        border: 1px solid var(--gray-300) !important;
        font-family: var(--font-primary) !important;
        font-size: 0.85rem !important;
        padding: 8px 12px !important;
        background: #fff !important;
        transition: all 0.2s ease !important;
        height: 2.25rem !important;
        color: var(--gray-800) !important;
        appearance: none !important;
        background-image: url("data:image/svg+xml,%3csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 16 16'%3e%3cpath fill='none' stroke='%2378716c' stroke-linecap='round' stroke-linejoin='round' stroke-width='2' d='M2 5l6 6 6-6'/%3e%3c/svg%3e") !important;
        background-repeat: no-repeat !important;
        background-position: right 0.75rem center !important;
        background-size: 16px 12px !important;
        padding-right: 2.5rem !important;

        &:focus {
            border-color: var(--orange) !important;
            box-shadow: none !important;
            background: #fafaf9 !important;
            outline: none !important;
            background-image: url("data:image/svg+xml,%3csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 16 16'%3e%3cpath fill='none' stroke='%23ea580c' stroke-linecap='round' stroke-linejoin='round' stroke-width='2' d='M2 5l6 6 6-6'/%3e%3c/svg%3e") !important;
        }

        &:disabled {
            background: var(--gray-100) !important;
            cursor: not-allowed;
            opacity: 0.7;
        }
    }
</style>
