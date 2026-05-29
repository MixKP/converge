<script setup lang="ts">
  import LoadingSpinner from './LoadingSpinner.vue'

  defineProps<{
    isDisabled: boolean
    isLoading: boolean
    loadingText: string
    normalText: string
  }>()
</script>

<template>
  <button
    type="submit"
    :disabled="isDisabled"
    class="bg-(--ink-primary) hover:bg-(--ink-secondary) text-(--paper-white) font-semibold py-2.5 px-6 rounded-sm text-sm transition-all duration-250 shadow-md shadow-(--ink-primary)/12 disabled:opacity-50 disabled:cursor-not-allowed disabled:shadow-none flex items-center gap-2 tracking-wide"
  >
    <LoadingSpinner v-if="isLoading" />
    {{ isLoading ? loadingText : normalText }}
  </button>
</template>
