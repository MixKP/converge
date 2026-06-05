<script setup lang="ts">
  import { ref, computed, nextTick, onMounted, onUnmounted } from 'vue'
  import { useBooking } from '../composables/useBooking'
  import PageLayout from '../components/PageLayout.vue'
  import Calendar from '../components/Calendar.vue'
  import CalendarDisabledOverlay from '../components/CalendarDisabledOverlay.vue'
  import BookingResults from '../components/BookingResults.vue'
  import type { EventClickArg } from '@fullcalendar/core'

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
  } = useBooking()

  const aiMode = ref<'idle' | 'expanding' | 'expanded'>('idle')

  // Mobile step flow: 'selection' -> 'calendar'
  const mobileStep = ref<'selection' | 'calendar'>('selection')
  const isMobile = ref(false)
  const isTablet = ref(false)

  // AI-specific time slots (subject/branch/teacher now shared with manual form)
  const aiTimeSlots = ref<Array<{ day_of_week: number; start: string; end: string }>>([])
  const aiNewSlotDay = ref<number>(1)
  const aiNewSlotStart = ref<string>('09:00')
  const aiNewSlotEnd = ref<string>('10:00')

  const canAddTimeSlots = computed(() => !!selectedSubjectId.value)
  const canProceedToCalendar = computed(() => !!selectedSubjectId.value && !!selectedBranchId.value)

  // Detect mobile on mount and resize
  const checkMobile = () => {
    const width = globalThis.innerWidth
    isMobile.value = width <= 425
    isTablet.value = width > 425 && width < 1024
  }

  onMounted(() => {
    checkMobile()
    globalThis.addEventListener('resize', checkMobile)
  })
  onUnmounted(() => {
    globalThis.removeEventListener('resize', checkMobile)
  })

  const proceedToCalendar = () => {
    if (canProceedToCalendar.value) {
      mobileStep.value = 'calendar'
    }
  }

  const backToSelection = () => {
    mobileStep.value = 'selection'
  }

  // Focus management
  const aiSubmitButtonRef = ref<HTMLButtonElement | null>(null)
  const previouslyFocusedElement = ref<HTMLElement | null>(null)

  // Keyboard shortcuts
  const handleKeyDown = (event: KeyboardEvent) => {
    // Cmd/Ctrl + K to open AI mode
    if ((event.metaKey || event.ctrlKey) && event.key === 'k') {
      event.preventDefault()
      if (aiMode.value === 'idle') {
        openAIMode()
        nextTick(() => {
          aiSubmitButtonRef.value?.focus()
        })
      }
    }
    // Escape to close AI mode or dismiss messages
    if (event.key === 'Escape') {
      if (aiMode.value !== 'idle') {
        closeAIMode()
        previouslyFocusedElement.value?.focus()
      } else if (errorMessage.value) {
        errorMessage.value = ''
      } else if (successMessage.value) {
        successMessage.value = ''
      }
    }
  }

  onMounted(() => {
    globalThis.addEventListener('keydown', handleKeyDown)
  })

  onUnmounted(() => {
    globalThis.removeEventListener('keydown', handleKeyDown)
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
    previouslyFocusedElement.value = globalThis.document.activeElement as HTMLElement
    aiMode.value = 'expanding'
    nextTick(() => {
      aiMode.value = 'expanded'
    })
  }

  const closeAIMode = () => {
    aiMode.value = 'idle'
    aiTimeSlots.value = []
    resetBookingState()
    previouslyFocusedElement.value?.focus()
    previouslyFocusedElement.value = null
  }

  const handleSuggestionClick = (info: EventClickArg): void => {
    const props = info.event.extendedProps
    if (props?.isSuggestion) {
      addToCartDirectly(
        props.teacherId,
        props.teacherName,
        info.event.startStr,
        info.event.endStr,
        selectedSubjectId.value ?? undefined,
        selectedBranchId.value ?? undefined
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
      selectedSubjectId.value ?? undefined,
      selectedBranchId.value ?? undefined
    )
  }

  const handleGetAISuggestions = async () => {
    if (!selectedSubjectId.value || !selectedBranchId.value || aiTimeSlots.value.length === 0) {
      const missing = []
      if (!selectedSubjectId.value) missing.push('a subject')
      if (!selectedBranchId.value) missing.push('a branch')
      if (aiTimeSlots.value.length === 0) missing.push('at least one time slot')
      errorMessage.value = `To find teachers, please select ${missing.join(' and ')}`
      return
    }

    isEvaluating.value = true
    errorMessage.value = ''
    successMessage.value = ''

    try {
      const { bookingApi } = await import('../services/bookingApi')
      const response = await bookingApi.evaluate({
        subject_id: selectedSubjectId.value!,
        branch_id: selectedBranchId.value!,
        preferred_slots: aiTimeSlots.value,
        duration_minutes: 60,
        preferred_teacher_id: selectedTeacherId.value ?? undefined,
      })

      suggestions.value = response
      showDetailedResults.value = true
      activeTab.value = 'ai'
      successMessage.value = `${response.results.length} teacher${response.results.length > 1 ? 's' : ''} available. Click a time slot on the calendar or a suggestion below to book.`
      setTimeout(() => (successMessage.value = ''), 8000)
    } catch (error) {
      const isNetworkError =
        error instanceof Error &&
        (error.message.includes('fetch') || error.message.includes('network'))
      errorMessage.value = isNetworkError
        ? 'Network error. Check your connection and try again.'
        : error instanceof Error
          ? error.message
          : 'No teachers available for those time slots. Try adding more time options or different days.'
    } finally {
      isEvaluating.value = false
    }
  }

  const retryAISearch = () => {
    handleGetAISuggestions()
  }

  initWatchers()
</script>

<template>
  <PageLayout title="Book a Session">
    <div class="booking-container">
      <!-- Mobile Layout (≤425px) -->
      <div v-if="isMobile" class="mobile-layout">
        <!-- Form Section -->
        <div class="mobile-form-section">
          <!-- Subject - Full Width -->
          <div class="mobile-form-field mobile-form-field--full">
            <label class="mobile-label">
              <span class="mobile-label-dot"></span>
              <span>SUBJECT</span>
            </label>
            <select v-model="selectedSubjectId" class="mobile-select" aria-label="Select subject">
              <option :value="null">Select subject</option>
              <option v-for="subject in subjects" :key="subject.id" :value="subject.id">
                {{ subject.name }}
              </option>
            </select>
          </div>

          <!-- Branch + Teacher - 2 Columns -->
          <div class="mobile-form-row">
            <div class="mobile-form-field">
              <label class="mobile-label">
                <span class="mobile-label-dot"></span>
                <span>BRANCH</span>
              </label>
              <select
                v-model="selectedBranchId"
                :disabled="!selectedSubjectId"
                class="mobile-select"
                aria-label="Select branch"
              >
                <option :value="null">Select branch</option>
                <option v-for="branch in branches" :key="branch.id" :value="branch.id">
                  {{ branch.name }}
                </option>
              </select>
            </div>
            <div class="mobile-form-field">
              <label class="mobile-label">
                <span class="mobile-label-dot"></span>
                <span>TEACHER</span>
              </label>
              <select
                v-model="selectedTeacherId"
                :disabled="!selectedSubjectId"
                class="mobile-select"
                aria-label="Select teacher"
              >
                <option :value="null">All teachers</option>
                <option v-for="teacher in filteredTeachers" :key="teacher.id" :value="teacher.id">
                  {{ teacher.name }}
                </option>
              </select>
            </div>
          </div>
        </div>

        <!-- Availability Section -->
        <div class="mobile-availability-section">
          <div class="mobile-availability-header">
            <div class="mobile-availability-titles">
              <h2 class="mobile-section-title">Availability</h2>
              <p class="mobile-section-subtitle">Select your preferred time slot</p>
            </div>
            <button
              type="button"
              class="mobile-ai-button"
              aria-label="Open smart booking suggestions"
              @click="openAIMode"
            >
              <svg
                class="mobile-ai-icon"
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

          <!-- Empty State -->
          <div v-if="!selectedSubjectId" class="mobile-empty-state">
            <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="1.5"
                d="M15.042 21.672L13.684 16.6m0 0l-2.51 2.225.569-9.47 5.227 7.917-3.286-.672zM12 2.25V4.5m5.834.166l-1.591 1.591M20.25 10.5H18M7.757 14.743l-1.59 1.59M6 10.5H3.75m4.007-4.243l-1.59-1.59"
              />
            </svg>
            <button class="mobile-empty-btn" disabled>Select a subject to view availability</button>
          </div>

          <!-- Calendar -->
          <div v-else class="mobile-calendar-wrapper">
            <Calendar
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
          </div>
        </div>

        <!-- Mobile AI Panel -->
        <div v-if="aiMode !== 'idle'" class="mobile-ai-panel">
          <div class="mobile-ai-panel-header">
            <div>
              <h3 class="mobile-ai-panel-title">Smart Suggestions</h3>
              <p class="mobile-ai-panel-subtitle">
                Find available teachers for your preferred times
              </p>
            </div>
            <button
              type="button"
              class="mobile-ai-close-btn"
              aria-label="Close suggestions"
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

          <div v-if="!showDetailedResults && !isEvaluating" class="mobile-ai-form">
            <div class="mobile-ai-field">
              <label class="mobile-ai-label" for="mobile-ai-subject-select"
                >Subject <span class="mobile-ai-required">*</span></label
              >
              <select
                id="mobile-ai-subject-select"
                v-model="selectedSubjectId"
                class="mobile-ai-select"
              >
                <option :value="null">Select subject</option>
                <option v-for="subject in subjects" :key="subject.id" :value="subject.id">
                  {{ subject.name }}
                </option>
              </select>
            </div>

            <div class="mobile-ai-field">
              <label class="mobile-ai-label"
                >Preferred Time Slots <span class="mobile-ai-required">*</span></label
              >
              <p class="mobile-ai-hint">Add the days and times you're free</p>

              <div class="mobile-slot-controls">
                <select
                  v-model.number="aiNewSlotDay"
                  class="mobile-ai-time-select"
                  :disabled="!canAddTimeSlots"
                  aria-label="Day of week"
                >
                  <option value="0">Sun</option>
                  <option value="1">Mon</option>
                  <option value="2">Tue</option>
                  <option value="3">Wed</option>
                  <option value="4">Thu</option>
                  <option value="5">Fri</option>
                  <option value="6">Sat</option>
                </select>
                <select
                  v-model="aiNewSlotStart"
                  class="mobile-ai-time-select"
                  :disabled="!canAddTimeSlots"
                  aria-label="Start time"
                >
                  <option v-for="time in TIME_OPTIONS" :key="time" :value="time">
                    {{ time }}
                  </option>
                </select>
                <span class="mobile-ai-separator">to</span>
                <select
                  v-model="aiNewSlotEnd"
                  class="mobile-ai-time-select"
                  :disabled="!canAddTimeSlots"
                  aria-label="End time"
                >
                  <option v-for="time in TIME_OPTIONS" :key="time" :value="time">
                    {{ time }}
                  </option>
                </select>
                <button
                  type="button"
                  class="mobile-ai-add-slot-btn"
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

              <div v-if="aiTimeSlots.length > 0" class="mobile-ai-slot-list">
                <div v-for="(slot, index) in aiTimeSlots" :key="index" class="mobile-ai-slot-item">
                  <span class="mobile-ai-slot-text"
                    >{{ getDayName(slot.day_of_week) }} {{ slot.start }} - {{ slot.end }}</span
                  >
                  <button
                    type="button"
                    class="mobile-ai-slot-remove"
                    :aria-label="`Remove ${getDayName(slot.day_of_week)} ${slot.start} - ${slot.end}`"
                    @click="removeAiTimeSlot(index)"
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

            <div class="mobile-ai-field">
              <label class="mobile-ai-label" for="mobile-ai-branch-select"
                >Branch <span class="mobile-ai-required">*</span></label
              >
              <select
                id="mobile-ai-branch-select"
                v-model="selectedBranchId"
                :disabled="!selectedSubjectId"
                class="mobile-ai-select"
              >
                <option :value="null">Select branch</option>
                <option v-for="branch in branches" :key="branch.id" :value="branch.id">
                  {{ branch.name }}
                </option>
              </select>
            </div>

            <button
              type="button"
              class="mobile-ai-submit-btn"
              :disabled="
                !selectedSubjectId || !selectedBranchId || aiTimeSlots.length === 0 || isEvaluating
              "
              :aria-busy="isEvaluating"
              @click="handleGetAISuggestions"
            >
              <span v-if="isEvaluating">Finding teachers...</span>
              <span v-else>Find Available Teachers</span>
              <svg
                v-if="!isEvaluating"
                class="mobile-ai-submit-icon"
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

      <!-- Tablet Layout (426px - 768px) -->
      <div v-else-if="isTablet" class="tablet-layout">
        <!-- Form Section -->
        <div class="tablet-form-section">
          <div class="tablet-form-row">
            <div class="tablet-form-field">
              <label class="tablet-label">Subject</label>
              <select v-model="selectedSubjectId" class="tablet-select" aria-label="Select subject">
                <option :value="null">Select subject</option>
                <option v-for="subject in subjects" :key="subject.id" :value="subject.id">
                  {{ subject.name }}
                </option>
              </select>
            </div>
            <div class="tablet-form-field">
              <label class="tablet-label">Branch</label>
              <select
                v-model="selectedBranchId"
                :disabled="!selectedSubjectId"
                class="tablet-select"
                aria-label="Select branch"
              >
                <option :value="null">Select branch</option>
                <option v-for="branch in branches" :key="branch.id" :value="branch.id">
                  {{ branch.name }}
                </option>
              </select>
            </div>
            <div class="tablet-form-field">
              <label class="tablet-label">Teacher</label>
              <select
                v-model="selectedTeacherId"
                :disabled="!selectedSubjectId"
                class="tablet-select"
                aria-label="Select teacher"
              >
                <option :value="null">All teachers</option>
                <option v-for="teacher in filteredTeachers" :key="teacher.id" :value="teacher.id">
                  {{ teacher.name }}
                </option>
              </select>
            </div>
          </div>
        </div>

        <!-- Availability Section -->
        <div class="tablet-availability-section">
          <div class="tablet-availability-header">
            <div class="tablet-availability-titles">
              <h2 class="tablet-section-title">Availability</h2>
              <p class="tablet-section-subtitle">Select your preferred time slot</p>
            </div>
            <button
              type="button"
              class="tablet-ai-button"
              aria-label="Open smart booking suggestions"
              @click="openAIMode"
            >
              <span class="tablet-ai-text">Smart Suggestions</span>
              <svg
                class="tablet-ai-icon"
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

          <!-- Empty State -->
          <div v-if="!selectedSubjectId" class="tablet-empty-state">
            <svg class="tablet-empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="1.5"
                d="M15.042 21.672L13.684 16.6m0 0l-2.51 2.225.569-9.47 5.227 7.917-3.286-.672zM12 2.25V4.5m5.834.166l-1.591 1.591M20.25 10.5H18M7.757 14.743l-1.59 1.59M6 10.5H3.75m4.007-4.243l-1.59-1.59"
              />
            </svg>
            <p class="tablet-empty-text">Select a subject to view availability</p>
          </div>

          <!-- Calendar -->
          <div v-else class="tablet-calendar-wrapper">
            <Calendar
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
          </div>
        </div>

        <!-- Selected Slots & Add Button -->
        <div v-if="events.length > 0" class="tablet-action-section">
          <div class="tablet-slots-info">
            <span class="tablet-slots-count">{{ events.length }}</span>
            <span class="tablet-slots-label">slot{{ events.length > 1 ? 's' : '' }} selected</span>
          </div>
          <button
            type="button"
            class="tablet-add-btn"
            :disabled="!selectedTeacherId"
            @click="
              events.forEach((e) => {
                const teacher = filteredTeachers.find((t) => t.id === selectedTeacherId)
                if (teacher) {
                  addToCartDirectly(teacher.id, teacher.name, e.start as string, e.end as string)
                }
              })
            "
          >
            <svg class="tablet-add-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M12 4.5v15m7.5-7.5h-15"
              />
            </svg>
            <span>{{ selectedTeacherId ? 'Add to Cart' : 'Select a Teacher' }}</span>
          </button>
        </div>

        <!-- Tablet AI Panel -->
        <div v-if="aiMode !== 'idle'" class="tablet-ai-panel">
          <div class="tablet-ai-panel-header">
            <div>
              <h3 class="tablet-ai-title">Smart Suggestions</h3>
              <p class="tablet-ai-subtitle">Find available teachers for your preferred times</p>
            </div>
            <button
              type="button"
              class="tablet-ai-close"
              aria-label="Close suggestions"
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

          <div v-if="!showDetailedResults && !isEvaluating" class="tablet-ai-form">
            <div class="tablet-ai-field">
              <label class="tablet-ai-label" for="tablet-ai-subject-select"
                >Subject <span class="tablet-ai-required">*</span></label
              >
              <select
                id="tablet-ai-subject-select"
                v-model="selectedSubjectId"
                class="tablet-ai-select"
              >
                <option :value="null">Select subject</option>
                <option v-for="subject in subjects" :key="subject.id" :value="subject.id">
                  {{ subject.name }}
                </option>
              </select>
            </div>

            <div class="tablet-ai-field">
              <label class="tablet-ai-label"
                >Preferred Time Slots <span class="tablet-ai-required">*</span></label
              >
              <div class="tablet-slot-controls">
                <select
                  v-model.number="aiNewSlotDay"
                  class="tablet-ai-time-select"
                  :disabled="!canAddTimeSlots"
                  aria-label="Day of week"
                >
                  <option value="0">Sunday</option>
                  <option value="1">Monday</option>
                  <option value="2">Tuesday</option>
                  <option value="3">Wednesday</option>
                  <option value="4">Thursday</option>
                  <option value="5">Friday</option>
                  <option value="6">Saturday</option>
                </select>
                <select
                  v-model="aiNewSlotStart"
                  class="tablet-ai-time-select"
                  :disabled="!canAddTimeSlots"
                  aria-label="Start time"
                >
                  <option v-for="time in TIME_OPTIONS" :key="time" :value="time">
                    {{ time }}
                  </option>
                </select>
                <span class="tablet-ai-separator">to</span>
                <select
                  v-model="aiNewSlotEnd"
                  class="tablet-ai-time-select"
                  :disabled="!canAddTimeSlots"
                  aria-label="End time"
                >
                  <option v-for="time in TIME_OPTIONS" :key="time" :value="time">
                    {{ time }}
                  </option>
                </select>
                <button
                  type="button"
                  class="tablet-ai-add-slot"
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

              <div v-if="aiTimeSlots.length > 0" class="tablet-ai-slot-list">
                <div v-for="(slot, index) in aiTimeSlots" :key="index" class="tablet-ai-slot-item">
                  <span class="tablet-ai-slot-text"
                    >{{ getDayName(slot.day_of_week) }} {{ slot.start }} - {{ slot.end }}</span
                  >
                  <button
                    type="button"
                    class="tablet-ai-slot-remove"
                    :aria-label="`Remove ${getDayName(slot.day_of_week)} ${slot.start} - ${slot.end}`"
                    @click="removeAiTimeSlot(index)"
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

            <div class="tablet-ai-field">
              <label class="tablet-ai-label" for="tablet-ai-branch-select"
                >Branch <span class="tablet-ai-required">*</span></label
              >
              <select
                id="tablet-ai-branch-select"
                v-model="selectedBranchId"
                :disabled="!selectedSubjectId"
                class="tablet-ai-select"
              >
                <option :value="null">Select branch</option>
                <option v-for="branch in branches" :key="branch.id" :value="branch.id">
                  {{ branch.name }}
                </option>
              </select>
            </div>

            <div class="tablet-ai-field">
              <label class="tablet-ai-label" for="tablet-ai-teacher-select"
                >Teacher (Optional)</label
              >
              <select
                id="tablet-ai-teacher-select"
                v-model="selectedTeacherId"
                :disabled="!selectedSubjectId"
                class="tablet-ai-select"
              >
                <option :value="null">Show all available teachers</option>
                <option v-for="teacher in filteredTeachers" :key="teacher.id" :value="teacher.id">
                  {{ teacher.name }}
                </option>
              </select>
            </div>

            <button
              type="button"
              class="tablet-ai-submit"
              :disabled="
                !selectedSubjectId || !selectedBranchId || aiTimeSlots.length === 0 || isEvaluating
              "
              :aria-busy="isEvaluating"
              @click="handleGetAISuggestions"
            >
              <span v-if="isEvaluating">Finding teachers...</span>
              <span v-else>Find Available Teachers</span>
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

      <!-- Desktop Layout (>768px) -->
      <div v-else class="booking-layout" :class="{ 'booking-layout--split': aiMode !== 'idle' }">
        <!-- Calendar Section -->
        <div
          class="calendar-section"
          :class="{ 'calendar-section--hidden-mobile': isMobile && mobileStep === 'selection' }"
        >
          <div class="section-header">
            <div class="header-title">
              <h2 class="title">Availability</h2>
              <p class="subtitle">Select your preferred time slot</p>
            </div>
            <div class="header-actions">
              <!-- Mobile: Edit Selection button -->
              <button
                v-if="isMobile && mobileStep === 'calendar' && aiMode === 'idle'"
                type="button"
                class="edit-selection-btn"
                @click="backToSelection"
              >
                <svg class="edit-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M10.5 19.5L3 12m0 0l7.5-7.5M3 12h18"
                  />
                </svg>
                Edit Selection
              </button>
              <button
                v-if="aiMode === 'idle' && (!isMobile || mobileStep === 'selection')"
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
          </div>

          <!-- Mobile: Calendar hint -->
          <div v-if="isMobile && mobileStep === 'calendar'" class="mobile-calendar-hint">
            <svg class="hint-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09z"
              />
            </svg>
            <span>Tap & drag on a time slot to select</span>
          </div>

          <div class="calendar-wrapper">
            <div class="calendar-inner">
              <Calendar
                v-if="selectedSubjectId"
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

          <!-- Mobile: Add to Cart section (shows after selecting time slots) -->
          <div
            v-if="isMobile && mobileStep === 'calendar' && events.length > 0"
            class="mobile-add-to-cart"
          >
            <div class="mobile-slots-info">
              <span class="mobile-slots-count">{{ events.length }}</span>
              <span class="mobile-slots-label"
                >slot{{ events.length > 1 ? 's' : '' }} selected</span
              >
            </div>
            <button
              type="button"
              class="mobile-add-button"
              :disabled="!selectedTeacherId"
              @click="
                events.forEach((e) => {
                  const teacher = filteredTeachers.find((t) => t.id === selectedTeacherId)
                  if (teacher) {
                    addToCartDirectly(teacher.id, teacher.name, e.start as string, e.end as string)
                  }
                })
              "
            >
              <svg class="mobile-add-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M12 4.5v15m7.5-7.5h-15"
                />
              </svg>
              <span>{{ selectedTeacherId ? 'Add to Cart' : 'Select a Teacher' }}</span>
            </button>
          </div>
        </div>

        <!-- Manual Booking Panel -->
        <div
          v-if="aiMode === 'idle' && (!isMobile || mobileStep === 'selection')"
          class="control-panel"
        >
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
                    <option :value="null">Show all available teachers</option>
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

              <!-- Mobile: View Availability button -->
              <button
                v-if="isMobile && mobileStep === 'selection'"
                type="button"
                class="view-availability-btn"
                :disabled="!canProceedToCalendar"
                @click="proceedToCalendar"
              >
                <span>View Availability</span>
                <svg class="view-arrow-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M13.5 4.5L21 12m0 0l-7.5 7.5M21 12H3"
                  />
                </svg>
              </button>
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
              <p v-if="events.length > 0" class="slots-guidance">
                Select a teacher and branch, then add to cart
              </p>
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
        <div v-if="aiMode !== 'idle'" class="control-panel control-panel--ai">
          <div class="panel-content panel-content--ai">
            <div class="ai-header">
              <div>
                <h3 class="ai-title">Smart Suggestions</h3>
                <p class="ai-description">
                  Tell us when you're free, and we'll find available teachers for those times.
                </p>
              </div>
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
                <select id="ai-subject-select" v-model="selectedSubjectId" class="ai-select">
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
                <p class="ai-hint">
                  Add the days and times you're free. We'll match you with available teachers.
                </p>
                <div
                  class="slot-builder"
                  :class="{ 'slot-builder--disabled': !canAddTimeSlots }"
                  role="group"
                  aria-labelledby="ai-slots-label"
                >
                  <div v-if="!canAddTimeSlots" class="slot-disabled-msg">
                    Choose a subject first, then add your preferred times
                  </div>
                  <div class="slot-controls-label">Add a time slot:</div>
                  <div
                    class="slot-controls"
                    :class="{ 'slot-controls--disabled': !canAddTimeSlots }"
                  >
                    <select
                      v-model.number="aiNewSlotDay"
                      class="time-select"
                      :disabled="!canAddTimeSlots"
                      aria-label="Day of week"
                    >
                      <option value="0">Sunday</option>
                      <option value="1">Monday</option>
                      <option value="2">Tuesday</option>
                      <option value="3">Wednesday</option>
                      <option value="4">Thursday</option>
                      <option value="5">Friday</option>
                      <option value="6">Saturday</option>
                    </select>
                    <select
                      v-model="aiNewSlotStart"
                      class="time-select"
                      :disabled="!canAddTimeSlots"
                      aria-label="Start time"
                    >
                      <option v-for="time in TIME_OPTIONS" :key="time" :value="time">
                        {{ time }}
                      </option>
                    </select>
                    <span class="time-separator">to</span>
                    <select
                      v-model="aiNewSlotEnd"
                      class="time-select"
                      :disabled="!canAddTimeSlots"
                      aria-label="End time"
                    >
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
                    <div class="slot-list-label">Your time slots:</div>
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
                  v-model="selectedBranchId"
                  :disabled="!selectedSubjectId"
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
                  v-model="selectedTeacherId"
                  :disabled="!selectedSubjectId"
                  class="ai-select"
                >
                  <option :value="null">Show all available teachers</option>
                  <option v-for="teacher in filteredTeachers" :key="teacher.id" :value="teacher.id">
                    {{ teacher.name }}
                  </option>
                </select>
              </div>

              <button
                ref="aiSubmitButtonRef"
                type="button"
                class="ai-submit"
                :disabled="
                  !selectedSubjectId ||
                  !selectedBranchId ||
                  aiTimeSlots.length === 0 ||
                  isEvaluating
                "
                :aria-busy="isEvaluating"
                @click="handleGetAISuggestions"
              >
                <span v-if="isEvaluating">Finding teachers...</span>
                <span v-else>Find Available Teachers</span>
                <svg
                  v-if="!isEvaluating"
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
                <svg
                  v-else
                  class="submit-icon submit-spinner"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  aria-hidden="true"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M12 2v4m0 12v4M4.93 4.93l2.83 2.83m8.48 8.48l2.83 2.83M2 12h4m12 0h4M4.93 19.07l2.83-2.83m8.48-8.48l2.83-2.83"
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

      <!-- Mobile Sticky Bottom Button -->
      <button
        v-if="isMobile"
        type="button"
        class="mobile-sticky-add-btn"
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
        <svg class="mobile-add-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M12 4.5v15m7.5-7.5h-15"
          />
        </svg>
        <span>Add to Cart</span>
      </button>

      <!-- Tablet Sticky Bottom Button -->
      <button
        v-else-if="isTablet"
        type="button"
        class="tablet-sticky-add-btn"
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
        <svg class="tablet-sticky-add-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M12 4.5v15m7.5-7.5h-15"
          />
        </svg>
        <span>Add to Cart</span>
      </button>

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
        <span class="toast-message">{{ errorMessage }}</span>
        <div class="toast-actions">
          <button
            v-if="errorMessage.includes('Network')"
            class="toast-retry"
            aria-label="Retry the request"
            @click="retryAISearch"
          >
            Retry
          </button>
          <button class="toast-close" aria-label="Dismiss message" @click="errorMessage = ''">
            ×
          </button>
        </div>
      </div>
    </div>
  </PageLayout>
</template>

<style src="../css/BookingView.css"></style>
