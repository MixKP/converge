<script setup lang="ts">
  import { ref, computed } from 'vue'
  import BookingResults from './BookingResults.vue'
  import { toMinutes } from '../utils/dateValidation'
  import type { BookingResponse } from '../types'
  import type { CartItem } from '../composables/useBookingCart'

  interface TimeSlot {
    day_of_week: number
    start: string
    end: string
  }

  interface Props {
    modelValue: TimeSlot[]
    selectedSubjectId: number | null
    selectedBranchId: number | null
    selectedTeacherId: number | null
    subjects: Array<{ id: number; name: string }>
    branches: Array<{ id: number; name: string }>
    filteredTeachers: Array<{ id: number; name: string }>
    suggestions?: BookingResponse | null
    showDetailedResults: boolean
    isEvaluating: boolean
    layout?: 'mobile' | 'tablet' | 'desktop'
    isMobile?: boolean
    cartItems?: CartItem[]
  }

  const props = withDefaults(defineProps<Props>(), {
    layout: 'desktop',
    isMobile: false,
    suggestions: null,
    cartItems: () => [],
  })

  const emit = defineEmits<{
    'update:modelValue': [value: TimeSlot[]]
    'update:selectedSubjectId': [value: number | null]
    'update:selectedBranchId': [value: number | null]
    'update:selectedTeacherId': [value: number | null]
    submit: []
    close: []
    reset: []
    'confirm-booking': [teacherId: number, teacherName: string, startTime: string, endTime: string]
  }>()

  const localSubjectId = computed({
    get: () => props.selectedSubjectId,
    set: (val) => emit('update:selectedSubjectId', val),
  })

  const localBranchId = computed({
    get: () => props.selectedBranchId,
    set: (val) => emit('update:selectedBranchId', val),
  })

  const localTeacherId = computed({
    get: () => props.selectedTeacherId,
    set: (val) => emit('update:selectedTeacherId', val),
  })

  const timeSlots = computed({
    get: () => props.modelValue,
    set: (val) => emit('update:modelValue', val),
  })

  const DAY_NAMES = [
    'Sunday',
    'Monday',
    'Tuesday',
    'Wednesday',
    'Thursday',
    'Friday',
    'Saturday',
  ] as const

  const TIME_OPTIONS: string[] = []
  for (let hour = 8; hour <= 18; hour++) {
    const hourStr = String(hour).padStart(2, '0')
    TIME_OPTIONS.push(`${hourStr}:00`, `${hourStr}:30`)
  }
  TIME_OPTIONS.push('19:00')

  const newSlotDay = ref<number>(1)
  const newSlotStart = ref<string>('09:00')
  const newSlotEnd = ref<string>('10:00')

  const canAddTimeSlots = computed(() => !!props.selectedSubjectId)

  const getDayName = (dayOfWeek: number): string => {
    return DAY_NAMES[dayOfWeek] || 'Unknown'
  }

  const addTimeSlot = () => {
    if (toMinutes(newSlotStart.value) >= toMinutes(newSlotEnd.value)) return

    timeSlots.value = [
      ...timeSlots.value,
      {
        day_of_week: newSlotDay.value,
        start: newSlotStart.value,
        end: newSlotEnd.value,
      },
    ]
    newSlotDay.value = (newSlotDay.value + 1) % 7
    newSlotStart.value = '09:00'
    newSlotEnd.value = '10:00'
  }

  const removeTimeSlot = (index: number) => {
    timeSlots.value = timeSlots.value.filter((_, i) => i !== index)
  }

  const handleSubmit = () => {
    emit('submit')
  }

  const handleClose = () => {
    emit('close')
  }

  const handleReset = () => {
    emit('reset')
  }

  const handleConfirmBooking = (
    teacherId: number,
    teacherName: string,
    startTime: string,
    endTime: string
  ) => {
    emit('confirm-booking', teacherId, teacherName, startTime, endTime)
  }

  const isSubmitDisabled = computed(() => {
    return (
      !props.selectedSubjectId ||
      !props.selectedBranchId ||
      timeSlots.value.length === 0 ||
      props.isEvaluating
    )
  })

  const cssClass = computed(() => {
    return `smart-suggestions-panel smart-suggestions-panel--${props.layout}`
  })

  const getCls = (element: string) => {
    return `smart-suggestions-panel--${props.layout}__${element}`
  }
</script>

<template>
  <div :class="cssClass">
    <!-- Header -->
    <div :class="getCls('header')">
      <div>
        <h3 :class="getCls('title')">Smart Suggestions</h3>
      </div>
      <button
        type="button"
        :class="getCls('close')"
        aria-label="Close suggestions"
        @click="handleClose"
      >
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" aria-hidden="true">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M6 18L18 6M6 6l12 12"
          />
        </svg>
      </button>
    </div>

    <!-- Form -->
    <div v-if="!showDetailedResults && !isEvaluating" :class="getCls('form')">
      <!-- Subject -->
      <div :class="getCls('field')">
        <label :class="getCls('label')" :for="`${layout}-ai-subject`">
          Subject <span :class="getCls('required')">*</span>
        </label>
        <select :id="`${layout}-ai-subject`" v-model="localSubjectId" :class="getCls('select')">
          <option :value="null">Select subject</option>
          <option v-for="subject in subjects" :key="subject.id" :value="subject.id">
            {{ subject.name }}
          </option>
        </select>
      </div>

      <!-- Time Slots -->
      <div :class="getCls('field')">
        <label :class="getCls('label')">
          Preferred Time Slots <span :class="getCls('required')">*</span>
        </label>

        <div :class="getCls('slot-controls')">
          <select
            v-model.number="newSlotDay"
            :class="getCls('time-select')"
            :disabled="!canAddTimeSlots"
            aria-label="Day of week"
          >
            <option value="0">{{ layout === 'mobile' ? 'Sun' : 'Sunday' }}</option>
            <option value="1">{{ layout === 'mobile' ? 'Mon' : 'Monday' }}</option>
            <option value="2">{{ layout === 'mobile' ? 'Tue' : 'Tuesday' }}</option>
            <option value="3">{{ layout === 'mobile' ? 'Wed' : 'Wednesday' }}</option>
            <option value="4">{{ layout === 'mobile' ? 'Thu' : 'Thursday' }}</option>
            <option value="5">{{ layout === 'mobile' ? 'Fri' : 'Friday' }}</option>
            <option value="6">{{ layout === 'mobile' ? 'Sat' : 'Saturday' }}</option>
          </select>
          <select
            v-model="newSlotStart"
            :class="getCls('time-select')"
            :disabled="!canAddTimeSlots"
            aria-label="Start time"
          >
            <option v-for="time in TIME_OPTIONS" :key="time" :value="time">
              {{ time }}
            </option>
          </select>
          <span :class="getCls('separator')">to</span>
          <select
            v-model="newSlotEnd"
            :class="getCls('time-select')"
            :disabled="!canAddTimeSlots"
            aria-label="End time"
          >
            <option v-for="time in TIME_OPTIONS" :key="time" :value="time">
              {{ time }}
            </option>
          </select>
          <button
            type="button"
            :class="getCls('add-slot')"
            :disabled="!canAddTimeSlots"
            aria-label="Add time slot"
            @click="addTimeSlot"
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" aria-hidden="true">
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M12 4.5v15m7.5-7.5h-15"
              />
            </svg>
          </button>
        </div>

        <!-- Added Slots -->
        <div v-if="timeSlots.length > 0" :class="getCls('slot-list')">
          <div v-for="(slot, index) in timeSlots" :key="index" :class="getCls('slot-item')">
            <span :class="getCls('slot-text')">
              {{ getDayName(slot.day_of_week) }} {{ slot.start }} - {{ slot.end }}
            </span>
            <button
              type="button"
              :class="getCls('slot-remove')"
              :aria-label="`Remove ${getDayName(slot.day_of_week)} ${slot.start} - ${slot.end}`"
              @click="removeTimeSlot(index)"
            >
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" aria-hidden="true">
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </button>
          </div>
        </div>
      </div>

      <!-- Branch -->
      <div :class="getCls('field')">
        <label :class="getCls('label')" :for="`${layout}-ai-branch`">
          Branch <span :class="getCls('required')">*</span>
        </label>
        <select
          :id="`${layout}-ai-branch`"
          v-model="localBranchId"
          :class="getCls('select')"
          :disabled="!selectedSubjectId"
        >
          <option :value="null">Select branch</option>
          <option v-for="branch in branches" :key="branch.id" :value="branch.id">
            {{ branch.name }}
          </option>
        </select>
      </div>

      <!-- Teacher (Optional) -->
      <div v-if="layout !== 'mobile'" :class="getCls('field')">
        <label :class="getCls('label')" :for="`${layout}-ai-teacher`"> Teacher (Optional) </label>
        <select
          :id="`${layout}-ai-teacher`"
          v-model="localTeacherId"
          :class="getCls('select')"
          :disabled="!selectedSubjectId"
        >
          <option :value="null">
            {{ layout === 'desktop' ? 'Show all available teachers' : 'All teachers' }}
          </option>
          <option v-for="teacher in filteredTeachers" :key="teacher.id" :value="teacher.id">
            {{ teacher.name }}
          </option>
        </select>
      </div>

      <!-- Submit Button -->
      <button
        type="button"
        :class="getCls('submit')"
        :disabled="isSubmitDisabled"
        :aria-busy="isEvaluating"
        @click="handleSubmit"
      >
        <span v-if="isEvaluating">Finding teachers...</span>
        <span v-else>Find Available Teachers</span>
        <svg
          v-if="!isEvaluating"
          :class="getCls('submit-icon')"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          aria-hidden="true"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="1.5"
            d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09z"
          />
        </svg>
      </button>
    </div>

    <!-- Results -->
    <div :class="getCls('results')">
      <BookingResults
        :suggestions="suggestions"
        :show-detailed-results="showDetailedResults"
        :is-evaluating="isEvaluating"
        :cart-items="cartItems"
        @confirm-booking="handleConfirmBooking"
        @reset="handleReset"
      />
    </div>
  </div>
</template>

<style scoped>
  .smart-suggestions-panel {
    display: flex;
    flex-direction: column;
    background: var(--bg-cream);
  }

  /* Mobile */
  .smart-suggestions-panel--mobile {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 100;
    overflow-y: auto;
    overflow-x: hidden;
    padding: 4rem 1rem 72px;
  }

  .smart-suggestions-panel--mobile__header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 1rem;
    padding-bottom: 1rem;
    border-bottom: 1px solid var(--border-medium);
  }

  .smart-suggestions-panel--mobile__title {
    font-size: 1.125rem;
    font-weight: 600;
    color: var(--text-primary);
    margin: 0;
    font-family: 'Instrument Sans', sans-serif;
  }

  .smart-suggestions-panel--mobile__subtitle {
    font-size: 0.8125rem;
    color: var(--text-secondary);
    margin: 0.25rem 0 0 0;
    font-family: 'Inter', sans-serif;
  }

  .smart-suggestions-panel--mobile__close {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 2.25rem;
    height: 2.25rem;
    padding: 0;
    background: var(--bg-subtle);
    border: 1px solid var(--border-medium);
    border-radius: 8px;
    cursor: pointer;
    transition: all 0.2s ease;
    touch-action: manipulation;
  }

  .smart-suggestions-panel--mobile__close:hover {
    background: var(--bg-card);
    border-color: var(--border-strong);
  }

  .smart-suggestions-panel--mobile__close svg {
    width: 1rem;
    height: 1rem;
    color: var(--text-primary);
  }

  .smart-suggestions-panel--mobile__form {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
    padding-top: 1rem;
  }

  .smart-suggestions-panel--mobile__field {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .smart-suggestions-panel--mobile__label {
    font-size: 0.75rem;
    font-weight: 600;
    letter-spacing: 0.05em;
    text-transform: uppercase;
    color: var(--text-muted);
    font-family: 'Inter', sans-serif;
  }

  .smart-suggestions-panel--mobile__required {
    color: var(--accent-coral);
  }

  .smart-suggestions-panel--mobile__hint {
    font-size: 0.75rem;
    color: var(--text-secondary);
    margin: 0;
    font-family: 'Inter', sans-serif;
  }

  .smart-suggestions-panel--mobile__select {
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

  .smart-suggestions-panel--mobile__select:disabled {
    opacity: 0.5;
    cursor: not-allowed;
    background-color: var(--bg-subtle);
  }

  .smart-suggestions-panel--mobile__slot-controls {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.5rem;
    max-width: 100%;
  }

  .smart-suggestions-panel--mobile__time-select {
    flex: 1;
    min-width: 70px;
    max-width: 120px;
    padding: 0.5rem 0.75rem;
    font-size: 0.8125rem;
    font-weight: 400;
    color: var(--text-primary);
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.15s ease;
    font-family: 'Inter', sans-serif;
    appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' fill='none' stroke='%23585863' viewBox='0 0 24 24'%3E%3Cpath stroke-linecap='round' stroke-linejoin='round' stroke-width='2' d='M19.5 8.25l-7.5 7.5-7.5-7.5'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 0.5rem center;
    background-size: 0.75rem;
    padding-right: 1.5rem;
    touch-action: manipulation;
    box-sizing: border-box;
  }

  .smart-suggestions-panel--mobile__time-select:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .smart-suggestions-panel--mobile__separator {
    font-size: 0.75rem;
    color: var(--text-secondary);
    font-family: 'Inter', sans-serif;
  }

  .smart-suggestions-panel--mobile__add-slot {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 2rem;
    height: 2rem;
    padding: 0;
    background: var(--primary-indigo);
    border: none;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.2s ease;
    touch-action: manipulation;
  }

  .smart-suggestions-panel--mobile__add-slot:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .smart-suggestions-panel--mobile__add-slot:not(:disabled):hover {
    background: var(--primary-indigo-deep);
  }

  .smart-suggestions-panel--mobile__add-slot svg {
    width: 0.875rem;
    height: 0.875rem;
    color: white;
  }

  .smart-suggestions-panel--mobile__slot-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    margin-top: 0.5rem;
  }

  .smart-suggestions-panel--mobile__slot-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 0.75rem;
    padding: 0.5rem 0.75rem;
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: 6px;
  }

  .smart-suggestions-panel--mobile__slot-text {
    font-size: 0.8125rem;
    color: var(--text-primary);
    font-family: 'Inter', sans-serif;
  }

  .smart-suggestions-panel--mobile__slot-remove {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 1.5rem;
    height: 1.5rem;
    padding: 0;
    background: transparent;
    border: none;
    cursor: pointer;
    border-radius: 4px;
    transition: all 0.2s ease;
    touch-action: manipulation;
  }

  .smart-suggestions-panel--mobile__slot-remove:hover {
    background: var(--bg-subtle);
  }

  .smart-suggestions-panel--mobile__slot-remove svg {
    width: 0.75rem;
    height: 0.75rem;
    color: var(--text-secondary);
  }

  .smart-suggestions-panel--mobile__submit {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    width: 100%;
    padding: 0.75rem 1rem;
    font-size: 0.875rem;
    font-weight: 600;
    color: white;
    background: linear-gradient(135deg, var(--accent-sage) 0%, var(--accent-mint) 100%);
    border: none;
    border-radius: 8px;
    cursor: pointer;
    transition: all 0.2s ease;
    font-family: 'Inter', sans-serif;
    touch-action: manipulation;
  }

  .smart-suggestions-panel--mobile__submit:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .smart-suggestions-panel--mobile__submit:not(:disabled):hover {
    filter: brightness(0.9);
  }

  .smart-suggestions-panel--mobile__submit-icon {
    width: 1rem;
    height: 1rem;
  }

  .smart-suggestions-panel--mobile__results {
    padding-top: 1rem;
    overflow-x: hidden;
  }

  /* Tablet */
  .smart-suggestions-panel--tablet {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 100;
    overflow-y: auto;
    overflow-x: hidden;
    padding: 4rem 1.5rem 72px;
  }

  .smart-suggestions-panel--tablet__header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 1rem;
    padding-bottom: 1rem;
    border-bottom: 1px solid var(--border-medium);
  }

  .smart-suggestions-panel--tablet__title {
    font-size: 1.125rem;
    font-weight: 600;
    color: var(--text-primary);
    margin: 0;
    font-family: 'Instrument Sans', sans-serif;
  }

  .smart-suggestions-panel--tablet__subtitle {
    font-size: 0.8125rem;
    color: var(--text-secondary);
    margin: 0.25rem 0 0 0;
    font-family: 'Inter', sans-serif;
  }

  .smart-suggestions-panel--tablet__close {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 2.25rem;
    height: 2.25rem;
    padding: 0;
    background: var(--bg-subtle);
    border: 1px solid var(--border-medium);
    border-radius: 8px;
    cursor: pointer;
    transition: all 0.2s ease;
    touch-action: manipulation;
  }

  .smart-suggestions-panel--tablet__close:hover {
    background: var(--bg-card);
    border-color: var(--border-strong);
  }

  .smart-suggestions-panel--tablet__close svg {
    width: 1rem;
    height: 1rem;
    color: var(--text-primary);
  }

  .smart-suggestions-panel--tablet__form {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
    padding-top: 1rem;
  }

  .smart-suggestions-panel--tablet__field {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .smart-suggestions-panel--tablet__label {
    font-size: 0.75rem;
    font-weight: 600;
    letter-spacing: 0.05em;
    text-transform: uppercase;
    color: var(--text-muted);
    font-family: 'Inter', sans-serif;
  }

  .smart-suggestions-panel--tablet__required {
    color: var(--accent-coral);
  }

  .smart-suggestions-panel--tablet__hint {
    font-size: 0.75rem;
    color: var(--text-secondary);
    margin: 0;
    font-family: 'Inter', sans-serif;
  }

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

  .smart-suggestions-panel--tablet__select:disabled {
    opacity: 0.5;
    cursor: not-allowed;
    background-color: var(--bg-subtle);
  }

  .smart-suggestions-panel--tablet__slot-controls {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.5rem;
  }

  .smart-suggestions-panel--tablet__time-select {
    flex: 1;
    min-width: 70px;
    padding: 0.5rem 0.75rem;
    font-size: 0.8125rem;
    font-weight: 400;
    color: var(--text-primary);
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.15s ease;
    font-family: 'Inter', sans-serif;
    appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' fill='none' stroke='%23585863' viewBox='0 0 24 24'%3E%3Cpath stroke-linecap='round' stroke-linejoin='round' stroke-width='2' d='M19.5 8.25l-7.5 7.5-7.5-7.5'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 0.5rem center;
    background-size: 0.75rem;
    padding-right: 1.5rem;
    touch-action: manipulation;
    box-sizing: border-box;
  }

  .smart-suggestions-panel--tablet__time-select:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .smart-suggestions-panel--tablet__separator {
    font-size: 0.75rem;
    color: var(--text-secondary);
    font-family: 'Inter', sans-serif;
  }

  .smart-suggestions-panel--tablet__add-slot {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 2rem;
    height: 2rem;
    padding: 0;
    background: var(--primary-indigo);
    border: none;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.2s ease;
    touch-action: manipulation;
  }

  .smart-suggestions-panel--tablet__add-slot:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .smart-suggestions-panel--tablet__add-slot:not(:disabled):hover {
    background: var(--primary-indigo-deep);
  }

  .smart-suggestions-panel--tablet__add-slot svg {
    width: 0.875rem;
    height: 0.875rem;
    color: white;
  }

  .smart-suggestions-panel--tablet__slot-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    margin-top: 0.5rem;
  }

  .smart-suggestions-panel--tablet__slot-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 0.75rem;
    padding: 0.5rem 0.75rem;
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: 6px;
  }

  .smart-suggestions-panel--tablet__slot-text {
    font-size: 0.8125rem;
    color: var(--text-primary);
    font-family: 'Inter', sans-serif;
  }

  .smart-suggestions-panel--tablet__slot-remove {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 1.5rem;
    height: 1.5rem;
    padding: 0;
    background: transparent;
    border: none;
    cursor: pointer;
    border-radius: 4px;
    transition: all 0.2s ease;
    touch-action: manipulation;
  }

  .smart-suggestions-panel--tablet__slot-remove:hover {
    background: var(--bg-subtle);
  }

  .smart-suggestions-panel--tablet__slot-remove svg {
    width: 0.75rem;
    height: 0.75rem;
    color: var(--text-secondary);
  }

  .smart-suggestions-panel--tablet__submit {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    width: 100%;
    padding: 0.75rem 1rem;
    font-size: 0.875rem;
    font-weight: 600;
    color: white;
    background: linear-gradient(135deg, var(--accent-sage) 0%, var(--accent-mint) 100%);
    border: none;
    border-radius: 8px;
    cursor: pointer;
    transition: all 0.2s ease;
    font-family: 'Inter', sans-serif;
    touch-action: manipulation;
  }

  .smart-suggestions-panel--tablet__submit:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .smart-suggestions-panel--tablet__submit:not(:disabled):hover {
    filter: brightness(0.9);
  }

  .smart-suggestions-panel--tablet__submit-icon {
    width: 1rem;
    height: 1rem;
  }

  .smart-suggestions-panel--tablet__results {
    padding-top: 1rem;
    overflow-x: hidden;
  }

  /* Desktop */
  .smart-suggestions-panel--desktop {
    display: flex;
    flex-direction: column;
    height: 100%;
    overflow-y: auto;
    overflow-x: hidden;
    background: transparent;
    border: none;
    border-radius: 0;
    box-shadow: none;
  }

  .smart-suggestions-panel--desktop__header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1.125rem 1rem 1rem;
    border-bottom: 1px solid var(--border-subtle);
  }

  .smart-suggestions-panel--desktop__title {
    font-size: 1.125rem;
    font-weight: 600;
    color: var(--text-primary);
    font-family: 'Instrument Sans', sans-serif;
  }

  .smart-suggestions-panel--desktop__subtitle {
    margin: 0.25rem 0 0 0;
    font-size: 0.8125rem;
    color: var(--text-secondary);
    line-height: 1.4;
  }

  .smart-suggestions-panel--desktop__close {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 2rem;
    height: 2rem;
    color: var(--text-secondary);
    background: transparent;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    transition: all 0.2s ease;
    touch-action: manipulation;
  }

  .smart-suggestions-panel--desktop__close:hover {
    color: var(--text-primary);
    background: var(--bg-subtle);
  }

  .smart-suggestions-panel--desktop__close svg {
    width: 1.125rem;
    height: 1.125rem;
  }

  .smart-suggestions-panel--desktop__form {
    display: flex;
    flex-direction: column;
    padding: 1rem;
    gap: 1rem;
  }

  .smart-suggestions-panel--desktop__field {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .smart-suggestions-panel--desktop__label {
    font-size: 0.75rem;
    font-weight: 500;
    color: var(--text-secondary);
    letter-spacing: 0.05em;
    text-transform: uppercase;
    font-family: 'JetBrains Mono', monospace;
  }

  .smart-suggestions-panel--desktop__required {
    color: var(--accent-coral);
    margin-left: 0.25rem;
  }

  .smart-suggestions-panel--desktop__hint {
    margin: 0;
    font-size: 0.8125rem;
    color: var(--text-muted);
    line-height: 1.4;
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

  .smart-suggestions-panel--desktop__slot-controls {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    align-items: center;
    justify-content: flex-start;
  }

  .smart-suggestions-panel--desktop__time-select {
    padding: 0.5rem 0.625rem;
    font-size: 0.75rem;
    font-family: 'JetBrains Mono', monospace;
    background: var(--bg-card);
    border: 1px solid var(--border-medium);
    border-radius: 4px;
    color: var(--text-primary);
    touch-action: manipulation;
  }

  .smart-suggestions-panel--desktop__time-select:hover:not(:disabled) {
    border-color: var(--primary-indigo);
  }

  .smart-suggestions-panel--desktop__time-select:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .smart-suggestions-panel--desktop__separator {
    color: var(--text-muted);
    font-size: 0.75rem;
  }

  .smart-suggestions-panel--desktop__add-slot {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 1.75rem;
    height: 1.75rem;
    background: var(--primary-indigo);
    border: none;
    border-radius: 4px;
    cursor: pointer;
    transition: all 0.2s ease;
    touch-action: manipulation;
  }

  .smart-suggestions-panel--desktop__add-slot:hover:not(:disabled) {
    background: var(--primary-indigo-deep);
  }

  .smart-suggestions-panel--desktop__add-slot:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .smart-suggestions-panel--desktop__add-slot svg {
    width: 0.875rem;
    height: 0.875rem;
    color: white;
  }

  .smart-suggestions-panel--desktop__slot-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    padding-top: 0.75rem;
    border-top: 1px solid var(--border-subtle);
  }

  .smart-suggestions-panel--desktop__slot-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.5rem 0.625rem;
    font-size: 0.75rem;
    font-family: 'JetBrains Mono', monospace;
    color: var(--text-primary);
    background: var(--bg-subtle);
    border-radius: 4px;
  }

  .smart-suggestions-panel--desktop__slot-remove {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 1.25rem;
    height: 1.25rem;
    color: var(--accent-coral);
    background: transparent;
    border: none;
    border-radius: 3px;
    cursor: pointer;
    transition: all 0.2s ease;
    touch-action: manipulation;
  }

  .smart-suggestions-panel--desktop__slot-remove:hover {
    background: rgba(201, 109, 93, 0.1);
  }

  .smart-suggestions-panel--desktop__slot-remove svg {
    width: 0.75rem;
    height: 0.75rem;
  }

  .smart-suggestions-panel--desktop__submit {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    padding: 0.75rem 1.25rem;
    font-size: 0.875rem;
    font-weight: 500;
    color: white;
    background: linear-gradient(135deg, var(--accent-sage) 0%, var(--accent-mint) 100%);
    border: none;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.2s cubic-bezier(0.25, 1, 0.5, 1);
    box-shadow: 0 2px 8px rgba(157, 180, 160, 0.2);
    font-family: 'Inter', sans-serif;
    touch-action: manipulation;
    flex-shrink: 0;
    max-height: 44px;
  }

  .smart-suggestions-panel--desktop__submit:hover:not(:disabled) {
    filter: brightness(0.9);
    transform: translateY(-1px);
    box-shadow: 0 3px 12px rgba(157, 180, 160, 0.25);
  }

  .smart-suggestions-panel--desktop__submit:disabled {
    opacity: 1;
    cursor: not-allowed;
    transform: none;
    background: var(--bg-subtle);
    color: var(--text-muted);
    box-shadow: none;
  }

  .smart-suggestions-panel--desktop__submit-icon {
    width: 1rem;
    height: 1rem;
    flex-shrink: 0;
  }

  .smart-suggestions-panel--desktop__results {
    padding: 0 1rem 1rem;
    overflow-x: hidden;
  }
</style>
