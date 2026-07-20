<script setup lang="ts">
  import { computed, toRef, useId } from 'vue'
  import { useField } from 'vee-validate'

  const props = defineProps<{
    /** vee-validate field name (matches a key in the form's validationSchema). */
    name: string
    /** Class applied to the inner <select> (e.g. "field-select", "mobile-select"). */
    selectClass?: string
    label?: string
    disabled?: boolean
    ariaLabel?: string
    /** v-model value; kept in sync with the field via syncVModel. */
    modelValue?: number | string | null
    /**
     * Gate error display. Parents pass `submitCount > 0` so the field error only
     * appears after a submit attempt, not the moment a sibling field changes.
     * Defaults to true so the component still surfaces errors when unset.
     */
    showError?: boolean
  }>()

  defineEmits<{ 'update:modelValue': [value: number | string | null] }>()

  const { value, errorMessage } = useField<number | string | null>(
    toRef(props, 'name'),
    undefined,
    { syncVModel: true }
  )

  const id = useId()
  const errorId = `${id}-error`
  const displayError = computed(() => (props.showError ?? true) && !!errorMessage.value)
</script>

<template>
  <div class="form-select">
    <label v-if="label" :for="id" class="form-select__label">{{ label }}</label>
    <select
      :id="id"
      v-model="value"
      :class="selectClass"
      :disabled="disabled"
      :aria-label="ariaLabel"
      :aria-invalid="displayError ? true : undefined"
      :aria-describedby="displayError ? errorId : undefined"
    >
      <slot />
    </select>
    <p v-if="displayError" :id="errorId" role="alert" class="form-select__error">
      {{ errorMessage }}
    </p>
  </div>
</template>

<style scoped>
  .form-select {
    display: flex;
    flex-direction: column;
    width: 100%;
  }

  .form-select__label {
    margin-bottom: 0.375rem;
    font-size: 0.8125rem;
    font-weight: 500;
    color: var(--text-secondary);
  }

  .form-select__error {
    margin-top: 0.375rem;
    font-size: 0.75rem;
    font-weight: 500;
    color: var(--accent-coral);
  }
</style>

<!--
  The <select> styles below are unscoped: FormSelect renders the <select>, so the
  duplicated variant classes previously living in each form's scoped block are
  centralized here. Class names are specific enough that global scope is safe.
  The dropdown arrow is an inline SVG data-URI, so `appearance: none` and these
  resets don't map cleanly to Tailwind utilities.
-->
<style>
  /* SubmitAvailability teacher selector */
  .teacher-select {
    width: 100%;
    max-width: 320px;
    padding: 0.875rem 2.75rem 0.875rem 1.25rem;
    font-size: 0.9rem;
    font-family: Inter, sans-serif;
    font-weight: 500;
    color: var(--text-primary);
    background-color: var(--bg-card);
    border: 2px solid var(--border-subtle);
    border-radius: 12px;
    cursor: pointer;
    transition: all 0.3s ease;
    appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' fill='none' stroke='%234A5C52' viewBox='0 0 24 24'%3E%3Cpath stroke-linecap='round' stroke-linejoin='round' stroke-width='2.5' d='M8 10l4 4 4-4'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 1rem center;
    background-size: 1rem;
  }

  .teacher-select:hover {
    border-color: var(--accent-sage);
    background-color: var(--bg-subtle);
    transform: translateY(-1px);
  }

  .teacher-select:focus {
    outline: none;
    border-color: var(--accent-sage);
    box-shadow: 0 0 0 4px rgba(157, 180, 160, 0.2);
    background-color: var(--bg-card);
  }

  @media (max-width: 767px) {
    .teacher-select {
      padding: 0.625rem 2.5rem 0.625rem 0.875rem;
    }
  }

  @media (max-width: 480px) {
    .teacher-select {
      padding: 0.5rem 2.25rem 0.5rem 0.75rem;
      font-size: 0.8rem;
    }
  }

  /* BookingView — mobile layout */
  .mobile-select {
    width: 100%;
    padding: 0.625rem 2rem 0.625rem 0.875rem;
    font-size: 0.875rem;
    font-weight: 400;
    color: var(--text-primary);
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.15s ease;
    font-family: 'Inter', sans-serif;
    appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' fill='none' stroke='%237A8A82' viewBox='0 0 24 24'%3E%3Cpath stroke-linecap='round' stroke-linejoin='round' stroke-width='2' d='M8 10l4 4 4-4'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 0.625rem center;
    background-size: 1rem;
  }

  .mobile-select:hover {
    border-color: var(--primary-navy);
  }

  .mobile-select:focus {
    outline: none;
    border-color: var(--primary-navy);
    box-shadow: 0 0 0 2px rgba(45, 74, 62, 0.1);
  }

  .mobile-select:disabled {
    opacity: 0.5;
    cursor: not-allowed;
    background-color: var(--bg-subtle);
  }

  /* BookingView — tablet layout */
  .tablet-select {
    width: 100%;
    padding: 0.75rem 2.5rem 0.75rem 1rem;
    font-size: 0.9375rem;
    font-weight: 400;
    color: var(--text-primary);
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: 8px;
    cursor: pointer;
    transition: all 0.2s ease;
    font-family: 'Inter', sans-serif;
    appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' fill='none' stroke='%237A8A82' viewBox='0 0 24 24'%3E%3Cpath stroke-linecap='round' stroke-linejoin='round' stroke-width='2' d='M8 10l4 4 4-4'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 0.75rem center;
    background-size: 1rem;
    touch-action: manipulation;
  }

  .tablet-select:hover {
    border-color: var(--primary-navy);
  }

  .tablet-select:focus {
    outline: none;
    border-color: var(--primary-navy);
    box-shadow: 0 0 0 3px rgba(45, 74, 62, 0.1);
  }

  .tablet-select:disabled {
    opacity: 0.5;
    cursor: not-allowed;
    background-color: var(--bg-subtle);
  }

  /* BookingView — desktop field layout */
  .field-select {
    width: 100%;
    padding: 0.75rem 2rem 0.75rem 0.875rem;
    font-size: 0.875rem;
    color: var(--text-primary);
    background-color: var(--bg-cream);
    border: none;
    border-radius: 10px;
    cursor: pointer;
    transition: all 0.2s ease;
    font-family: 'Inter', sans-serif;
    appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' fill='none' stroke='%232D4A3E' viewBox='0 0 24 24'%3E%3Cpath stroke-linecap='round' stroke-linejoin='round' stroke-width='2' d='M8 10l4 4 4-4'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 0.75rem center;
    background-size: 0.875rem;
    touch-action: manipulation;
  }

  .field-select:hover {
    border-color: var(--primary-navy);
  }

  .field-select:focus {
    outline: none;
    border-color: var(--primary-navy);
    box-shadow: 0 0 0 3px rgba(45, 74, 62, 0.1);
  }

  .field-select:focus-visible {
    outline: none;
    border-color: var(--primary-indigo);
    box-shadow: 0 0 0 3px rgba(45, 74, 62, 0.3);
  }

  .field-select:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  /* SmartSuggestionsPanel — subject/branch/teacher selects, per layout */
  .smart-suggestions-panel--mobile__select,
  .smart-suggestions-panel--tablet__select {
    width: 100%;
    padding: 0.625rem 0.875rem;
    font-size: 0.875rem;
    font-weight: 400;
    color: var(--text-primary);
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.15s ease;
    font-family: 'Inter', sans-serif;
    appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' fill='none' stroke='%23585863' viewBox='0 0 24 24'%3E%3Cpath stroke-linecap='round' stroke-linejoin='round' stroke-width='2' d='M8 10l4 4 4-4'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 0.625rem center;
    background-size: 1rem;
    padding-right: 2rem;
    touch-action: manipulation;
  }

  .smart-suggestions-panel--mobile__select:disabled,
  .smart-suggestions-panel--tablet__select:disabled {
    opacity: 0.5;
    cursor: not-allowed;
    background-color: var(--bg-subtle);
  }

  .smart-suggestions-panel--desktop__select {
    width: 100%;
    padding: 0.625rem 0.875rem;
    font-size: 0.8125rem;
    color: var(--text-primary);
    background: var(--bg-card);
    border: 1px solid var(--border-medium);
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.2s ease;
    font-family: 'Inter', sans-serif;
    touch-action: manipulation;
  }

  .smart-suggestions-panel--desktop__select:hover {
    border-color: var(--primary-indigo);
  }

  .smart-suggestions-panel--desktop__select:focus {
    outline: none;
    border-color: var(--primary-indigo);
    box-shadow: 0 0 0 3px rgba(62, 76, 122, 0.1);
  }

  .smart-suggestions-panel--desktop__select:disabled {
    opacity: 0.6;
    cursor: not-allowed;
    background: var(--bg-card);
  }
</style>
