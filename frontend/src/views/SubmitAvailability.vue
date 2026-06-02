<script setup lang="ts">
  import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
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
  const showKeyboardHint = ref(false)

  const selectedTeacherId = computed({
    get: () => teacherStore.selectedTeacherId,
    set: (val) => teacherStore.setSelectedTeacherById(val === null ? null : Number(val)),
  })

  const selectedTeacher = computed(() =>
    teacherStore.teachers.find((t) => t.id === selectedTeacherId.value)
  )

  const canSubmit = computed(
    () => !!selectedTeacherId.value && !isLoading.value && events.value.length > 0
  )

  const formattedSlots = computed(() => {
    return events.value
      .map((event) => {
        const start = new Date(event.start as string)
        const end = new Date(event.end as string)
        const dayName = start.toLocaleDateString('en-US', { weekday: 'long' })
        const startTime = start.toLocaleTimeString('en-US', {
          hour: '2-digit',
          minute: '2-digit',
          hour12: false,
        })
        const endTime = end.toLocaleTimeString('en-US', {
          hour: '2-digit',
          minute: '2-digit',
          hour12: false,
        })
        return { day: dayName, start: startTime, end: endTime }
      })
      .sort((a, b) => {
        const days = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']
        return days.indexOf(a.day) - days.indexOf(b.day) || a.start.localeCompare(b.start)
      })
  })

  watch(selectedTeacherId, () => {
    events.value = []
    errorMessage.value = ''
  })

  const showConfirm = ref(false)

  const handleSubmit = async () => {
    if (!selectedTeacherId.value || events.value.length === 0 || isLoading.value) return
    showConfirm.value = true
  }

  const confirmSubmit = async () => {
    isLoading.value = true
    errorMessage.value = ''
    successMessage.value = ''

    try {
      const payload = generateAvailabilityPayload(events.value, selectedTeacherId.value!)
      await availabilityApi.submitAvailability(payload)
      events.value = []
      teacherStore.setSelectedTeacherById(null)
      showConfirm.value = false
      successMessage.value = 'Availability submitted successfully!'
      setTimeout(() => (successMessage.value = ''), 3000)
    } catch (error) {
      errorMessage.value =
        error instanceof Error ? error.message : 'Failed to submit availability. Please try again.'
    } finally {
      isLoading.value = false
    }
  }

  const cancelConfirm = () => {
    showConfirm.value = false
  }

  const clearEvents = () => {
    events.value = []
  }

  const handleKeydown = (e: KeyboardEvent) => {
    if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
      e.preventDefault()
      showKeyboardHint.value = true
      setTimeout(() => (showKeyboardHint.value = false), 2000)
    }
    if (e.key === 'Enter' && canSubmit.value) {
      handleSubmit()
    }
  }

  onMounted(() => {
    window.addEventListener('keydown', handleKeydown)
  })

  onUnmounted(() => {
    window.removeEventListener('keydown', handleKeydown)
  })
</script>

<template>
  <PageLayout title="Submit Availability" :show-cart="false">
    <div class="availability-container">
      <!-- Technical Background -->
      <div class="technical-bg" aria-hidden="true">
        <div class="bg-label">SCHEDULE</div>
      </div>

      <!-- Teacher Selector -->
      <div class="selector-section">
        <div v-if="!selectedTeacher" class="teacher-selector-form">
          <label class="form-label">Teacher</label>
          <select v-model="selectedTeacherId" class="teacher-select">
            <option :value="null">Select your name</option>
            <option v-for="teacher in teacherStore.teachers" :key="teacher.id" :value="teacher.id">
              {{ teacher.name }}
            </option>
          </select>
        </div>
        <div v-else class="teacher-display-box">
          <span class="teacher-label">TEACHER</span>
          <span class="teacher-name">{{ selectedTeacher.name }}</span>
          <button type="button" class="change-teacher-btn" @click="selectedTeacherId = null">
            Change
          </button>
        </div>
        <div v-if="events.length > 0" class="slots-info">
          <span class="slots-count"
            >{{ events.length }} slot{{ events.length > 1 ? 's' : '' }}</span
          >
          <button type="button" class="clear-btn" @click="clearEvents">Clear slots</button>
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

          <!-- Inline Guidance -->
          <div v-if="selectedTeacherId && events.length === 0" class="calendar-guidance">
            <span class="guidance-icon">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="1.5"
                  d="M15.042 21.672L13.684 16.6m0 0l-2.51 2.225.569-9.47 5.227 7.917-3.286-.672zM12 2.25V4.5m5.834-.166l-1.591 1.591M20.25 10.5H18M7.757 19.536l-1.59-1.59M6 10.5H3.75m4.007-4.243l-1.59-1.59"
                />
              </svg>
            </span>
            <span class="guidance-text"
              >Click and drag on the calendar to add available time slots</span
            >
          </div>
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

      <!-- Confirmation Dialog -->
      <div v-if="showConfirm" class="confirm-dialog-overlay" @click.self="cancelConfirm">
        <div class="confirm-dialog">
          <h3 class="confirm-title">Confirm Availability</h3>
          <div class="confirm-content">
            <div class="confirm-row">
              <span class="confirm-label">TEACHER</span>
              <span class="confirm-value">{{ selectedTeacher?.name }}</span>
            </div>
            <div class="confirm-row">
              <span class="confirm-label">TIME SLOTS</span>
              <span class="confirm-value"
                >{{ events.length }} slot{{ events.length > 1 ? 's' : '' }}</span
              >
            </div>
            <div class="confirm-slots">
              <div v-for="(slot, index) in formattedSlots" :key="index" class="confirm-slot">
                {{ slot.day }} {{ slot.start }}-{{ slot.end }}
              </div>
            </div>
          </div>
          <div class="confirm-actions">
            <button type="button" class="confirm-btn confirm-btn--cancel" @click="cancelConfirm">
              Cancel
            </button>
            <button type="button" class="confirm-btn confirm-btn--confirm" @click="confirmSubmit">
              Confirm
            </button>
          </div>
        </div>
      </div>

      <!-- Keyboard Hint -->
      <div v-if="showKeyboardHint" class="keyboard-hint">Press <kbd>Enter</kbd> to submit</div>

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
    position: relative;
  }

  /* Technical Background */
  .technical-bg {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    pointer-events: none;
    z-index: 0;
    overflow: hidden;
  }

  .bg-label {
    position: absolute;
    bottom: 10%;
    left: 50%;
    transform: translateX(-50%);
    font-family: 'JetBrains Mono', monospace;
    font-size: 4rem;
    font-weight: 600;
    color: var(--primary-indigo);
    opacity: 0.03;
    letter-spacing: 0.5em;
    white-space: nowrap;
  }

  /* Selector Section */
  .selector-section {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 1rem;
    padding: 0.5rem 0.75rem;
    position: relative;
    z-index: 1;
  }

  /* Teacher Selector Form (before selection) */
  .teacher-selector-form {
    display: flex;
    flex-direction: column;
    gap: 0.375rem;
  }

  .form-label {
    font-family: 'IBM Plex Sans', sans-serif;
    font-size: 0.8125rem;
    font-weight: 500;
    color: var(--text-primary);
  }

  /* Teacher Display Box (after selection - like the image) */
  .teacher-display-box {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 0.5rem;
    padding: 0.75rem 1rem;
    background: var(--bg-card);
    border: 1px solid var(--border-medium);
    border-radius: 6px;
    min-width: 280px;
  }

  .teacher-label {
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.6875rem;
    font-weight: 600;
    letter-spacing: 0.1em;
    color: var(--text-secondary);
    text-transform: uppercase;
  }

  .teacher-name {
    flex: 1;
    font-family: 'IBM Plex Sans', sans-serif;
    font-size: 0.875rem;
    font-weight: 400;
    color: var(--text-primary);
  }

  .change-teacher-btn {
    padding: 0.375rem 0.75rem;
    font-size: 0.75rem;
    font-weight: 500;
    color: var(--primary-indigo);
    background: transparent;
    border: 1px solid var(--border-medium);
    border-radius: 4px;
    cursor: pointer;
    transition: all 0.2s ease;
    font-family: 'IBM Plex Sans', sans-serif;
  }

  .change-teacher-btn:hover {
    background: var(--bg-subtle);
    border-color: var(--primary-indigo);
  }

  /* Summary Card */
  .summary-card {
    background: var(--bg-card);
    border: 1px solid var(--border-medium);
    border-radius: 8px;
    padding: 1rem;
    position: relative;
    z-index: 1;
  }

  .summary-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding-bottom: 0.75rem;
    border-bottom: 1px solid var(--border-subtle);
  }

  .summary-row:not(:last-child) {
    margin-bottom: 0.75rem;
  }

  .summary-label {
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.6875rem;
    font-weight: 600;
    letter-spacing: 0.1em;
    color: var(--text-secondary);
    text-transform: uppercase;
  }

  .summary-value {
    font-family: 'IBM Plex Sans', sans-serif;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-primary);
  }

  /* Slots List */
  .slots-list {
    padding-top: 0.75rem;
  }

  .slots-header {
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.6875rem;
    font-weight: 600;
    letter-spacing: 0.1em;
    color: var(--text-secondary);
    text-transform: uppercase;
    margin-bottom: 0.75rem;
  }

  .slots-items {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .slot-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.5rem 0.75rem;
    background: var(--bg-subtle);
    border-radius: 4px;
    transition: all 0.2s ease;
  }

  .slot-item:hover {
    background: var(--border-subtle);
  }

  .slot-day {
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.8125rem;
    font-weight: 500;
    color: var(--primary-indigo);
  }

  .slot-time {
    font-family: 'IBM Plex Sans', sans-serif;
    font-size: 0.8125rem;
    color: var(--text-secondary);
  }

  .teacher-select {
    flex: 1;
    max-width: 320px;
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
    position: relative;
    z-index: 1;
  }

  .calendar-inner {
    height: 100%;
    background: var(--bg-card);
    border-radius: 8px;
    border: 1px solid var(--border-medium);
    box-shadow: var(--shadow-card);
    transition: all 0.4s cubic-bezier(0.34, 1.56, 0.64, 1);
    position: relative;
    overflow: hidden;
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
    z-index: 1;
  }

  .calendar-inner--disabled {
    opacity: 0.4;
    filter: grayscale(0.3);
  }

  .availability-calendar {
    height: 100%;
    position: relative;
    z-index: 2;
  }

  /* Calendar Guidance */
  .calendar-guidance {
    position: absolute;
    bottom: 1rem;
    left: 50%;
    transform: translateX(-50%);
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem 1rem;
    background: var(--bg-cream);
    border: 1px solid var(--border-subtle);
    border-radius: 6px;
    font-size: 0.75rem;
    color: var(--text-secondary);
    font-family: 'IBM Plex Sans', sans-serif;
    z-index: 3;
    animation: guidance-in 0.4s ease-out;
  }

  .guidance-icon {
    display: flex;
    align-items: center;
    color: var(--primary-indigo);
  }

  .guidance-icon svg {
    width: 1rem;
    height: 1rem;
  }

  .guidance-text {
    font-size: 0.75rem;
  }

  @keyframes guidance-in {
    from {
      opacity: 0;
      transform: translate(-50%, 8px);
    }
    to {
      opacity: 1;
      transform: translate(-50%, 0);
    }
  }

  /* Keyboard Hint */
  .keyboard-hint {
    position: fixed;
    bottom: 6rem;
    right: 2rem;
    padding: 0.5rem 0.75rem;
    background: var(--text-primary);
    color: var(--bg-card);
    border-radius: 4px;
    font-size: 0.75rem;
    font-family: 'IBM Plex Sans', sans-serif;
    z-index: 100;
    animation: hint-in 0.2s ease-out;
  }

  .keyboard-hint kbd {
    font-family: 'JetBrains Mono', monospace;
    background: rgba(255, 255, 255, 0.15);
    padding: 0.125rem 0.375rem;
    border-radius: 3px;
    font-size: 0.6875rem;
  }

  @keyframes hint-in {
    from {
      opacity: 0;
      transform: translateY(4px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
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

  /* Confirmation Dialog */
  .confirm-dialog-overlay {
    position: fixed;
    inset: 0;
    background: rgba(26, 28, 35, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 200;
    animation: fade-in 0.2s ease-out;
  }

  .confirm-dialog {
    background: var(--bg-card);
    border: 1px solid var(--border-medium);
    border-top: 2px solid var(--accent-sage);
    border-radius: 8px;
    padding: 1.5rem;
    max-width: 400px;
    margin: 0 1rem;
    animation: dialog-in 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
  }

  .confirm-title {
    font-size: 1rem;
    font-weight: 600;
    font-family: 'Instrument Sans', sans-serif;
    color: var(--text-primary);
    margin: 0 0 1rem 0;
  }

  .confirm-content {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    margin-bottom: 1.25rem;
  }

  .confirm-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .confirm-label {
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.6875rem;
    font-weight: 600;
    letter-spacing: 0.1em;
    color: var(--text-secondary);
    text-transform: uppercase;
  }

  .confirm-value {
    font-family: 'IBM Plex Sans', sans-serif;
    font-size: 0.875rem;
    color: var(--text-primary);
  }

  .confirm-slots {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    padding: 0.75rem;
    background: var(--bg-subtle);
    border-radius: 4px;
  }

  .confirm-slot {
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.8125rem;
    color: var(--text-primary);
  }

  .confirm-actions {
    display: flex;
    gap: 0.75rem;
    justify-content: flex-end;
  }

  .confirm-btn {
    padding: 0.625rem 1rem;
    font-size: 0.875rem;
    font-weight: 500;
    font-family: 'IBM Plex Sans', sans-serif;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .confirm-btn--cancel {
    color: var(--text-primary);
    background: var(--bg-subtle);
    border: 1px solid var(--border-medium);
  }

  .confirm-btn--cancel:hover {
    background: var(--border-subtle);
  }

  .confirm-btn--confirm {
    color: white;
    background: var(--accent-sage);
    border: none;
  }

  .confirm-btn--confirm:hover {
    filter: brightness(0.9);
  }

  @keyframes dialog-in {
    from {
      opacity: 0;
      transform: scale(0.95);
    }
    to {
      opacity: 1;
      transform: scale(1);
    }
  }

  @keyframes fade-in {
    from {
      opacity: 0;
    }
    to {
      opacity: 1;
    }
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

    .teacher-selector-form,
    .teacher-display-box {
      width: 100%;
    }

    .teacher-display-box {
      min-width: unset;
    }

    .teacher-select {
      max-width: 100%;
    }

    .slots-info {
      justify-content: space-between;
    }

    .calendar-guidance {
      bottom: 0.5rem;
      left: 1rem;
      right: 1rem;
      transform: none;
    }

    .keyboard-hint {
      bottom: 5rem;
      right: 1rem;
      left: 1rem;
      text-align: center;
    }

    .toast {
      right: 1rem;
      left: 1rem;
    }
  }
</style>
