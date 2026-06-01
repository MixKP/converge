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

  const clearEvents = () => {
    events.value = []
  }
</script>

<template>
  <PageLayout title="Submit Availability" :show-cart="false">
    <div class="availability-container">
      <!-- Teacher Selector -->
      <div class="selector-section">
        <select v-model="selectedTeacherId" class="teacher-select">
          <option :value="null">Select your name</option>
          <option v-for="teacher in teacherStore.teachers" :key="teacher.id" :value="teacher.id">
            {{ teacher.name }}
          </option>
        </select>
        <div v-if="events.length > 0" class="slots-info">
          <span class="slots-count"
            >{{ events.length }} slot{{ events.length > 1 ? 's' : '' }}</span
          >
          <button type="button" class="clear-btn" @click="clearEvents">Clear</button>
        </div>
      </div>

      <!-- Calendar -->
      <div class="calendar-wrapper">
        <div class="calendar-inner" :class="{ 'calendar-inner--disabled': !selectedTeacherId }">
          <Calendar
            v-if="selectedTeacherId"
            v-model="events"
            :editable="true"
            class="availability-calendar"
          />
          <CalendarDisabledOverlay v-else message="Select a teacher first" />
        </div>
      </div>

      <!-- Submit Button -->
      <SubmitButton
        :is-disabled="!canSubmit"
        :is-loading="isLoading"
        normal-text="Submit Availability"
        loading-text="Submitting..."
        @click="handleSubmit"
      />

      <!-- Toast Notifications -->
      <div v-if="successMessage" class="toast toast--success">
        <svg class="toast-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
          />
        </svg>
        {{ successMessage }}
      </div>

      <div v-if="errorMessage" class="toast toast--error">
        <svg class="toast-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z"
          />
        </svg>
        {{ errorMessage }}
        <button class="toast-close" @click="errorMessage = ''">×</button>
      </div>
    </div>
  </PageLayout>
</template>

<style scoped>
  .availability-container {
    display: flex;
    flex-direction: column;
    height: 100%;
    gap: 1rem;
  }

  /* Selector Section */
  .selector-section {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.25rem 0.5rem;
  }

  .teacher-select {
    flex: 1;
    max-width: 280px;
    padding: 0.75rem 1rem;
    font-size: 0.875rem;
    color: var(--text-primary);
    background: var(--bg-card);
    border: 1px solid var(--border-medium);
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.2s ease;
    font-family: 'IBM Plex Sans', sans-serif;
    appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' fill='none' stroke='%233e4c7a' viewBox='0 0 24 24'%3E%3Cpath stroke-linecap='round' stroke-linejoin='round' stroke-width='2' d='M8 10l4 4 4-4'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 0.75rem center;
    background-size: 1rem;
    padding-right: 2.25rem;
  }

  .teacher-select:hover {
    border-color: var(--primary-indigo);
  }

  .teacher-select:focus {
    outline: none;
    border-color: var(--primary-indigo);
    box-shadow: 0 0 0 3px rgba(62, 76, 122, 0.1);
  }

  .slots-info {
    display: flex;
    align-items: center;
    gap: 1rem;
  }

  .slots-count {
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--primary-indigo);
    font-family: 'JetBrains Mono', monospace;
  }

  .clear-btn {
    font-size: 0.75rem;
    font-weight: 500;
    color: var(--accent-coral);
    background: transparent;
    border: none;
    cursor: pointer;
    padding: 0.375rem 0.625rem;
    border-radius: 4px;
    transition: all 0.2s ease;
  }

  .clear-btn:hover {
    background: rgba(201, 109, 93, 0.1);
  }

  /* Calendar */
  .calendar-wrapper {
    flex: 1;
    min-height: 0;
  }

  .calendar-inner {
    height: 100%;
    background: var(--bg-card);
    border-radius: 8px;
    border: 1px solid var(--border-medium);
    box-shadow: var(--shadow-card);
    transition: all 0.4s cubic-bezier(0.34, 1.56, 0.64, 1);
    position: relative;
  }

  .calendar-inner::before {
    content: '';
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    height: 1px;
    background: linear-gradient(
      90deg,
      transparent,
      var(--border-technical) 20%,
      var(--border-technical) 80%,
      transparent
    );
  }

  .calendar-inner--disabled {
    opacity: 0.4;
    filter: grayscale(0.3);
  }

  .availability-calendar {
    height: 100%;
  }

  /* Toast */
  .toast {
    position: fixed;
    bottom: 2rem;
    right: 2rem;
    display: flex;
    align-items: center;
    gap: 0.625rem;
    padding: 0.875rem 1.25rem;
    border-radius: 6px;
    box-shadow: var(--shadow-elevated);
    animation: toast-in 0.4s cubic-bezier(0.34, 1.56, 0.64, 1);
    z-index: 100;
    font-size: 0.875rem;
    font-family: 'IBM Plex Sans', sans-serif;
  }

  .toast--success {
    background: var(--accent-sage);
    color: white;
  }

  .toast--error {
    background: var(--accent-coral);
    color: white;
  }

  .toast-icon {
    width: 1.125rem;
    height: 1.125rem;
    flex-shrink: 0;
  }

  .toast-close {
    margin-left: 0.625rem;
    color: rgba(255, 255, 255, 0.8);
    background: transparent;
    border: none;
    font-size: 1.25rem;
    cursor: pointer;
    width: 1.25rem;
    height: 1.25rem;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  @keyframes toast-in {
    from {
      opacity: 0;
      transform: translateY(16px) scale(0.95);
    }
    to {
      opacity: 1;
      transform: translateY(0) scale(1);
    }
  }

  @media (max-width: 640px) {
    .selector-section {
      flex-direction: column;
      align-items: stretch;
      gap: 0.75rem;
    }

    .teacher-select {
      max-width: 100%;
    }

    .slots-info {
      justify-content: space-between;
    }

    .toast {
      right: 1rem;
      left: 1rem;
    }
  }
</style>
