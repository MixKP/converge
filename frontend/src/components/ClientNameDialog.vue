<script setup lang="ts">
  import { ref, watch, nextTick } from 'vue'

  interface Props {
    isOpen: boolean
    teacherName: string
    isProcessing?: boolean
  }

  const props = withDefaults(defineProps<Props>(), {
    isProcessing: false,
  })

  const clientName = defineModel<string>('clientName', { default: '' })

  const emit = defineEmits<{
    confirm: []
    cancel: []
  }>()

  const inputRef = ref<HTMLInputElement>()

  watch(
    () => props.isOpen,
    (isOpen: boolean) => {
      if (isOpen) {
        nextTick(() => inputRef.value?.focus())
      } else {
        clientName.value = ''
      }
    }
  )

  const handleSubmit = () => {
    if (clientName.value.trim()) {
      emit('confirm')
    }
  }

  const handleCancel = () => {
    emit('cancel')
  }

  const handleKeydown = (e: KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault()
      handleSubmit()
    } else if (e.key === 'Escape') {
      handleCancel()
    }
  }
</script>

<template>
  <Transition
    enter-active-class="transition-opacity duration-200"
    enter-from-class="opacity-0"
    enter-to-class="opacity-100"
    leave-active-class="transition-opacity duration-150"
    leave-from-class="opacity-100"
    leave-to-class="opacity-0"
  >
    <div
      v-if="isOpen"
      class="fixed inset-0 z-50 flex items-center justify-center bg-(--ink-primary)/40 backdrop-blur-sm"
      @click.self="handleCancel"
    >
      <div
        class="bg-(--paper-white) rounded-sm shadow-2xl border border-(--border-strong) p-6 w-full max-w-sm mx-4 transition-all duration-300 ease-out"
        @keydown="handleKeydown"
      >
        <div class="flex items-start gap-4 mb-5">
          <div
            class="shrink-0 w-12 h-12 rounded-sm bg-(--accent-terracotta-soft) flex items-center justify-center border border-(--accent-terracotta)/20"
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
                d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"
              />
            </svg>
          </div>
          <div class="flex-1">
            <h3 class="text-lg font-semibold text-(--ink-primary) mb-1.5 tracking-tight">
              Confirm Booking
            </h3>
            <p class="text-sm text-(--text-secondary) leading-relaxed">
              Enter your name to confirm booking with
              <span class="font-medium text-(--ink-primary)">{{ teacherName }}</span>
            </p>
          </div>
        </div>

        <div class="mb-5">
          <label
            for="client-name-input"
            class="block text-sm font-medium text-(--ink-primary) mb-2"
          >
            Your Name
          </label>
          <input
            id="client-name-input"
            ref="inputRef"
            v-model="clientName"
            type="text"
            placeholder="Enter your full name"
            class="w-full bg-white border border-(--border-strong) rounded-sm px-4 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-(--accent-terracotta)/20 focus:border-(--accent-terracotta) transition-all"
            :disabled="isProcessing"
          />
        </div>

        <div class="flex gap-3 justify-end">
          <button
            type="button"
            :disabled="isProcessing"
            class="px-4 py-2 text-sm font-medium text-(--ink-primary) bg-white border border-(--border-strong) rounded-sm hover:bg-(--paper-cream) focus:ring-2 focus:ring-(--border-strong) transition-all disabled:opacity-50 disabled:cursor-not-allowed"
            @click="handleCancel"
          >
            Cancel
          </button>
          <button
            type="button"
            :disabled="!clientName.trim() || isProcessing"
            class="px-4 py-2 text-sm font-semibold text-white bg-(--accent-terracotta) rounded-sm hover:bg-(--accent-terracotta-dark) focus:ring-2 focus:ring-(--accent-terracotta)/30 transition-all shadow-sm shadow-(--accent-terracotta)/15 tracking-wide disabled:opacity-50 disabled:cursor-not-allowed"
            @click="handleSubmit"
          >
            {{ isProcessing ? 'Confirming...' : 'Confirm Booking' }}
          </button>
        </div>
      </div>
    </div>
  </Transition>
</template>
