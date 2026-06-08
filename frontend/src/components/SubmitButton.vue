<script setup lang="ts">
  import LoadingSpinner from './LoadingSpinner.vue'
  withDefaults(
    defineProps<{
      isDisabled: boolean
      isLoading: boolean
      loadingText?: string
      normalText?: string
    }>(),
    {
      loadingText: 'Submitting...',
      normalText: 'Submit',
    }
  )
  defineEmits<{
    click: []
  }>()
</script>

<template>
  <button
    type="submit"
    :disabled="isDisabled"
    :aria-busy="isLoading"
    class="submit-button"
    :class="{ 'submit-button--loading': isLoading }"
    @click="$emit('click')"
  >
    <span class="button-content">
      <LoadingSpinner v-if="isLoading" />
      <span class="button-text">{{ isLoading ? loadingText : normalText }}</span>
    </span>
    <span class="button-shine"></span>
  </button>
</template>

<style scoped>
  .submit-button {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 0.875rem 1.5rem;
    font-family: 'Inter', sans-serif;
    font-size: 0.875rem;
    font-weight: 500;
    color: white;
    background: var(--primary-indigo);
    border: none;
    border-radius: 6px;
    cursor: pointer;
    overflow: hidden;
    transition: all 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
    box-shadow:
      0 2px 8px rgba(62, 76, 122, 0.2),
      inset 0 -1px 0 rgba(0, 0, 0, 0.1);
  }

  .submit-button:hover:not(:disabled) {
    background: var(--primary-indigo-deep);
    transform: translateY(-2px);
    box-shadow:
      0 4px 16px rgba(62, 76, 122, 0.3),
      inset 0 -1px 0 rgba(0, 0, 0, 0.1);
  }

  .submit-button:active:not(:disabled) {
    transform: translateY(0);
    box-shadow:
      0 2px 8px rgba(62, 76, 122, 0.2),
      inset 0 1px 0 rgba(0, 0, 0, 0.1);
  }

  .submit-button:disabled {
    opacity: 0.4;
    cursor: not-allowed;
    transform: none;
    box-shadow: none;
  }

  .submit-button--loading {
    pointer-events: none;
  }

  .button-content {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    position: relative;
    z-index: 1;
  }

  .button-text {
    transition: opacity 0.2s ease;
  }

  .button-shine {
    position: absolute;
    top: 0;
    left: -100%;
    width: 100%;
    height: 100%;
    background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.1), transparent);
    transition: left 0.6s ease;
  }

  .submit-button:hover:not(:disabled) .button-shine {
    left: 100%;
  }

  .submit-button:focus-visible {
    outline: none;
    box-shadow:
      0 0 0 3px rgba(62, 76, 122, 0.3),
      0 2px 8px rgba(62, 76, 122, 0.2);
  }
</style>
