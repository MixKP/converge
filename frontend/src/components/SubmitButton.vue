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
    :aria-busy="isLoading"
    class="bg-(--ink-primary) hover:bg-(--ink-secondary) text-(--paper-white) font-semibold py-2.5 sm:py-3 px-6 sm:px-8 rounded-sm text-sm transition-all duration-250 shadow-elevated disabled:opacity-50 disabled:cursor-not-allowed disabled:shadow-none flex items-center gap-2 sm:gap-2.5 tracking-wide border-b-2 border-(--ink-secondary)"
  >
    <LoadingSpinner v-if="isLoading" />
    {{ isLoading ? loadingText : normalText }}
  </button>
</template>
