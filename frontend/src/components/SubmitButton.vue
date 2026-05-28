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
    class="bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-6 rounded-lg text-sm transition-all duration-200 shadow-sm shadow-blue-600/20 disabled:opacity-50 disabled:cursor-not-allowed disabled:shadow-none flex items-center gap-2"
  >
    <LoadingSpinner v-if="isLoading" />
    {{ isLoading ? loadingText : normalText }}
  </button>
</template>
