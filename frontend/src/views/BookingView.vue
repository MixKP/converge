<script setup lang="ts">
  import { onMounted } from 'vue'
  import { useBooking } from '../composables/useBooking'
  import PageLayout from '../components/PageLayout.vue'
  import Calendar from '../components/Calendar.vue'
  import CalendarDisabledOverlay from '../components/CalendarDisabledOverlay.vue'
  import BookingForm from '../components/BookingForm.vue'
  import BookingResults from '../components/BookingResults.vue'
  import type { EventClickArg } from '@fullcalendar/core'

  const {
    calendarRef,
    currentView,
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
    preferredTeacherId,
    durationMinutes,
    suggestions,
    showDetailedResults,
    manualSlots,
    newSlotDay,
    newSlotStart,
    newSlotEnd,
    currentStep,
    canSelectTeacher,
    canAddManualSlot,
    canEvaluate,
    suggestionEvents,
    initWatchers,
    handleAddManualSlot,
    removeManualSlot,
    addToCartDirectly,
    resetBookingState,
    handleEvaluate,
  } = useBooking()

  onMounted(() => {
    initWatchers()
  })

  const handleSuggestionClick = (info: EventClickArg): void => {
    const props = info.event.extendedProps
    if (props?.isSuggestion) {
      addToCartDirectly(props.teacherId, props.teacherName, info.event.startStr, info.event.endStr)
    }
  }
</script>

<template>
  <PageLayout title="Book a Session">
    <div class="flex h-full gap-6">
      <div class="w-1/2 flex flex-col">
        <div class="flex items-center justify-between mb-4">
          <button
            v-if="currentView === 'options'"
            type="button"
            class="text-sm text-(--accent-terracotta) hover:underline"
            @click="currentView = 'details'"
          >
            ← Back to Details
          </button>
          <h2 class="text-lg font-semibold text-(--ink-primary)">
            {{ currentView === 'details' ? 'Your Availability' : 'Booking Options' }}
          </h2>
          <div v-if="currentView === 'details'" class="flex items-center gap-1">
            <div
              v-for="i in 3"
              :key="i"
              class="w-8 h-1 rounded-full transition-colors duration-300"
              :class="{
                'bg-(--accent-terracotta)': currentStep >= i,
                'bg-gray-200': currentStep < i,
              }"
            ></div>
          </div>
        </div>

        <div class="flex-1 relative">
          <Calendar
            v-if="selectedTeacherId"
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
          <CalendarDisabledOverlay v-else message="Select a teacher to view their availability" />
        </div>
      </div>

      <div class="w-1/2">
        <div v-if="currentView === 'details'" class="flex flex-col h-full">
          <BookingForm
            :subjects="subjects"
            :branches="branches"
            :filtered-teachers="filteredTeachers"
            :selected-subject-id="selectedSubjectId"
            :selected-branch-id="selectedBranchId"
            :selected-teacher-id="selectedTeacherId"
            :preferred-teacher-id="preferredTeacherId"
            :duration-minutes="durationMinutes"
            :manual-slots="manualSlots"
            :new-slot-day="newSlotDay"
            :new-slot-start="newSlotStart"
            :new-slot-end="newSlotEnd"
            :current-step="currentStep"
            :can-select-teacher="canSelectTeacher"
            :can-add-manual-slot="canAddManualSlot"
            @update:selected-subject-id="selectedSubjectId = $event"
            @update:selected-branch-id="selectedBranchId = $event"
            @update:selected-teacher-id="selectedTeacherId = $event"
            @update:preferred-teacher-id="preferredTeacherId = $event"
            @update:duration-minutes="durationMinutes = $event"
            @update:new-slot-day="newSlotDay = $event"
            @update:new-slot-start="newSlotStart = $event"
            @update:new-slot-end="newSlotEnd = $event"
            @add-manual-slot="handleAddManualSlot"
            @remove-manual-slot="removeManualSlot"
          />
          <div class="flex gap-3 mt-4">
            <button
              type="button"
              class="flex-1 px-4 py-2 text-white bg-(--accent-terracotta) rounded-lg hover:bg-(--accent-terracotta-dark) transition-all font-medium"
              :disabled="!canEvaluate"
              @click="handleEvaluate"
            >
              {{ isEvaluating ? 'Evaluating...' : 'Evaluate Options' }}
            </button>
          </div>
        </div>

        <div v-else class="flex flex-col h-full">
          <div class="flex items-center justify-between mb-4">
            <h2 class="text-lg font-semibold text-(--ink-primary)">Booking Options</h2>
          </div>
          <BookingResults
            :suggestions="suggestions"
            :show-detailed-results="showDetailedResults"
            :is-evaluating="isEvaluating"
            @confirm-booking="addToCartDirectly"
            @go-back="currentView = 'details'"
            @reset="resetBookingState"
          />
        </div>
      </div>
    </div>

    <div
      v-if="successMessage"
      class="fixed bottom-4 right-4 bg-green-600 text-white px-4 py-3 rounded-lg shadow-lg flex items-center gap-2 animate-slide-up"
    >
      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="2"
          d="M5 13l4 4L19 7"
        ></path>
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
        ></path>
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
