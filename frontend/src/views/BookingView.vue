<script setup lang="ts">
  import { ref, computed, nextTick, onMounted, onUnmounted } from 'vue'
  import { useBooking } from '../composables/useBooking'
  import PageLayout from '../components/PageLayout.vue'
  import Calendar from '../components/Calendar.vue'
  import CalendarDisabledOverlay from '../components/CalendarDisabledOverlay.vue'
  import SmartSuggestionsPanel from '../components/SmartSuggestionsPanel.vue'
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

  // Request token to guard against stale AI responses after panel is closed
  const currentAiRequestId = ref<string | null>(null)

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
  const previouslyFocusedElement = ref<HTMLElement | null>(null)

  // Keyboard shortcuts
  const handleKeyDown = (event: KeyboardEvent) => {
    // Cmd/Ctrl + K to open AI mode
    if ((event.metaKey || event.ctrlKey) && event.key === 'k') {
      event.preventDefault()
      if (aiMode.value === 'idle') {
        openAIMode()
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
    currentAiRequestId.value = null // Invalidate any in-flight AI requests
    resetBookingState()
    previouslyFocusedElement.value?.focus()
    previouslyFocusedElement.value = null
  }

  const handleResetAIResults = () => {
    showDetailedResults.value = false
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

    // Calculate duration from the first AI time slot
    const toMinutes = (timeStr: string) => {
      const [hour, minute] = timeStr.split(':').map(Number)
      return hour * 60 + minute
    }

    // Validate all slots have the same duration
    const durations = new Set(
      aiTimeSlots.value.map((slot) => toMinutes(slot.end) - toMinutes(slot.start))
    )
    if (durations.size !== 1) {
      errorMessage.value = 'All AI time slots must use the same duration.'
      return
    }
    const aiDuration = [...durations][0]

    isEvaluating.value = true
    errorMessage.value = ''
    successMessage.value = ''

    // Generate unique request ID to guard against stale responses
    const requestId = Math.random().toString(36).substring(7)
    currentAiRequestId.value = requestId

    try {
      const { bookingApi } = await import('../services/bookingApi')
      const response = await bookingApi.evaluate({
        subject_id: selectedSubjectId.value!,
        branch_id: selectedBranchId.value!,
        preferred_slots: aiTimeSlots.value,
        duration_minutes: aiDuration,
        preferred_teacher_id: selectedTeacherId.value ?? undefined,
      })

      // Only apply results if this is still the current request
      if (currentAiRequestId.value !== requestId) {
        return // Request was cancelled/invalidated by closing the panel
      }

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

  // Add all selected slots to cart for a given teacher
  const addAllSlotsToCart = (teacherId: number | null) => {
    if (!teacherId) return
    const teacher = filteredTeachers.value.find((t) => t.id === teacherId)
    if (!teacher) return
    events.value.forEach((e) => {
      addToCartDirectly(teacher.id, teacher.name, e.start as string, e.end as string)
    })
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
              :editable="!!selectedBranchId && !!selectedTeacherId"
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
        <SmartSuggestionsPanel
          v-if="aiMode !== 'idle'"
          v-model="aiTimeSlots"
          :selected-subject-id="selectedSubjectId"
          :selected-branch-id="selectedBranchId"
          :selected-teacher-id="selectedTeacherId"
          :subjects="subjects"
          :branches="branches"
          :filtered-teachers="filteredTeachers"
          :suggestions="suggestions"
          :show-detailed-results="showDetailedResults"
          :is-evaluating="isEvaluating"
          layout="mobile"
          :is-mobile="isMobile"
          @update:selected-subject-id="(v) => (selectedSubjectId = v)"
          @update:selected-branch-id="(v) => (selectedBranchId = v)"
          @update:selected-teacher-id="(v) => (selectedTeacherId = v)"
          @submit="handleGetAISuggestions"
          @close="closeAIMode"
          @reset="handleResetAIResults"
          @confirm-booking="handleAIBooking"
        />
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
              :editable="!!selectedBranchId && !!selectedTeacherId"
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
            @click="addAllSlotsToCart(selectedTeacherId)"
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
        <SmartSuggestionsPanel
          v-if="aiMode !== 'idle'"
          v-model="aiTimeSlots"
          :selected-subject-id="selectedSubjectId"
          :selected-branch-id="selectedBranchId"
          :selected-teacher-id="selectedTeacherId"
          :subjects="subjects"
          :branches="branches"
          :filtered-teachers="filteredTeachers"
          :suggestions="suggestions"
          :show-detailed-results="showDetailedResults"
          :is-evaluating="isEvaluating"
          layout="tablet"
          @update:selected-subject-id="(v) => (selectedSubjectId = v)"
          @update:selected-branch-id="(v) => (selectedBranchId = v)"
          @update:selected-teacher-id="(v) => (selectedTeacherId = v)"
          @submit="handleGetAISuggestions"
          @close="closeAIMode"
          @reset="handleResetAIResults"
          @confirm-booking="handleAIBooking"
        />
      </div>

      <!-- Desktop Layout (>768px) -->
      <div v-else class="booking-layout">
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
              <!-- Smart Suggestions button -->
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
                :editable="!!selectedBranchId && !!selectedTeacherId"
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
              @click="addAllSlotsToCart(selectedTeacherId)"
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
              @click="addAllSlotsToCart(selectedTeacherId)"
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
          <SmartSuggestionsPanel
            v-model="aiTimeSlots"
            :selected-subject-id="selectedSubjectId"
            :selected-branch-id="selectedBranchId"
            :selected-teacher-id="selectedTeacherId"
            :subjects="subjects"
            :branches="branches"
            :filtered-teachers="filteredTeachers"
            :suggestions="suggestions"
            :show-detailed-results="showDetailedResults"
            :is-evaluating="isEvaluating"
            layout="desktop"
            @update:selected-subject-id="(v) => (selectedSubjectId = v)"
            @update:selected-branch-id="(v) => (selectedBranchId = v)"
            @update:selected-teacher-id="(v) => (selectedTeacherId = v)"
            @submit="handleGetAISuggestions"
            @close="closeAIMode"
            @reset="handleResetAIResults"
            @confirm-booking="handleAIBooking"
          />
        </div>
      </div>

      <!-- Desktop AI Panel Overlay (remove this, now using swap) -->
      <!-- Mobile Sticky Bottom Button -->
      <button
        v-if="isMobile"
        type="button"
        class="mobile-sticky-add-btn"
        :disabled="
          !selectedSubjectId || !selectedBranchId || !selectedTeacherId || events.length === 0
        "
        @click="addAllSlotsToCart(selectedTeacherId)"
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
        @click="addAllSlotsToCart(selectedTeacherId)"
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
