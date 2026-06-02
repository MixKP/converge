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

  const canAddTimeSlots = computed(() => !!aiSubjectId.value)

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

  const addAiTimeSlot = () => {
    // Convert time strings to minutes for proper comparison
    const toMinutes = (timeStr: string) => {
      const [hour, minute] = timeStr.split(':').map(Number)
      return hour * 60 + minute
    }
    if (toMinutes(aiNewSlotStart.value) >= toMinutes(aiNewSlotEnd.value)) return

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
      addToCartDirectly(
        props.teacherId,
        props.teacherName,
        info.event.startStr,
        info.event.endStr,
        aiSubjectId.value ?? undefined,
        aiBranchId.value ?? undefined
      )
    }
  }

  const handleAIBooking = (
    teacherId: number,
    teacherName: string,
    startTime: string,
    endTime: string
  ): void => {
    addToCartDirectly(
      teacherId,
      teacherName,
      startTime,
      endTime,
      aiSubjectId.value ?? undefined,
      aiBranchId.value ?? undefined
    )
  }

  const handleGetAISuggestions = async () => {
    if (!aiSubjectId.value || !aiBranchId.value || aiTimeSlots.value.length === 0) {
      errorMessage.value = 'Please select a subject, branch, and add at least one time slot'
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
      successMessage.value = `${response.results.length} booking option${response.results.length > 1 ? 's' : ''} found!`
      setTimeout(() => (successMessage.value = ''), 4000)
    } catch (error) {
      errorMessage.value =
        error instanceof Error
          ? error.message
          : 'Unable to find available teachers. Please try different time slots.'
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

      <div class="booking-layout" :class="{ 'booking-layout--split': aiMode !== 'idle' }">
        <!-- Calendar Section -->
        <div class="calendar-section">
          <div class="section-header">
            <div class="header-title">
              <h2 class="title">Availability</h2>
              <p class="subtitle">Select your preferred time slot</p>
            </div>
            <button
              v-if="aiMode === 'idle'"
              type="button"
              class="ai-button"
              aria-label="Open smart booking suggestions"
              @click="openAIMode"
            >
              <span class="ai-button-text">Smart Suggestions</span>
              <svg
                class="ai-icon"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                aria-hidden="true"
              >
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

        <!-- Manual Booking Panel -->
        <div v-if="aiMode === 'idle'" class="control-panel">
          <div class="panel-content">
            <!-- Selection Form -->
            <div class="selection-form">
              <div class="form-row">
                <div class="form-field">
                  <div class="field-label">
                    <span class="label-indicator"></span>
                    <span class="label-text">Subject</span>
                  </div>
                  <select
                    id="subject-select"
                    v-model="selectedSubjectId"
                    class="field-select"
                    aria-label="Select subject"
                  >
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
                    id="branch-select"
                    v-model="selectedBranchId"
                    :disabled="!selectedSubjectId"
                    class="field-select"
                    aria-label="Select branch"
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
                    id="teacher-select"
                    v-model="selectedTeacherId"
                    :disabled="!selectedSubjectId"
                    class="field-select"
                    aria-label="Select teacher"
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
              class="add-button"
              :disabled="
                !selectedSubjectId || !selectedBranchId || !selectedTeacherId || events.length === 0
              "
              :aria-label="
                events.length === 1 ? 'Add slot to cart' : `Add ${events.length} slots to cart`
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
              <span class="button-text">{{
                events.length > 0
                  ? `Add ${events.length} slot${events.length > 1 ? 's' : ''} to Cart`
                  : 'Add to Cart'
              }}</span>
              <svg
                class="button-icon"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                aria-hidden="true"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M12 4.5v15m7.5-7.5h-15"
                />
              </svg>
            </button>
          </div>
        </div>

        <!-- AI Panel (Split View) -->
        <div v-else class="control-panel control-panel--ai">
          <div class="panel-content panel-content--ai">
            <div class="ai-header">
              <h3 class="ai-title">Smart Suggestions</h3>
              <button
                type="button"
                class="ai-close"
                aria-label="Close suggestions and return to manual booking"
                @click="closeAIMode"
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

            <div v-if="!showDetailedResults && !isEvaluating" class="ai-form">
              <div class="ai-field">
                <label class="ai-label" for="ai-subject-select"
                  >Subject <span class="required">*</span></label
                >
                <select id="ai-subject-select" v-model="aiSubjectId" class="ai-select">
                  <option :value="null">Select subject</option>
                  <option v-for="subject in subjects" :key="subject.id" :value="subject.id">
                    {{ subject.name }}
                  </option>
                </select>
              </div>

              <div class="ai-field">
                <label id="ai-slots-label" class="ai-label"
                  >Preferred Time Slots <span class="required">*</span></label
                >
                <div
                  class="slot-builder"
                  :class="{ 'slot-builder--disabled': !canAddTimeSlots }"
                  role="group"
                  aria-labelledby="ai-slots-label"
                >
                  <div v-if="!canAddTimeSlots" class="slot-disabled-msg">
                    Select a subject above to add time slots
                  </div>
                  <div
                    class="slot-controls"
                    :class="{ 'slot-controls--disabled': !canAddTimeSlots }"
                  >
                    <select
                      v-model.number="aiNewSlotDay"
                      class="time-select"
                      :disabled="!canAddTimeSlots"
                    >
                      <option value="0">SUN</option>
                      <option value="1">MON</option>
                      <option value="2">TUE</option>
                      <option value="3">WED</option>
                      <option value="4">THU</option>
                      <option value="5">FRI</option>
                      <option value="6">SAT</option>
                    </select>
                    <select
                      v-model="aiNewSlotStart"
                      class="time-select"
                      :disabled="!canAddTimeSlots"
                    >
                      <option v-for="time in TIME_OPTIONS" :key="time" :value="time">
                        {{ time }}
                      </option>
                    </select>
                    <span class="time-separator">—</span>
                    <select v-model="aiNewSlotEnd" class="time-select" :disabled="!canAddTimeSlots">
                      <option v-for="time in TIME_OPTIONS" :key="time" :value="time">
                        {{ time }}
                      </option>
                    </select>
                    <button
                      type="button"
                      class="slot-add"
                      :disabled="!canAddTimeSlots"
                      aria-label="Add time slot"
                      @click="addAiTimeSlot"
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

                  <div v-if="aiTimeSlots.length > 0" class="slot-list">
                    <div v-for="(slot, index) in aiTimeSlots" :key="index" class="slot-item">
                      <span class="slot-item-text"
                        >{{ getDayName(slot.day_of_week) }} {{ slot.start }} - {{ slot.end }}</span
                      >
                      <button
                        type="button"
                        class="slot-item-remove"
                        :aria-label="`Remove ${getDayName(slot.day_of_week)} ${slot.start} - ${slot.end}`"
                        @click="removeAiTimeSlot(index)"
                      >
                        <svg
                          viewBox="0 0 24 24"
                          fill="none"
                          stroke="currentColor"
                          aria-hidden="true"
                        >
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
                <label class="ai-label" for="ai-branch-select"
                  >Branch <span class="required">*</span></label
                >
                <select
                  id="ai-branch-select"
                  v-model="aiBranchId"
                  :disabled="!aiSubjectId"
                  class="ai-select"
                >
                  <option :value="null">Select branch</option>
                  <option v-for="branch in branches" :key="branch.id" :value="branch.id">
                    {{ branch.name }}
                  </option>
                </select>
              </div>

              <div class="ai-field">
                <label class="ai-label" for="ai-teacher-select">Teacher (Optional)</label>
                <select
                  id="ai-teacher-select"
                  v-model="aiTeacherId"
                  :disabled="!aiSubjectId"
                  class="ai-select"
                >
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
                class="ai-submit"
                :disabled="!aiSubjectId || !aiBranchId || aiTimeSlots.length === 0"
                @click="handleGetAISuggestions"
              >
                <span>Find Available Teachers</span>
                <svg
                  class="submit-icon"
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

            <BookingResults
              :suggestions="suggestions"
              :show-detailed-results="showDetailedResults"
              :is-evaluating="isEvaluating"
              @confirm-booking="handleAIBooking"
              @reset="closeAIMode"
            />
          </div>
        </div>
      </div>

      <!-- Toast Notifications -->
      <div v-if="successMessage" class="toast toast--success" role="status" aria-live="polite">
        <svg
          class="toast-icon"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          aria-hidden="true"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
          />
        </svg>
        {{ successMessage }}
      </div>

      <div v-if="errorMessage" class="toast toast--error" role="alert" aria-live="assertive">
        <svg
          class="toast-icon"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          aria-hidden="true"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z"
          />
        </svg>
        {{ errorMessage }}
        <button class="toast-close" aria-label="Dismiss message" @click="errorMessage = ''">
          ×
        </button>
      </div>
    </div>
  </PageLayout>
</template>

<style src="../css/BookingView.css"></style>
