<script setup lang="ts">
  import { computed } from 'vue'
  import type { CartItem } from '../composables/useBookingCart'

  interface Props {
    item: CartItem
  }

  const props = defineProps<Props>()

  const emit = defineEmits<{
    remove: [id: number]
  }>()

  const formatDate = (dateStr: string): string => {
    const date = new Date(dateStr)
    return date.toLocaleDateString('en-US', {
      weekday: 'long',
      month: 'short',
      day: 'numeric',
    })
  }

  const formatTime = (dateStr: string): string => {
    const date = new Date(dateStr)
    return date.toLocaleTimeString('en-US', {
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  const dayName = computed(() => formatDate(props.item.start_time))
</script>

<template>
  <div class="bg-white border border-(--border-subtle) rounded-lg p-4 flex items-start gap-4">
    <div
      class="w-12 h-12 rounded-full bg-(--accent-terracotta-soft) flex items-center justify-center shrink-0"
    >
      <svg
        class="w-6 h-6 text-(--accent-terracotta)"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
      >
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="2"
          d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
        />
      </svg>
    </div>

    <div class="flex-1 min-w-0">
      <div class="flex items-start justify-between gap-2">
        <div>
          <h3 class="font-semibold text-(--ink-primary)">{{ item.teacher_name }}</h3>
          <p class="text-sm text-(--text-secondary) mt-1">
            {{ dayName }} • {{ formatTime(item.start_time) }} - {{ formatTime(item.end_time) }}
          </p>
        </div>
        <button
          type="button"
          class="text-red-600 hover:text-red-700 p-1 rounded hover:bg-red-50 transition-all"
          @click="emit('remove', item.id)"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
            />
          </svg>
        </button>
      </div>
      <div class="flex items-center gap-2 mt-2">
        <span
          class="inline-flex items-center px-2 py-0.5 bg-(--accent-terracotta-soft) text-(--accent-terracotta) text-xs font-medium rounded-full"
        >
          60 min
        </span>
        <span class="text-(--text-secondary) text-sm">$50.00</span>
      </div>
    </div>
  </div>
</template>
