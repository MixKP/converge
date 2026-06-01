<script setup lang="ts">
  import { ref, computed, watch } from 'vue'
  import { useTeacherStore } from '../stores/teacherStore'
  import { availabilityApi } from '../services/availabilityApi'
  import PageLayout from '../components/PageLayout.vue'
  import Calendar from '../components/Calendar.vue'
  import CalendarDisabledOverlay from '../components/CalendarDisabledOverlay.vue'
  import SubmitButton from '../components/SubmitButton.vue'
  import type { EventInput } from '@fullcalendar/core'
  import { generateAvailabilityPayload } from '../utils/calendarHelpers'
  const teacherStore = useTeacherStore()
  const isLoading = ref(false)
  const errorMessage = ref<string>('')
  const successMessage = ref<string>('')
  const events = ref<EventInput[]>([])
  const selectedTeacherId = computed({
    get: () => teacherStore.selectedTeacherId,
    set: (val) => teacherStore.setSelectedTeacherById(val === null ? null : Number(val)),
  })
  const canSubmit = computed(
    () => !!selectedTeacherId.value && !isLoading.value && events.value.length > 0
  )
  watch(selectedTeacherId, () => {
    events.value = []
    errorMessage.value = ''
  })
  const handleSubmit = async () => {
    if (!selectedTeacherId.value || events.value.length === 0 || isLoading.value) return
    isLoading.value = true
    errorMessage.value = ''
    successMessage.value = ''
    try {
      const payload = generateAvailabilityPayload(events.value, selectedTeacherId.value)
      await availabilityApi.submitAvailability(payload)
      events.value = []
      teacherStore.setSelectedTeacherById(null)
      successMessage.value = 'Availability submitted successfully!'
      setTimeout(() => (successMessage.value = ''), 3000)
    } catch (error) {
      errorMessage.value =
        error instanceof Error ? error.message : 'Failed to submit availability. Please try again.'
    } finally {
      isLoading.value = false
    }
  }
</script>
<template>
  <PageLayout title="Submit Availability" :show-cart="false">
    <form class="flex flex-col gap-3 sm:gap-4 flex-1 min-h-0" @submit.prevent="handleSubmit">
      <div
        class="bg-(--paper-white) p-4 sm:p-5 rounded-sm border border-(--border-subtle) shadow-card stagger-in"
      >
        <label class="font-semibold text-(--ink-primary) text-sm block mb-2 sm:mb-3 tracking-tight">
          <span
            class="font-mono text-xs border-2 border-(--ink-primary) bg-(--paper-cream) px-2.5 sm:px-3 py-1 rounded-sm tracking-widest mr-2 sm:mr-3 shadow-badge inline-block"
            >01</span
          >
          Teacher
        </label>
        <select
          v-model="selectedTeacherId"
          class="w-full bg-white border border-(--border-strong) rounded-sm px-3 sm:px-4 py-2.5 sm:py-3 text-sm focus:ring-2 focus:ring-(--accent-terracotta)/10 focus:border-(--accent-terracotta) outline-none transition-all text-(--ink-primary) cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed hover:border-(--text-secondary)"
        >
          <option :value="null" disabled>-- Choose Your Name --</option>
          <option v-for="teacher in teacherStore.teachers" :key="teacher.id" :value="teacher.id">
            {{ teacher.name }}
          </option>
        </select>
      </div>
      <div class="flex-1 flex flex-col min-h-0 stagger-in">
        <div class="flex items-center justify-between mb-3 sm:mb-4 shrink-0">
          <label class="font-semibold text-(--ink-primary) text-sm tracking-tight">
            <span
              class="font-mono text-xs border-2 border-(--ink-primary) bg-(--paper-cream) px-2.5 sm:px-3 py-1 rounded-sm tracking-widest mr-2 sm:mr-3 shadow-badge inline-block"
              >02</span
            >
            Time Slots
          </label>
          <span
            v-if="selectedTeacherId"
            class="text-xs font-medium text-(--accent-terracotta) bg-(--accent-terracotta-soft) px-3 sm:px-4 py-1 sm:py-1.5 rounded-sm border border-(--accent-terracotta)/25 tracking-wide hidden sm:inline-block"
          >
            Click and drag to select
          </span>
        </div>
        <div
          class="relative border border-(--border-subtle) rounded-sm overflow-hidden shadow-card bg-(--paper-white) flex-1 min-h-0"
        >
          <div
            :class="{
              'opacity-45 grayscale-30 pointer-events-none transition-opacity duration-300':
                !selectedTeacherId,
            }"
            class="h-full"
          >
            <Calendar v-model="events" :editable="!!selectedTeacherId" class="h-full" />
          </div>
          <CalendarDisabledOverlay v-if="!selectedTeacherId" message="Select a teacher first" />
        </div>
      </div>
      <div
        class="flex justify-end pt-4 sm:pt-5 border-t border-(--border-subtle) shrink-0 stagger-in"
      >
        <SubmitButton
          :is-disabled="!canSubmit"
          :is-loading="isLoading"
          loading-text="Submitting..."
          normal-text="Submit Availability"
        />
      </div>
    </form>
    <div
      v-if="successMessage"
      class="fixed bottom-4 right-4 bg-green-600 text-white px-4 py-3 rounded-lg shadow-lg flex items-center gap-2 animate-slide-up"
    >
      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
      </svg>
      {{ successMessage }}
    </div>
    <div
      v-if="errorMessage"
      class="fixed bottom-4 right-4 bg-red-600 text-white px-4 py-3 rounded-lg shadow-lg flex items-center gap-2 animate-slide-up"
    >
      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="2"
          d="M6 18L18 6M6 6l12 12"
        />
      </svg>
      {{ errorMessage }}
      <button class="ml-2 text-white/80 hover:text-white" @click="errorMessage = ''">×</button>
    </div>
  </PageLayout>
</template>

<style scoped>
  @keyframes slide-up {
    from {
      opacity: 0;
      transform: translateY(20px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .animate-slide-up {
    animation: slide-up 0.3s ease-out;
  }
</style>
