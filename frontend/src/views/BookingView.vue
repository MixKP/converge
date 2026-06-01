<script setup lang="ts">
  import { ref, computed, watch, nextTick } from 'vue'
  import { useBooking } from '../composables/useBooking'
  import PageLayout from '../components/PageLayout.vue'
  import Calendar from '../components/Calendar.vue'
  import CalendarDisabledOverlay from '../components/CalendarDisabledOverlay.vue'
  import BookingResults from '../components/BookingResults.vue'
  import type { EventClickArg } from '@fullcalendar/core'
  import type { Teacher } from '../types'

  const {
    calendarRef,
    activeTab,
    isEvaluating,
    errorMessage,
    successMessage,
    events,
    businessHours,
    subjects,
    branches,
    filteredTeachers,
    selectedSubjectId,
    selectedBranchId,
    selectedTeacherId,
    suggestions,
    showDetailedResults,
    suggestionEvents,
    initWatchers,
    addToCartDirectly,
    resetBookingState,
    getTeachersBySubject,
  } = useBooking()

  const aiMode = ref<'idle' | 'expanding' | 'expanded'>('idle')

  const aiSubjectId = ref<number | null>(null)
  const aiBranchId = ref<number | null>(null)
  const aiTeacherId = ref<number | null>(null)
  const aiFilteredTeachers = ref<Teacher[]>([])
  const aiTimeSlots = ref<Array<{ day_of_week: number; start: string; end: string }>>([])
  const aiNewSlotDay = ref<number>(1)
  const aiNewSlotStart = ref<string>('09:00')
  const aiNewSlotEnd = ref<string>('10:00')

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

  const currentStep = computed(() => {
    if (!selectedSubjectId.value) return 1
    if (!selectedBranchId.value) return 2
    return 3
  })

  const showAIOverlay = computed(() => aiMode.value !== 'idle' || showDetailedResults.value)

  const addAiTimeSlot = () => {
    if (aiNewSlotStart.value >= aiNewSlotEnd.value) return
    aiTimeSlots.value.push({
      day_of_week: aiNewSlotDay.value,
      start: aiNewSlotStart.value,
      end: aiNewSlotEnd.value,
    })
    aiNewSlotDay.value = (aiNewSlotDay.value + 1) % 7
    aiNewSlotStart.value = '09:00'
    aiNewSlotEnd.value = '10:00'
  }

  const removeAiTimeSlot = (index: number) => {
    aiTimeSlots.value.splice(index, 1)
  }

  const getDayName = (dayOfWeek: number): string => {
    return DAY_NAMES[dayOfWeek] || 'Unknown'
  }

  const openAIMode = () => {
    aiMode.value = 'expanding'
    nextTick(() => {
      aiMode.value = 'expanded'
    })
  }

  const closeAIMode = () => {
    aiMode.value = 'idle'
    resetBookingState()
  }

  const handleSuggestionClick = (info: EventClickArg): void => {
    const props = info.event.extendedProps
    if (props?.isSuggestion) {
      addToCartDirectly(props.teacherId, props.teacherName, info.event.startStr, info.event.endStr)
    }
  }

  const handleGetAISuggestions = async () => {
    if (!aiSubjectId.value || !aiBranchId.value || aiTimeSlots.value.length === 0) {
      errorMessage.value = 'Please select subject, branch, and time slots'
      return
    }

    isEvaluating.value = true
    errorMessage.value = ''
    successMessage.value = ''

    try {
      const { bookingApi } = await import('../services/bookingApi')
      const response = await bookingApi.evaluate({
        subject_id: aiSubjectId.value!,
        branch_id: aiBranchId.value!,
        preferred_slots: aiTimeSlots.value,
        duration_minutes: 60,
        preferred_teacher_id: aiTeacherId.value ?? undefined,
      })

      suggestions.value = response
      showDetailedResults.value = true
      successMessage.value = 'Booking options generated!'
      setTimeout(() => (successMessage.value = ''), 3000)
    } catch (error) {
      errorMessage.value =
        error instanceof Error ? error.message : 'Failed to generate booking options'
    } finally {
      isEvaluating.value = false
    }
  }

  watch(aiSubjectId, async (newSubjectId) => {
    if (newSubjectId) {
      aiFilteredTeachers.value = await getTeachersBySubject(newSubjectId)
    } else {
      aiFilteredTeachers.value = []
    }
    aiTeacherId.value = null
  })

  initWatchers()
</script>

<template>
  <PageLayout title="Book a Session">
    <div class="booking-container">
      <!-- Technical watermark -->
      <div class="technical-watermark" aria-hidden="true">
        <div class="watermark-grid"></div>
        <div class="watermark-label">
          <span class="label-line"></span>
          <span class="label-text">SCHEDULING_SYSTEM</span>
          <span class="label-line"></span>
        </div>
      </div>

      <div class="booking-layout">
        <!-- Calendar Section -->
        <div class="calendar-section" :class="{ 'calendar-section--expanded': showAIOverlay }">
          <div class="section-header">
            <div class="header-title">
              <h2 class="title">Availability</h2>
              <p class="subtitle">Select your preferred time slot</p>
            </div>
            <button
              v-if="!showAIOverlay"
              type="button"
              class="ai-button"
              @click="openAIMode"
            >
              <span class="ai-button-text">AI Assist</span>
              <svg class="ai-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="1.5"
                  d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09zM18.259 8.715L18 9.75l-.259-1.035a3.375 3.375 0 00-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 002.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 002.456 2.456L21.75 6l-1.035.259a3.375 3.375 0 00-2.456 2.456zM16.894 20.567L16.5 21.75l-.394-1.183a2.25 2.25 0 00-1.423-1.423L13.5 18.75l1.183-.394a2.25 2.25 0 001.423-1.423l.394-1.183.394 1.183a2.25 2.25 0 001.423 1.423l1.183.394-1.183.394a2.25 2.25 0 00-1.423 1.423z"
                />
              </svg>
            </button>
          </div>

          <div class="calendar-wrapper">
            <div class="calendar-inner">
              <Calendar
                v-if="activeTab === 'manual' ? selectedSubjectId : aiSubjectId"
                ref="calendarRef"
                :model-value="events"
                :additional-events="suggestionEvents"
                :editable="true"
                :business-hours="businessHours"
                constraint="businessHours"
                @update:model-value="events = $event"
                @event-click="handleSuggestionClick"
                @update:options="
                  (e: { dayHeaderFormat: string }) =>
                    calendarRef?.setOption('dayHeaderFormat', e.dayHeaderFormat)
                "
              />
              <CalendarDisabledOverlay v-else message="Select a subject to view availability" />
            </div>
          </div>
        </div>

        <!-- Control Panel -->
        <div class="control-panel" :class="{ 'control-panel--hidden': showAIOverlay }">
          <div v-if="!showAIOverlay" class="panel-content">
            <!-- Selection Form -->
            <div class="selection-form">
              <div class="form-row">
                <div class="form-field">
                  <div class="field-label">
                    <span class="label-indicator"></span>
                    <span class="label-text">Subject</span>
                  </div>
                  <select v-model="selectedSubjectId" class="field-select">
                    <option :value="null">Select subject</option>
                    <option v-for="subject in subjects" :key="subject.id" :value="subject.id">
                      {{ subject.name }}
                    </option>
                  </select>
                </div>
              </div>

              <div class="form-row">
                <div class="form-field">
                  <div class="field-label">
                    <span class="label-indicator"></span>
                    <span class="label-text">Branch</span>
                  </div>
                  <select
                    v-model="selectedBranchId"
                    :disabled="!selectedSubjectId"
                    class="field-select"
                  >
                    <option :value="null">Select branch</option>
                    <option v-for="branch in branches" :key="branch.id" :value="branch.id">
                      {{ branch.name }}
                    </option>
                  </select>
                </div>
              </div>

              <div class="form-row">
                <div class="form-field">
                  <div class="field-label">
                    <span class="label-indicator"></span>
                    <span class="label-text">Teacher</span>
                  </div>
                  <select
                    v-model="selectedTeacherId"
                    :disabled="!selectedSubjectId"
                    class="field-select"
                  >
                    <option :value="null">Any available</option>
                    <option
                      v-for="teacher in filteredTeachers"
                      :key="teacher.id"
                      :value="teacher.id"
                    >
                      {{ teacher.name }}
                    </option>
                  </select>
                </div>
              </div>
            </div>

            <!-- Selected Slots Display -->
            <div v-if="events.length > 0" class="slots-display">
              <div class="slots-header">
                <span class="slots-count">{{ events.length }}</span>
                <span class="slots-label">slot{{ events.length > 1 ? 's' : '' }} selected</span>
              </div>
              <div class="slots-grid">
                <div v-for="(event, idx) in events.slice(0, 3)" :key="idx" class="slot-chip">
                  <span class="slot-time">{{
                    new Date(event.start as Date).toLocaleTimeString('en-US', {
                      hour: '2-digit',
                      minute: '2-digit',
                      hour12: false,
                    })
                  }}</span>
                </div>
                <div v-if="events.length > 3" class="slot-chip slot-chip--more">
                  +{{ events.length - 3 }}
                </div>
              </div>
            </div>

            <!-- Action Button -->
            <button
              type="button"
              class="add-button spring-press"
              :disabled="
                !selectedSubjectId || !selectedBranchId || !selectedTeacherId || events.length === 0
              "
              @click="
                events.forEach((e) => {
                  const teacher = filteredTeachers.find((t) => t.id === selectedTeacherId)
                  if (teacher) {
                    addToCartDirectly(teacher.id, teacher.name, e.start as string, e.end as string)
                  }
                })
              "
            >
              <span class="button-text">Add to Cart</span>
              <svg class="button-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M12 4.5v15m7.5-7.5h-15"
                />
              </svg>
            </button>
          </div>

          <!-- AI Panel -->
          <div v-else class="ai-content">
            <div class="ai-header">
              <h3 class="ai-title">Smart Suggestions</h3>
              <button type="button" class="ai-close" @click="closeAIMode">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M6 18L18 6M6 6l12 12"
                  />
                </svg>
              </button>
            </div>

            <div v-if="!showDetailedResults && !isEvaluating" class="ai-form">
              <div class="ai-field">
                <label class="ai-label">Subject</label>
                <select v-model="aiSubjectId" class="ai-select">
                  <option :value="null">Select subject</option>
                  <option v-for="subject in subjects" :key="subject.id" :value="subject.id">
                    {{ subject.name }}
                  </option>
                </select>
              </div>

              <div class="ai-field">
                <label class="ai-label">Preferred Time Slots</label>
                <div class="slot-builder">
                  <div class="slot-controls">
                    <select v-model="aiNewSlotDay" class="time-select">
                      <option value="0">SUN</option>
                      <option value="1">MON</option>
                      <option value="2">TUE</option>
                      <option value="3">WED</option>
                      <option value="4">THU</option>
                      <option value="5">FRI</option>
                      <option value="6">SAT</option>
                    </select>
                    <select v-model="aiNewSlotStart" class="time-select">
                      <option v-for="time in TIME_OPTIONS" :key="time" :value="time">
                        {{ time }}
                      </option>
                    </select>
                    <span class="time-separator">—</span>
                    <select v-model="aiNewSlotEnd" class="time-select">
                      <option v-for="time in TIME_OPTIONS" :key="time" :value="time">
                        {{ time }}
                      </option>
                    </select>
                    <button
                      type="button"
                      class="slot-add"
                      :disabled="aiNewSlotStart >= aiNewSlotEnd"
                      @click="addAiTimeSlot"
                    >
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
                        <path
                          stroke-linecap="round"
                          stroke-linejoin="round"
                          stroke-width="2"
                          d="M12 4.5v15m7.5-7.5h-15"
                        />
                      </svg>
                    </button>
                  </div>

                  <div v-if="aiTimeSlots.length > 0" class="slot-list">
                    <div v-for="(slot, index) in aiTimeSlots" :key="index" class="slot-item">
                      <span class="slot-item-text"
                        >{{ getDayName(slot.day_of_week) }} {{ slot.start }} - {{ slot.end }}</span
                      >
                      <button
                        type="button"
                        class="slot-item-remove"
                        @click="removeAiTimeSlot(index)"
                      >
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
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
              </div>

              <div class="ai-field">
                <label class="ai-label">Branch</label>
                <select v-model="aiBranchId" :disabled="!aiSubjectId" class="ai-select">
                  <option :value="null">Select branch</option>
                  <option v-for="branch in branches" :key="branch.id" :value="branch.id">
                    {{ branch.name }}
                  </option>
                </select>
              </div>

              <div class="ai-field">
                <label class="ai-label">Teacher (Optional)</label>
                <select v-model="aiTeacherId" :disabled="!aiSubjectId" class="ai-select">
                  <option :value="null">Any available</option>
                  <option
                    v-for="teacher in aiFilteredTeachers"
                    :key="teacher.id"
                    :value="teacher.id"
                  >
                    {{ teacher.name }}
                  </option>
                </select>
              </div>

              <button
                type="button"
                class="ai-submit spring-press"
                :disabled="!aiSubjectId || !aiBranchId || aiTimeSlots.length === 0"
                @click="handleGetAISuggestions"
              >
                <span>Generate Suggestions</span>
                <svg class="submit-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="1.5"
                    d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09z"
                  />
                </svg>
              </button>
            </div>

            <BookingResults
              :suggestions="suggestions"
              :show-detailed-results="showDetailedResults"
              :is-evaluating="isEvaluating"
              @confirm-booking="addToCartDirectly"
              @reset="closeAIMode"
            />
          </div>
        </div>
      </div>

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
  .booking-container {
    position: relative;
    height: 100%;
    overflow: hidden;
  }

  /* Technical Watermark */
  .technical-watermark {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    pointer-events: none;
    z-index: 0;
    overflow: hidden;
  }

  .watermark-grid {
    position: absolute;
    inset: 0;
    background-image:
      linear-gradient(var(--border-subtle) 1px, transparent 1px),
      linear-gradient(90deg, var(--border-subtle) 1px, transparent 1px);
    background-size: 40px 40px;
    opacity: 0.3;
  }

  .watermark-label {
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%) rotate(-45deg);
    display: flex;
    align-items: center;
    gap: 2rem;
    opacity: 0.04;
  }

  .label-line {
    width: 80px;
    height: 1px;
    background: var(--primary-indigo);
  }

  .label-text {
    font-family: 'JetBrains Mono', monospace;
    font-size: 1.5rem;
    font-weight: 600;
    letter-spacing: 0.3em;
    color: var(--primary-indigo);
  }

  /* Main Layout */
  .booking-layout {
    position: relative;
    display: grid;
    grid-template-columns: 1fr 340px;
    gap: 1.5rem;
    height: 100%;
    z-index: 1;
  }

  @media (max-width: 1024px) {
    .booking-layout {
      grid-template-columns: 1fr;
      grid-template-rows: 1fr auto;
    }
  }

  /* Calendar Section */
  .calendar-section {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    transition: all 0.5s cubic-bezier(0.34, 1.56, 0.64, 1);
  }

  .calendar-section--expanded {
    grid-column: 1 / -1;
  }

  .section-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.5rem 0;
  }

  .header-title .title {
    font-size: 1.25rem;
    font-weight: 600;
    color: var(--text-primary);
    font-family: 'Instrument Sans', sans-serif;
    letter-spacing: -0.01em;
  }

  .subtitle {
    font-size: 0.8125rem;
    color: var(--text-secondary);
    margin-top: 0.25rem;
    font-family: 'JetBrains Mono', monospace;
    letter-spacing: 0.05em;
  }

  .ai-button {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.625rem 1rem;
    background: var(--primary-indigo);
    color: white;
    border: none;
    border-radius: 6px;
    font-size: 0.8125rem;
    font-weight: 500;
    font-family: 'IBM Plex Sans', sans-serif;
    cursor: pointer;
    transition: all 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
    box-shadow: 0 2px 8px rgba(62, 76, 122, 0.2);
  }

  .ai-button:hover {
    background: var(--primary-indigo-deep);
    transform: translateY(-2px);
    box-shadow: 0 4px 16px rgba(62, 76, 122, 0.3);
  }

  .ai-icon {
    width: 1rem;
    height: 1rem;
    animation: sparkle 2s ease-in-out infinite;
  }

  @keyframes sparkle {
    0%,
    100% {
      transform: scale(1) rotate(0deg);
    }
    50% {
      transform: scale(1.1) rotate(15deg);
    }
  }

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
    overflow: hidden;
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

  /* Control Panel */
  .control-panel {
    display: flex;
    flex-direction: column;
    background: var(--bg-card);
    border-radius: 8px;
    border: 1px solid var(--border-medium);
    box-shadow: var(--shadow-card);
    overflow: hidden;
    transition: all 0.5s cubic-bezier(0.34, 1.56, 0.64, 1);
  }

  .control-panel--hidden {
    position: absolute;
    right: 0;
    top: 0;
    width: 380px;
    height: 100%;
    z-index: 10;
    box-shadow: var(--shadow-elevated);
  }

  @media (max-width: 1024px) {
    .control-panel--hidden {
      position: relative;
      width: 100%;
    }
  }

  .panel-content,
  .ai-content {
    display: flex;
    flex-direction: column;
    padding: 1.25rem;
    gap: 1.25rem;
    height: 100%;
    overflow-y: auto;
  }

  /* Selection Form */
  .selection-form {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .form-row {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .form-field {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .field-label {
    display: flex;
    align-items: center;
    gap: 0.625rem;
  }

  .label-indicator {
    width: 4px;
    height: 4px;
    background: var(--accent-sage);
    border-radius: 50%;
  }

  .label-text {
    font-size: 0.75rem;
    font-weight: 500;
    color: var(--text-secondary);
    letter-spacing: 0.05em;
    text-transform: uppercase;
    font-family: 'JetBrains Mono', monospace;
  }

  .field-select {
    width: 100%;
    padding: 0.75rem 1rem;
    font-size: 0.875rem;
    color: var(--text-primary);
    background: var(--bg-cream);
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

  .field-select:hover {
    border-color: var(--primary-indigo);
  }

  .field-select:focus {
    outline: none;
    border-color: var(--primary-indigo);
    box-shadow: 0 0 0 3px rgba(62, 76, 122, 0.1);
  }

  .field-select:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  /* Slots Display */
  .slots-display {
    padding: 1rem;
    background: var(--bg-subtle);
    border-radius: 6px;
    border: 1px solid var(--border-subtle);
  }

  .slots-header {
    display: flex;
    align-items: baseline;
    gap: 0.375rem;
    margin-bottom: 0.75rem;
  }

  .slots-count {
    font-size: 1.5rem;
    font-weight: 600;
    color: var(--primary-indigo);
    font-family: 'JetBrains Mono', monospace;
  }

  .slots-label {
    font-size: 0.75rem;
    color: var(--text-secondary);
  }

  .slots-grid {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  .slot-chip {
    padding: 0.375rem 0.625rem;
    font-size: 0.75rem;
    font-family: 'JetBrains Mono', monospace;
    color: var(--text-primary);
    background: var(--bg-card);
    border: 1px solid var(--border-medium);
    border-radius: 4px;
  }

  .slot-chip--more {
    color: var(--text-secondary);
  }

  /* Add Button */
  .add-button {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    padding: 0.875rem 1.25rem;
    font-size: 0.875rem;
    font-weight: 500;
    color: white;
    background: var(--accent-sage);
    border: none;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
    box-shadow: 0 2px 8px rgba(122, 139, 109, 0.2);
    font-family: 'IBM Plex Sans', sans-serif;
  }

  .add-button:hover:not(:disabled) {
    background: var(--accent-sage);
    filter: brightness(0.9);
    transform: translateY(-2px);
    box-shadow: 0 4px 16px rgba(122, 139, 109, 0.3);
  }

  .add-button:disabled {
    opacity: 0.4;
    cursor: not-allowed;
    transform: none;
  }

  .button-icon {
    width: 1rem;
    height: 1rem;
  }

  /* AI Panel */
  .ai-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding-bottom: 1rem;
    border-bottom: 1px solid var(--border-subtle);
  }

  .ai-title {
    font-size: 1.125rem;
    font-weight: 600;
    color: var(--text-primary);
    font-family: 'Instrument Sans', sans-serif;
  }

  .ai-close {
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
  }

  .ai-close:hover {
    color: var(--text-primary);
    background: var(--bg-subtle);
  }

  .ai-close svg {
    width: 1.125rem;
    height: 1.125rem;
  }

  .ai-form {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .ai-field {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .ai-label {
    font-size: 0.75rem;
    font-weight: 500;
    color: var(--text-secondary);
    letter-spacing: 0.05em;
    text-transform: uppercase;
    font-family: 'JetBrains Mono', monospace;
  }

  .ai-select {
    width: 100%;
    padding: 0.625rem 0.875rem;
    font-size: 0.8125rem;
    color: var(--text-primary);
    background: var(--bg-cream);
    border: 1px solid var(--border-medium);
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.2s ease;
    font-family: 'IBM Plex Sans', sans-serif;
  }

  .ai-select:hover {
    border-color: var(--primary-indigo);
  }

  .ai-select:focus {
    outline: none;
    border-color: var(--primary-indigo);
    box-shadow: 0 0 0 3px rgba(62, 76, 122, 0.1);
  }

  .ai-select:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  /* Slot Builder */
  .slot-builder {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .slot-controls {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    align-items: center;
  }

  .time-select {
    padding: 0.5rem 0.625rem;
    font-size: 0.75rem;
    font-family: 'JetBrains Mono', monospace;
    background: var(--bg-card);
    border: 1px solid var(--border-medium);
    border-radius: 4px;
    color: var(--text-primary);
  }

  .time-separator {
    color: var(--text-muted);
    font-size: 0.75rem;
  }

  .slot-add {
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
  }

  .slot-add:hover:not(:disabled) {
    background: var(--primary-indigo-deep);
  }

  .slot-add:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .slot-add svg {
    width: 0.875rem;
    height: 0.875rem;
    color: white;
  }

  .slot-list {
    display: flex;
    flex-direction: column;
    gap: 0.375rem;
  }

  .slot-item {
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

  .slot-item-remove {
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
  }

  .slot-item-remove:hover {
    background: rgba(201, 109, 93, 0.1);
  }

  .slot-item-remove svg {
    width: 0.75rem;
    height: 0.75rem;
  }

  /* AI Submit */
  .ai-submit {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    padding: 0.75rem 1.25rem;
    font-size: 0.875rem;
    font-weight: 500;
    color: white;
    background: var(--primary-indigo);
    border: none;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
    box-shadow: 0 2px 8px rgba(62, 76, 122, 0.2);
    font-family: 'IBM Plex Sans', sans-serif;
  }

  .ai-submit:hover:not(:disabled) {
    background: var(--primary-indigo-deep);
    transform: translateY(-2px);
    box-shadow: 0 4px 16px rgba(62, 76, 122, 0.3);
  }

  .ai-submit:disabled {
    opacity: 0.4;
    cursor: not-allowed;
    transform: none;
  }

  .submit-icon {
    width: 1rem;
    height: 1rem;
    animation: sparkle 2s ease-in-out infinite;
  }

  /* Toast */
  .toast {
    position: fixed;
    bottom: 2rem;
    right: 2rem;
    display: flex;
    align-items: center;
    gap: 0.75rem;
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
    margin-left: 0.75rem;
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
    .section-header {
      flex-direction: column;
      align-items: flex-start;
      gap: 0.75rem;
    }

    .ai-button {
      width: 100%;
      justify-content: center;
    }

    .control-panel--hidden {
      width: calc(100% - 1rem);
      right: 0.5rem;
    }

    .toast {
      right: 1rem;
      left: 1rem;
    }
  }
</style>
