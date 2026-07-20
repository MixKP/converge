<script setup lang="ts">
  import { ref, computed, nextTick, onMounted } from 'vue'
  import { useForm } from 'vee-validate'
  import { useBooking } from '../composables/useBooking'
  import { useCart } from '../composables/useCart'
  import { useNotification } from '../composables/useNotification'
  import { useScreenSize } from '../composables/useScreenSize'
  import { useAISuggestions } from '../composables/useAISuggestions'
  import PageLayout from '../components/PageLayout.vue'
  import Calendar from '../components/Calendar.vue'
  import CalendarDisabledOverlay from '../components/CalendarDisabledOverlay.vue'
  import SmartSuggestionsPanel from '../components/SmartSuggestionsPanel.vue'
  import FormSelect from '../components/form/FormSelect.vue'
  import { manualBookingFormSchema } from '../schemas/forms'
  import { toTypedSchema } from '../schemas/veeValidate'
  import type { EventClickArg } from '@fullcalendar/core'

  const {
    calendarRef,
    isEvaluating,
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
    filteredSuggestionEvents,
    allEvents,
    initialize,
    resetBookingState,
    fetchSubjects,
    fetchBranches,
  } = useBooking()

  const { cartItems, loadCart, addSlotToCart } = useCart()
  const { successMessage, errorMessage, showSuccess } = useNotification()
  const { isMobile, isTablet } = useScreenSize()
  const { getSuggestions, invalidateRequest } = useAISuggestions()

  const allCalendarEvents = computed(() => allEvents.value)

  const aiMode = ref<'idle' | 'expanding' | 'expanded'>('idle')

  const isAddingToCart = ref(false)

  const { meta: bookingFormMeta, submitCount } = useForm({
    validationSchema: toTypedSchema(manualBookingFormSchema),
    keepValuesOnUnmount: true,
    initialValues: {
      subject_id: selectedSubjectId.value,
      branch_id: selectedBranchId.value,
      teacher_id: selectedTeacherId.value,
    },
  })

  const showErrors = computed(() => submitCount.value > 0)

  const canAddToBooking = computed(
    () => bookingFormMeta.value.valid && events.value.length > 0 && !isAddingToCart.value
  )

  const aiTimeSlots = ref<Array<{ day_of_week: number; start: string; end: string }>>([])

  onMounted(() => {
    loadCart()
    fetchSubjects()
    fetchBranches()
  })

  const previouslyFocusedElement = ref<HTMLElement | null>(null)

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
    invalidateRequest()
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
      showSuccess('Click "Book Now" in Smart Suggestions to book this slot', 3000)
    }
  }

  const handleAIBooking = (
    teacherId: number,
    teacherName: string,
    startTime: string,
    endTime: string
  ): void => {
    addSlotToCart(
      teacherId,
      teacherName,
      startTime,
      endTime,
      selectedSubjectId.value ?? undefined,
      selectedBranchId.value ?? undefined
    )
  }

  const handleGetAISuggestions = () => {
    getSuggestions(aiTimeSlots.value)
  }

  const retryAISearch = () => {
    getSuggestions(aiTimeSlots.value)
  }

  const addAllSlotsToCart = (teacherId: number | null) => {
    if (!teacherId) return
    if (isAddingToCart.value) return

    const teacher = filteredTeachers.value.find((t) => t.id === teacherId)
    if (!teacher) return

    isAddingToCart.value = true
    try {
      events.value.forEach((e) => {
        addSlotToCart(teacher.id, teacher.name, e.start as string, e.end as string)
      })
    } finally {
      setTimeout(() => {
        isAddingToCart.value = false
      }, 300)
    }
  }

  initialize()
</script>

<template>
  <PageLayout title="Book a Session">
    <div class="relative h-full overflow-hidden px-6 lg:px-8">
      <!-- Mobile Layout (≤425px) -->
      <div
        v-if="isMobile"
        class="flex max-w-full flex-col gap-3.5 overflow-x-hidden p-4 pb-[calc(72px+env(safe-area-inset-bottom))]"
      >
        <!-- Form Section -->
        <div class="flex flex-col gap-4">
          <!-- Subject - Full Width -->
          <div class="flex flex-col gap-2">
            <label
              class="flex items-center gap-2 font-[Inter,sans-serif] text-[0.6875rem] font-semibold tracking-[0.08em] text-(--text-muted) uppercase"
            >
              <span class="h-1 w-1 rounded-full bg-(--text-muted)"></span>
              <span>SUBJECT</span>
            </label>
            <FormSelect
              v-model="selectedSubjectId"
              name="subject_id"
              select-class="mobile-select"
              aria-label="Select subject"
              :show-error="showErrors"
            >
              <option :value="null">Select subject</option>
              <option v-for="subject in subjects" :key="subject.id" :value="subject.id">
                {{ subject.name }}
              </option>
            </FormSelect>
          </div>

          <!-- Branch + Teacher - 2 Columns -->
          <div class="grid grid-cols-2 gap-3">
            <div class="flex flex-col gap-2">
              <label
                class="flex items-center gap-2 font-[Inter,sans-serif] text-[0.6875rem] font-semibold tracking-[0.08em] text-(--text-muted) uppercase"
              >
                <span class="h-1 w-1 rounded-full bg-(--text-muted)"></span>
                <span>BRANCH</span>
              </label>
              <FormSelect
                v-model="selectedBranchId"
                name="branch_id"
                :disabled="!selectedSubjectId"
                select-class="mobile-select"
                aria-label="Select branch"
                :show-error="showErrors"
              >
                <option :value="null">Select branch</option>
                <option v-for="branch in branches" :key="branch.id" :value="branch.id">
                  {{ branch.name }}
                </option>
              </FormSelect>
            </div>
            <div class="flex flex-col gap-2">
              <label
                class="flex items-center gap-2 font-[Inter,sans-serif] text-[0.6875rem] font-semibold tracking-[0.08em] text-(--text-muted) uppercase"
              >
                <span class="h-1 w-1 rounded-full bg-(--text-muted)"></span>
                <span>TEACHER</span>
              </label>
              <FormSelect
                v-model="selectedTeacherId"
                name="teacher_id"
                :disabled="!selectedSubjectId"
                select-class="mobile-select"
                aria-label="Select teacher"
                :show-error="showErrors"
              >
                <option :value="null">All teachers</option>
                <option v-for="teacher in filteredTeachers" :key="teacher.id" :value="teacher.id">
                  {{ teacher.name }}
                </option>
              </FormSelect>
            </div>
          </div>
        </div>

        <!-- Availability Section -->
        <div class="flex flex-col gap-3">
          <div class="flex items-start justify-between gap-3">
            <div class="flex-1">
              <h2
                class="m-0 font-['Instrument_Sans',sans-serif] text-lg font-semibold text-(--text-primary)"
              >
                Availability
              </h2>
              <p class="m-0 font-[Inter,sans-serif] text-[0.8125rem] text-(--text-secondary)">
                Select your preferred time slot
              </p>
            </div>
            <button
              type="button"
              class="flex h-9 w-9 shrink-0 touch-manipulation items-center justify-center rounded-lg border border-(--primary-indigo) bg-[rgba(45,74,62,0.08)] p-0 transition-all duration-200 hover:scale-105 hover:bg-[rgba(45,74,62,0.15)] active:scale-95"
              aria-label="Open smart booking suggestions"
              @click="openAIMode"
            >
              <svg
                class="h-4.5 w-4.5 text-(--primary-indigo)"
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
          <div
            v-if="!selectedSubjectId"
            class="flex min-h-60 flex-col items-center justify-center gap-3.5 rounded-lg border-2 border-dashed border-(--border-medium) bg-(--bg-subtle) p-6"
          >
            <svg
              class="h-7 w-7 text-(--accent-coral) opacity-80"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="1.5"
                d="M15.042 21.672L13.684 16.6m0 0l-2.51 2.225.569-9.47 5.227 7.917-3.286-.672zM12 2.25V4.5m5.834.166l-1.591 1.591M20.25 10.5H18M7.757 14.743l-1.59 1.59M6 10.5H3.75m4.007-4.243l-1.59-1.59"
              />
            </svg>
            <button
              class="rounded-md border border-(--border-medium) bg-(--bg-card) px-3.5 py-2 font-[Inter,sans-serif] text-xs font-medium text-(--text-primary) disabled:cursor-not-allowed disabled:opacity-100"
              disabled
            >
              Select a subject to view availability
            </button>
          </div>

          <!-- Calendar -->
          <div
            v-else
            class="mobile-calendar-wrapper flex h-100 [touch-action:pan-y_pinch-zoom] flex-col overflow-hidden rounded-lg border border-(--border-subtle) bg-(--bg-card)"
          >
            <Calendar
              ref="calendarRef"
              :model-value="allCalendarEvents"
              :additional-events="filteredSuggestionEvents"
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
          :cart-items="cartItems"
          @update:selected-subject-id="(v) => (selectedSubjectId = v)"
          @update:selected-branch-id="(v) => (selectedBranchId = v)"
          @update:selected-teacher-id="(v) => (selectedTeacherId = v)"
          @submit="handleGetAISuggestions"
          @close="closeAIMode"
          @reset="handleResetAIResults"
          @confirm-booking="handleAIBooking"
        />
      </div>

      <!-- Tablet Layout (426px - 1023px) -->
      <div v-else-if="isTablet" class="flex max-w-full flex-col gap-5 overflow-x-hidden p-6">
        <!-- Form Section -->
        <div class="flex flex-col gap-4">
          <div class="grid grid-cols-3 gap-4">
            <div class="flex flex-col gap-2.5">
              <label
                class="font-[Inter,sans-serif] text-xs font-semibold tracking-wider text-(--text-muted) uppercase"
                >Subject</label
              >
              <FormSelect
                v-model="selectedSubjectId"
                name="subject_id"
                select-class="tablet-select"
                aria-label="Select subject"
                :show-error="showErrors"
              >
                <option :value="null">Select subject</option>
                <option v-for="subject in subjects" :key="subject.id" :value="subject.id">
                  {{ subject.name }}
                </option>
              </FormSelect>
            </div>
            <div class="flex flex-col gap-2.5">
              <label
                class="font-[Inter,sans-serif] text-xs font-semibold tracking-wider text-(--text-muted) uppercase"
                >Branch</label
              >
              <FormSelect
                v-model="selectedBranchId"
                name="branch_id"
                :disabled="!selectedSubjectId"
                select-class="tablet-select"
                aria-label="Select branch"
                :show-error="showErrors"
              >
                <option :value="null">Select branch</option>
                <option v-for="branch in branches" :key="branch.id" :value="branch.id">
                  {{ branch.name }}
                </option>
              </FormSelect>
            </div>
            <div class="flex flex-col gap-2.5">
              <label
                class="font-[Inter,sans-serif] text-xs font-semibold tracking-wider text-(--text-muted) uppercase"
                >Teacher</label
              >
              <FormSelect
                v-model="selectedTeacherId"
                name="teacher_id"
                :disabled="!selectedSubjectId"
                select-class="tablet-select"
                aria-label="Select teacher"
                :show-error="showErrors"
              >
                <option :value="null">All teachers</option>
                <option v-for="teacher in filteredTeachers" :key="teacher.id" :value="teacher.id">
                  {{ teacher.name }}
                </option>
              </FormSelect>
            </div>
          </div>
        </div>

        <!-- Availability Section -->
        <div class="flex flex-col gap-4">
          <div class="flex items-start justify-between gap-4">
            <div class="flex-1">
              <h2
                class="m-0 font-['Instrument_Sans',sans-serif] text-[1.375rem] font-semibold text-(--text-primary)"
              >
                Availability
              </h2>
              <p class="mt-1 font-[Inter,sans-serif] text-sm text-(--text-secondary)">
                Select your preferred time slot
              </p>
            </div>
            <button
              type="button"
              class="flex touch-manipulation items-center gap-2 rounded-[10px] border border-(--primary-indigo) bg-[rgba(45,74,62,0.08)] px-4 py-2.5 font-[Inter,sans-serif] transition-all duration-200 hover:-translate-y-px hover:bg-[rgba(45,74,62,0.15)] active:translate-y-0"
              aria-label="Open smart booking suggestions"
              @click="openAIMode"
            >
              <span class="text-sm font-medium text-(--primary-indigo)">Smart Suggestions</span>
              <svg
                class="h-4.5 w-4.5 text-(--primary-indigo)"
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
          <div
            v-if="!selectedSubjectId"
            class="flex min-h-70 flex-col items-center justify-center gap-4 rounded-xl border-2 border-dashed border-(--border-medium) bg-(--bg-subtle) p-8"
          >
            <svg
              class="h-9 w-9 text-(--accent-coral) opacity-80"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="1.5"
                d="M15.042 21.672L13.684 16.6m0 0l-2.51 2.225.569-9.47 5.227 7.917-3.286-.672zM12 2.25V4.5m5.834.166l-1.591 1.591M20.25 10.5H18M7.757 14.743l-1.59 1.59M6 10.5H3.75m4.007-4.243l-1.59-1.59"
              />
            </svg>
            <p class="m-0 font-[Inter,sans-serif] text-[0.9375rem] text-(--text-secondary)">
              Select a subject to view availability
            </p>
          </div>

          <!-- Calendar -->
          <div
            v-else
            class="tablet-calendar-wrapper flex h-105 [touch-action:pan-y_pinch-zoom] flex-col overflow-hidden rounded-xl border border-(--border-subtle) bg-(--bg-card)"
          >
            <Calendar
              ref="calendarRef"
              :model-value="allCalendarEvents"
              :additional-events="filteredSuggestionEvents"
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
        <div
          v-if="events.length > 0"
          class="flex items-center justify-between gap-4 rounded-xl border border-(--border-subtle) bg-(--bg-card) px-5 py-4"
        >
          <div class="flex items-center gap-2">
            <span
              class="font-['JetBrains_Mono',monospace] text-2xl font-semibold text-(--primary-indigo)"
              >{{ events.length }}</span
            >
            <span class="font-[Inter,sans-serif] text-sm text-(--text-secondary)"
              >slot{{ events.length > 1 ? 's' : '' }} selected</span
            >
          </div>
          <button
            type="button"
            class="flex touch-manipulation items-center gap-2 rounded-[10px] border-none bg-(--primary-indigo) px-5 py-3 font-[Inter,sans-serif] text-[0.9375rem] font-semibold text-white transition-all duration-200 enabled:hover:-translate-y-px enabled:hover:bg-(--primary-indigo-deep) enabled:active:translate-y-0 disabled:cursor-not-allowed disabled:opacity-50"
            :disabled="!selectedTeacherId || isAddingToCart"
            @click="addAllSlotsToCart(selectedTeacherId)"
          >
            <svg class="h-4.5 w-4.5" viewBox="0 0 24 24" fill="none" stroke="currentColor">
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M12 4.5v15m7.5-7.5h-15"
              />
            </svg>
            <span>{{ selectedTeacherId ? 'Add to Booking' : 'Select a Teacher' }}</span>
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
          :cart-items="cartItems"
          @update:selected-subject-id="(v) => (selectedSubjectId = v)"
          @update:selected-branch-id="(v) => (selectedBranchId = v)"
          @update:selected-teacher-id="(v) => (selectedTeacherId = v)"
          @submit="handleGetAISuggestions"
          @close="closeAIMode"
          @reset="handleResetAIResults"
          @confirm-booking="handleAIBooking"
        />
      </div>

      <!-- Desktop Layout (≥1024px) -->
      <div v-else class="relative z-1 flex flex-row gap-6 p-2">
        <!-- Calendar Section -->
        <div
          class="relative z-1 flex max-w-full flex-1 flex-col gap-4 transition-all duration-300 ease-[cubic-bezier(0.25,1,0.5,1)]"
        >
          <div class="flex items-center justify-between py-2">
            <div>
              <h2
                class="font-['Instrument_Sans',sans-serif] text-xl font-semibold tracking-[-0.01em] text-(--text-primary)"
              >
                Availability
              </h2>
              <p
                class="mt-1 font-['JetBrains_Mono',monospace] text-[0.8125rem] tracking-wider text-(--text-secondary)"
              >
                Select your preferred time slot
              </p>
            </div>
            <div class="flex items-center gap-3">
              <!-- Smart Suggestions button -->
              <button
                v-if="aiMode === 'idle'"
                type="button"
                class="flex touch-manipulation items-center gap-2 rounded-md border-none bg-(--primary-indigo) px-4 py-2.5 font-[Inter,sans-serif] text-[0.8125rem] font-medium text-white shadow-[0_2px_8px_rgba(45,74,62,0.2)] transition-all duration-200 ease-[cubic-bezier(0.25,1,0.5,1)] hover:-translate-y-px hover:bg-(--primary-indigo-deep) hover:shadow-[0_3px_12px_rgba(45,74,62,0.25)] focus-visible:shadow-[0_0_0_3px_rgba(45,74,62,0.4),0_2px_8px_rgba(45,74,62,0.2)] focus-visible:outline-none"
                aria-label="Open smart booking suggestions"
                @click="openAIMode"
              >
                <span>Smart Suggestions</span>
                <svg
                  class="h-4 w-4"
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

          <div class="relative z-1 h-155">
            <div
              class="calendar-inner relative h-full overflow-hidden rounded-lg border border-(--border-medium) bg-(--bg-card) shadow-(--shadow-card) transition-all duration-400 before:absolute before:inset-x-0 before:top-0 before:z-1 before:h-px before:bg-[linear-gradient(90deg,transparent,var(--border-technical)_20%,var(--border-technical)_80%,transparent)] before:content-['']"
            >
              <Calendar
                v-if="selectedSubjectId"
                ref="calendarRef"
                :model-value="allCalendarEvents"
                :additional-events="filteredSuggestionEvents"
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
        </div>

        <!-- Manual Booking Panel -->
        <div
          v-if="aiMode === 'idle'"
          class="flex max-h-[calc(100vh-150px)] w-85 shrink-0 flex-col overflow-hidden rounded-2xl border border-(--border-medium) bg-(--bg-card) shadow-[0_1px_3px_rgba(0,0,0,0.04),0_4px_12px_rgba(0,0,0,0.03)] transition-all duration-300"
        >
          <div class="flex h-full min-h-0 flex-col gap-4 overflow-y-auto px-4 py-4.5">
            <!-- Selection Form -->
            <div class="flex flex-col gap-3.5">
              <div class="flex flex-col gap-1.5">
                <div class="flex flex-col gap-1.5">
                  <div class="flex items-center gap-2">
                    <span
                      class="h-3 w-0.75 rounded-[1.5px] bg-(--primary-indigo) opacity-80"
                    ></span>
                    <span
                      class="font-[Inter,sans-serif] text-[0.8125rem] font-medium text-(--text-primary)"
                      >Subject</span
                    >
                  </div>
                  <FormSelect
                    v-model="selectedSubjectId"
                    name="subject_id"
                    select-class="field-select"
                    aria-label="Select subject"
                    :show-error="showErrors"
                  >
                    <option :value="null">Select subject</option>
                    <option v-for="subject in subjects" :key="subject.id" :value="subject.id">
                      {{ subject.name }}
                    </option>
                  </FormSelect>
                </div>
              </div>

              <div class="flex flex-col gap-1.5">
                <div class="flex flex-col gap-1.5">
                  <div class="flex items-center gap-2">
                    <span
                      class="h-3 w-0.75 rounded-[1.5px] bg-(--primary-indigo) opacity-80"
                    ></span>
                    <span
                      class="font-[Inter,sans-serif] text-[0.8125rem] font-medium text-(--text-primary)"
                      >Branch</span
                    >
                  </div>
                  <FormSelect
                    v-model="selectedBranchId"
                    name="branch_id"
                    :disabled="!selectedSubjectId"
                    select-class="field-select"
                    aria-label="Select branch"
                    :show-error="showErrors"
                  >
                    <option :value="null">Select branch</option>
                    <option v-for="branch in branches" :key="branch.id" :value="branch.id">
                      {{ branch.name }}
                    </option>
                  </FormSelect>
                </div>
              </div>

              <div class="flex flex-col gap-1.5">
                <div class="flex flex-col gap-1.5">
                  <div class="flex items-center gap-2">
                    <span
                      class="h-3 w-0.75 rounded-[1.5px] bg-(--primary-indigo) opacity-80"
                    ></span>
                    <span
                      class="font-[Inter,sans-serif] text-[0.8125rem] font-medium text-(--text-primary)"
                      >Teacher</span
                    >
                  </div>
                  <FormSelect
                    v-model="selectedTeacherId"
                    name="teacher_id"
                    :disabled="!selectedSubjectId"
                    select-class="field-select"
                    aria-label="Select teacher"
                    :show-error="showErrors"
                  >
                    <option :value="null">Show all available teachers</option>
                    <option
                      v-for="teacher in filteredTeachers"
                      :key="teacher.id"
                      :value="teacher.id"
                    >
                      {{ teacher.name }}
                    </option>
                  </FormSelect>
                </div>
              </div>
            </div>

            <!-- Action Button -->
            <button
              type="button"
              class="flex touch-manipulation items-center justify-center gap-2 rounded-md border-none bg-(--accent-sage) px-5 py-3.5 font-[Inter,sans-serif] text-sm font-medium text-white shadow-[0_2px_8px_rgba(122,139,109,0.2)] transition-all duration-200 ease-[cubic-bezier(0.25,1,0.5,1)] focus-visible:shadow-[0_0_0_3px_rgba(122,139,109,0.4),0_2px_8px_rgba(122,139,109,0.2)] focus-visible:outline-none enabled:hover:-translate-y-px enabled:hover:shadow-[0_3px_12px_rgba(122,139,109,0.25)] enabled:hover:brightness-90 disabled:transform-none disabled:cursor-not-allowed disabled:opacity-40"
              :disabled="!canAddToBooking"
              :aria-label="'Add to booking'"
              @click="addAllSlotsToCart(selectedTeacherId)"
            >
              <span>Add to Booking</span>
              <svg
                class="h-4 w-4"
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
        <div
          v-if="aiMode !== 'idle'"
          class="anim-panel-fade flex max-h-[calc(100vh-150px)] w-85 shrink-0 flex-col overflow-hidden rounded-2xl border border-(--primary-indigo) bg-(--bg-card) shadow-(--shadow-card) transition-all duration-300"
        >
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
            :cart-items="cartItems"
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

      <!-- Mobile Sticky Bottom Button -->
      <button
        v-if="isMobile"
        type="button"
        class="fixed right-0 bottom-[env(safe-area-inset-bottom)] left-0 z-10 flex touch-manipulation items-center justify-center gap-2 border-0 border-t border-t-[rgba(0,0,0,0.05)] bg-(--primary-indigo) px-4 pt-4 pb-[calc(1rem+env(safe-area-inset-bottom))] font-[Inter,sans-serif] text-[0.9375rem] font-medium text-white transition-all duration-150 enabled:hover:bg-(--primary-indigo-deep) enabled:active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-50"
        :disabled="!canAddToBooking"
        @click="addAllSlotsToCart(selectedTeacherId)"
      >
        <svg class="h-4.5 w-4.5" viewBox="0 0 24 24" fill="none" stroke="currentColor">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M12 4.5v15m7.5-7.5h-15"
          />
        </svg>
        <span>Add to Booking</span>
      </button>

      <!-- Tablet Sticky Bottom Button -->
      <button
        v-else-if="isTablet"
        type="button"
        class="fixed right-0 bottom-[env(safe-area-inset-bottom)] left-0 z-10 flex touch-manipulation items-center justify-center gap-2 border-0 border-t border-t-[rgba(0,0,0,0.05)] bg-(--primary-indigo) px-4 pt-4 pb-[calc(1rem+env(safe-area-inset-bottom))] font-[Inter,sans-serif] text-[0.9375rem] font-medium text-white transition-all duration-150 enabled:hover:bg-(--primary-indigo-deep) enabled:active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-50"
        :disabled="!canAddToBooking"
        @click="addAllSlotsToCart(selectedTeacherId)"
      >
        <svg class="h-4.5 w-4.5" viewBox="0 0 24 24" fill="none" stroke="currentColor">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M12 4.5v15m7.5-7.5h-15"
          />
        </svg>
        <span>Add to Booking</span>
      </button>

      <!-- Toast Notifications -->
      <div
        v-if="successMessage"
        class="anim-toast-in fixed right-8 bottom-8 z-100 flex items-center gap-3 rounded-xl bg-[linear-gradient(135deg,var(--accent-sage)_0%,var(--accent-mint)_100%)] px-6 py-4 font-[Inter,sans-serif] text-sm text-white shadow-(--shadow-elevated) max-[425px]:bottom-18 max-md:right-4 max-md:bottom-4 max-md:left-4 max-md:max-w-none"
        role="status"
        aria-live="polite"
      >
        <svg
          class="h-5 w-5 shrink-0"
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

      <div
        v-if="errorMessage"
        class="anim-toast-in fixed right-8 bottom-8 z-100 flex items-center gap-3 rounded-xl bg-[linear-gradient(135deg,var(--accent-coral)_0%,var(--accent-coral-light)_100%)] px-6 py-4 font-[Inter,sans-serif] text-sm text-white shadow-(--shadow-elevated) max-[425px]:bottom-18 max-md:right-4 max-md:bottom-4 max-md:left-4 max-md:max-w-none"
        role="alert"
        aria-live="assertive"
      >
        <svg
          class="h-5 w-5 shrink-0"
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
        <span class="min-w-0 flex-1">{{ errorMessage }}</span>
        <div class="ml-3 flex items-center gap-2">
          <button
            v-if="errorMessage.includes('Network')"
            class="cursor-pointer rounded-md border border-white/30 bg-white/20 px-3.5 py-1.5 text-xs font-semibold text-white transition-all duration-200 hover:border-white/40 hover:bg-white/30 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white/50"
            aria-label="Retry the request"
            @click="retryAISearch"
          >
            Retry
          </button>
          <button
            class="ml-3 flex h-6 w-6 cursor-pointer items-center justify-center rounded-[3px] border-none bg-transparent text-xl text-white/80 transition-all duration-200 hover:bg-white/15 hover:text-white focus-visible:bg-white/20 focus-visible:text-white focus-visible:outline-none"
            aria-label="Dismiss message"
            @click="errorMessage = ''"
          >
            ×
          </button>
        </div>
      </div>
    </div>
  </PageLayout>
</template>

<style scoped>
  @keyframes booking-toast-in {
    from {
      opacity: 0;
      transform: translateY(20px) scale(0.92);
    }
    to {
      opacity: 1;
      transform: translateY(0) scale(1);
    }
  }

  @keyframes booking-panel-fade-in {
    from {
      opacity: 0;
      transform: translateX(-8px);
    }
    to {
      opacity: 1;
      transform: translateX(0);
    }
  }

  .anim-toast-in {
    animation: booking-toast-in 0.4s cubic-bezier(0.34, 1.56, 0.64, 1);
  }

  .anim-panel-fade {
    animation: booking-panel-fade-in 0.3s cubic-bezier(0.25, 1, 0.5, 1);
  }

  .calendar-inner :deep(.fc .fc-toolbar.fc-header-toolbar) {
    margin: 0 !important;
  }

  .mobile-calendar-wrapper > div {
    height: 100% !important;
    display: flex !important;
    flex-direction: column !important;
  }

  .mobile-calendar-wrapper :deep(.border) {
    height: 100% !important;
    border: none !important;
  }

  .mobile-calendar-wrapper :deep(.fc) {
    height: 100% !important;
    font-size: 0.75rem;
    touch-action: pan-y pinch-zoom;
  }

  .mobile-calendar-wrapper :deep(.fc-view-harness),
  .mobile-calendar-wrapper :deep(.fc-view),
  .mobile-calendar-wrapper :deep(.fc-timegrid),
  .mobile-calendar-wrapper :deep(.fc-timegrid-body),
  .mobile-calendar-wrapper :deep(.fc-timegrid-slots) {
    touch-action: pan-y pinch-zoom;
  }

  .mobile-calendar-wrapper :deep(.fc-timegrid-slot),
  .mobile-calendar-wrapper :deep(.fc-timegrid-slot-lane) {
    min-height: 36px;
    touch-action: pan-y;
  }

  .mobile-calendar-wrapper :deep(.fc-col-header-cell) {
    touch-action: manipulation;
    padding: 0.25rem 0.0625rem !important;
    height: 24px !important;
  }

  .mobile-calendar-wrapper :deep(.fc-col-header-cell-cushion) {
    font-size: 0.5rem !important;
    padding: 0 !important;
    font-weight: 600 !important;
    letter-spacing: 0.05em !important;
    line-height: 1 !important;
  }

  .mobile-calendar-wrapper :deep(.fc-timegrid-axis) {
    width: 24px !important;
    touch-action: pan-y;
  }

  .mobile-calendar-wrapper :deep(.fc-timegrid-slot-label) {
    font-size: 0.5rem !important;
    padding: 0 !important;
  }

  .mobile-calendar-wrapper :deep(.fc-timegrid-slot-label-cushion) {
    padding: 0 0.0625rem !important;
    line-height: 1 !important;
  }

  .mobile-calendar-wrapper :deep(.fc-timegrid-axis-frame) {
    padding: 0 !important;
  }

  .mobile-calendar-wrapper :deep(.fc-event) {
    font-size: 0.625rem !important;
    padding: 0.25rem 0.375rem !important;
    touch-action: manipulation;
  }

  .mobile-calendar-wrapper :deep(.fc-timegrid-slot-minor),
  .mobile-calendar-wrapper :deep(.fc-timegrid-slot-major) {
    border-top-width: 1px !important;
  }

  .mobile-calendar-wrapper :deep(.fc .fc-toolbar.fc-header-toolbar) {
    padding: 0.125rem 0.375rem !important;
    gap: 0.125rem;
    margin: 0 !important;
  }

  .mobile-calendar-wrapper :deep(.fc-toolbar-chunk) {
    margin: 0 !important;
  }

  .mobile-calendar-wrapper :deep(.fc .fc-toolbar-title) {
    font-size: 0.75rem !important;
    font-weight: 600 !important;
    margin: 0 !important;
    line-height: 1.2 !important;
  }

  .mobile-calendar-wrapper :deep(.fc-button-group) {
    gap: 0.0625rem;
  }

  .mobile-calendar-wrapper :deep(.fc .fc-button) {
    min-width: 2rem;
    min-height: 2rem;
    padding: 0.25rem !important;
    background: transparent !important;
    border: none !important;
    font-size: 0.875rem !important;
  }

  .mobile-calendar-wrapper :deep(.fc-icon) {
    font-size: 1em !important;
  }

  .tablet-calendar-wrapper :deep(.fc) {
    height: 100% !important;
    font-size: 0.875rem;
    touch-action: pan-y pinch-zoom;
  }

  .tablet-calendar-wrapper :deep(.fc-view-harness),
  .tablet-calendar-wrapper :deep(.fc-view),
  .tablet-calendar-wrapper :deep(.fc-timegrid),
  .tablet-calendar-wrapper :deep(.fc-timegrid-body),
  .tablet-calendar-wrapper :deep(.fc-timegrid-slots) {
    touch-action: pan-y pinch-zoom;
  }

  .tablet-calendar-wrapper :deep(.fc-col-header-cell) {
    padding: 0.5rem 0.25rem !important;
    height: 32px !important;
    touch-action: manipulation;
  }

  .tablet-calendar-wrapper :deep(.fc-col-header-cell-cushion) {
    font-size: 0.625rem !important;
    padding: 0 0.25rem !important;
    font-weight: 600 !important;
  }

  .tablet-calendar-wrapper :deep(.fc-timegrid-axis) {
    width: 32px !important;
    touch-action: pan-y;
  }

  .tablet-calendar-wrapper :deep(.fc-timegrid-slot-label) {
    font-size: 0.625rem !important;
  }

  .tablet-calendar-wrapper :deep(.fc-timegrid-slot) {
    min-height: 40px;
    touch-action: pan-y;
  }

  .tablet-calendar-wrapper :deep(.fc-timegrid-slot-lane) {
    touch-action: pan-y;
  }

  .tablet-calendar-wrapper :deep(.fc .fc-toolbar.fc-header-toolbar) {
    padding: 0.25rem 0.5rem !important;
    gap: 0.25rem;
    margin: 0 !important;
  }

  .tablet-calendar-wrapper :deep(.fc-toolbar-chunk) {
    margin: 0 !important;
  }

  .tablet-calendar-wrapper :deep(.fc .fc-toolbar-title) {
    font-size: 0.875rem !important;
    font-weight: 600 !important;
    margin: 0 !important;
    line-height: 1.2 !important;
  }

  .tablet-calendar-wrapper :deep(.fc-button-group) {
    gap: 0.125rem;
  }

  .tablet-calendar-wrapper :deep(.fc .fc-button) {
    min-width: 2rem;
    min-height: 2rem;
    padding: 0.25rem !important;
    background: transparent !important;
    border: none !important;
    font-size: 0.875rem !important;
  }

  .tablet-calendar-wrapper :deep(.fc-icon) {
    font-size: 1em !important;
  }

  @media (min-width: 1024px) {
    .anim-panel-fade {
      animation: booking-panel-fade-in 0.3s cubic-bezier(0.25, 1, 0.5, 1);
    }
  }
</style>
