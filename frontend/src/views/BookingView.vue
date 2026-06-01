<script setup lang="ts">
  import { ref, onMounted, computed, watch } from 'vue'
  import { useBooking } from '../composables/useBooking'
  import PageLayout from '../components/PageLayout.vue'
  import Calendar from '../components/Calendar.vue'
  import CalendarDisabledOverlay from '../components/CalendarDisabledOverlay.vue'
  import BookingResults from '../components/BookingResults.vue'
  import type { EventClickArg } from '@fullcalendar/core'
  import type { Teacher } from '../types'

  // Force recompile
  const __RECOMPILE__ = Date.now()

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
    durationMinutes,
    suggestions,
    showDetailedResults,
    suggestionEvents,
    initWatchers,
    addToCartDirectly,
    resetBookingState,
    handleEvaluate,
    getTeachersBySubject,
  } = useBooking()

  // AI-specific selections (independent from manual tab)
  const aiSubjectId = ref<number | null>(null)
  const aiBranchId = ref<number | null>(null)
  const aiTeacherId = ref<number | null>(null)
  const aiFilteredTeachers = ref<Teacher[]>([])

  // Watch AI subject changes to fetch teachers
  watch(aiSubjectId, async (newSubjectId) => {
    if (newSubjectId) {
      aiFilteredTeachers.value = await getTeachersBySubject(newSubjectId)
    } else {
      aiFilteredTeachers.value = []
    }
    aiTeacherId.value = null
  })

  // AI time slots
  const aiTimeSlots = ref<Array<{ day_of_week: number; start: string; end: string }>>([])
  const aiNewSlotDay = ref<number>(1) // Monday default
  const aiNewSlotStart = ref<string>('09:00')
  const aiNewSlotEnd = ref<string>('10:00')

  const DAY_NAMES = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'] as const

  const TIME_OPTIONS: string[] = []
  for (let hour = 8; hour <= 18; hour++) {
    const hourStr = String(hour).padStart(2, '0')
    TIME_OPTIONS.push(`${hourStr}:00`, `${hourStr}:30`)
  }
  TIME_OPTIONS.push('19:00')

  const addAiTimeSlot = () => {
    if (aiNewSlotStart.value >= aiNewSlotEnd.value) return
    aiTimeSlots.value.push({
      day_of_week: aiNewSlotDay.value,
      start: aiNewSlotStart.value,
      end: aiNewSlotEnd.value,
    })
    // Reset to default values
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

  onMounted(() => {
    initWatchers()
  })

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

    // Use AI time slots directly
    try {
      isEvaluating.value = true
      errorMessage.value = ''
      successMessage.value = ''

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
      errorMessage.value = error instanceof Error ? error.message : 'Failed to generate booking options'
    } finally {
      isEvaluating.value = false
    }
  }
</script>

<template>
  <PageLayout title="Book a Session">
    <div class="flex h-full gap-6">
      <div class="w-1/2 flex flex-col">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-lg font-semibold text-(--ink-primary)">Availability</h2>
        </div>

        <div class="flex-1 relative">
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
          <CalendarDisabledOverlay
            v-else
            message="Select a subject to view availability"
          />
        </div>
      </div>

      <div class="w-1/2 flex flex-col">
        <div class="flex items-center gap-2 mb-4 border-b border-(--border-subtle)">
          <button
            type="button"
            class="px-4 py-2 font-medium transition-all relative"
            :class="activeTab === 'manual' ? 'text-(--accent-terracotta)' : 'text-(--text-secondary)'"
            @click="activeTab = 'manual'"
          >
            Manual
            <div
              v-if="activeTab === 'manual'"
              class="absolute bottom-0 left-0 right-0 h-0.5 bg-(--accent-terracotta)"
            ></div>
          </button>
          <button
            type="button"
            class="px-4 py-2 font-medium transition-all relative"
            :class="activeTab === 'ai' ? 'text-(--accent-terracotta)' : 'text-(--text-secondary)'"
            @click="activeTab = 'ai'"
          >
            AI Suggestion
            <div
              v-if="activeTab === 'ai'"
              class="absolute bottom-0 left-0 right-0 h-0.5 bg-(--accent-terracotta)"
            ></div>
          </button>
        </div>

        <div v-if="activeTab === 'manual'" class="flex flex-col gap-4">
          <div class="space-y-4">
            <div>
              <label class="block text-sm font-medium text-(--ink-primary) mb-1">Subject</label>
              <select
                v-model="selectedSubjectId"
                class="w-full px-3 py-2 border border-(--border-subtle) rounded-lg bg-white text-(--ink-primary) focus:outline-none focus:ring-2 focus:ring-(--accent-terracotta)"
              >
                <option :value="null">Select a subject</option>
                <option v-for="subject in subjects" :key="subject.id" :value="subject.id">
                  {{ subject.name }}
                </option>
              </select>
            </div>

            <div>
              <label class="block text-sm font-medium text-(--ink-primary) mb-1">Branch</label>
              <select
                v-model="selectedBranchId"
                :disabled="!selectedSubjectId"
                class="w-full px-3 py-2 border border-(--border-subtle) rounded-lg bg-white text-(--ink-primary) focus:outline-none focus:ring-2 focus:ring-(--accent-terracotta) disabled:bg-gray-100 disabled:cursor-not-allowed"
              >
                <option :value="null">Select a branch</option>
                <option v-for="branch in branches" :key="branch.id" :value="branch.id">
                  {{ branch.name }}
                </option>
              </select>
            </div>

            <div>
              <label class="block text-sm font-medium text-(--ink-primary) mb-1">Teacher</label>
              <select
                v-model="selectedTeacherId"
                :disabled="!selectedSubjectId"
                class="w-full px-3 py-2 border border-(--border-subtle) rounded-lg bg-white text-(--ink-primary) focus:outline-none focus:ring-2 focus:ring-(--accent-terracotta) disabled:bg-gray-100 disabled:cursor-not-allowed"
              >
                <option :value="null">Select a teacher</option>
                <option v-for="teacher in filteredTeachers" :key="teacher.id" :value="teacher.id">
                  {{ teacher.name }}
                </option>
              </select>
            </div>
          </div>

          <div v-if="events.length > 0" class="mt-4">
            <h3 class="text-sm font-medium text-(--ink-primary) mb-2">Selected Time Slot</h3>
            <div class="bg-(--paper-cream) p-3 rounded-lg border border-(--border-subtle)">
              <p class="text-sm text-(--text-secondary)">
                {{ events.length }} slot(s) selected from calendar
              </p>
            </div>
          </div>

          <button
            type="button"
            class="w-full px-4 py-3 text-white bg-(--accent-terracotta) rounded-lg hover:bg-(--accent-terracotta-dark) transition-all font-medium disabled:bg-gray-300 disabled:cursor-not-allowed"
            :disabled="!selectedSubjectId || !selectedBranchId || !selectedTeacherId || events.length === 0"
            @click="
              events.forEach((e) => {
                const teacher = filteredTeachers.find((t) => t.id === selectedTeacherId)
                if (teacher) {
                  addToCartDirectly(teacher.id, teacher.name, e.start as string, e.end as string)
                }
              })
            "
          >
            Add to Cart
          </button>
        </div>

        <div v-else class="flex flex-col h-full">
          <div v-if="!showDetailedResults && !isEvaluating" class="flex flex-col gap-4 overflow-y-auto pr-2 flex-1">
            <div class="space-y-4">
              <div>
                <label class="block text-sm font-medium text-(--ink-primary) mb-1">Subject</label>
                <select
                  v-model="aiSubjectId"
                  class="w-full px-3 py-2 border border-(--border-subtle) rounded-lg bg-white text-(--ink-primary) focus:outline-none focus:ring-2 focus:ring-(--accent-terracotta)"
                >
                  <option :value="null">Select a subject</option>
                  <option v-for="subject in subjects" :key="subject.id" :value="subject.id">
                    {{ subject.name }}
                  </option>
                </select>
              </div>

              <div class="p-4 bg-(--paper-cream) rounded-lg border border-(--border-subtle)">
                <h4 class="text-sm font-semibold text-(--ink-primary) mb-3">Preferred Time Slots</h4>

                <div class="space-y-3 mb-3">
                  <div>
                    <label class="block text-xs text-(--text-secondary) mb-1">Day</label>
                    <select
                      v-model="aiNewSlotDay"
                      class="w-full px-2 py-1.5 text-sm border border-(--border-subtle) rounded bg-white text-(--ink-primary) focus:outline-none focus:ring-1 focus:ring-(--accent-terracotta)"
                    >
                      <option :value="0">Sunday</option>
                      <option :value="1">Monday</option>
                      <option :value="2">Tuesday</option>
                      <option :value="3">Wednesday</option>
                      <option :value="4">Thursday</option>
                      <option :value="5">Friday</option>
                      <option :value="6">Saturday</option>
                    </select>
                  </div>
                  <div class="flex gap-2">
                    <div class="flex-1">
                      <label class="block text-xs text-(--text-secondary) mb-1">Start</label>
                      <select
                        v-model="aiNewSlotStart"
                        class="w-full px-2 py-1.5 text-sm border border-(--border-subtle) rounded bg-white text-(--ink-primary) focus:outline-none focus:ring-1 focus:ring-(--accent-terracotta)"
                      >
                        <option v-for="time in TIME_OPTIONS" :key="time" :value="time">
                          {{ time }}
                        </option>
                      </select>
                    </div>
                    <div class="flex-1">
                      <label class="block text-xs text-(--text-secondary) mb-1">End</label>
                      <select
                        v-model="aiNewSlotEnd"
                        class="w-full px-2 py-1.5 text-sm border border-(--border-subtle) rounded bg-white text-(--ink-primary) focus:outline-none focus:ring-1 focus:ring-(--accent-terracotta)"
                      >
                        <option v-for="time in TIME_OPTIONS" :key="time" :value="time">
                          {{ time }}
                        </option>
                      </select>
                    </div>
                  </div>
                  <button
                    type="button"
                    class="w-full px-3 py-1.5 text-sm bg-(--accent-terracotta) text-white rounded hover:bg-(--accent-terracotta-dark) transition-colors disabled:bg-gray-300 disabled:cursor-not-allowed"
                    :disabled="aiNewSlotStart >= aiNewSlotEnd"
                    @click="addAiTimeSlot"
                  >
                    Add Time Slot
                  </button>
                </div>

                <div v-if="aiTimeSlots.length > 0" class="space-y-2 max-h-48 overflow-y-auto pr-1">
                  <div
                    v-for="(slot, index) in aiTimeSlots"
                    :key="index"
                    class="flex items-center justify-between p-2 bg-white rounded border border-(--border-subtle)"
                  >
                    <span class="text-sm text-(--ink-primary)">
                      {{ getDayName(slot.day_of_week) }} {{ slot.start }} - {{ slot.end }}
                    </span>
                    <button
                      type="button"
                      class="text-red-500 hover:text-red-700 text-sm px-2 py-1"
                      @click="removeAiTimeSlot(index)"
                    >
                      Remove
                    </button>
                  </div>
                </div>
                <p v-else class="text-xs text-(--text-secondary)">No time slots added yet</p>
              </div>

              <div>
                <label class="block text-sm font-medium text-(--ink-primary) mb-1">Branch</label>
                <select
                  v-model="aiBranchId"
                  :disabled="!aiSubjectId"
                  class="w-full px-3 py-2 border border-(--border-subtle) rounded-lg bg-white text-(--ink-primary) focus:outline-none focus:ring-2 focus:ring-(--accent-terracotta) disabled:bg-gray-100 disabled:cursor-not-allowed"
                >
                  <option :value="null">Select a branch</option>
                  <option v-for="branch in branches" :key="branch.id" :value="branch.id">
                    {{ branch.name }}
                  </option>
                </select>
              </div>

              <div>
                <label class="block text-sm font-medium text-(--ink-primary) mb-1">Teacher (Optional)</label>
                <select
                  v-model="aiTeacherId"
                  :disabled="!aiSubjectId"
                  class="w-full px-3 py-2 border border-(--border-subtle) rounded-lg bg-white text-(--ink-primary) focus:outline-none focus:ring-2 focus:ring-(--accent-terracotta) disabled:bg-gray-100 disabled:cursor-not-allowed"
                >
                  <option :value="null">Any teacher</option>
                  <option v-for="teacher in aiFilteredTeachers" :key="teacher.id" :value="teacher.id">
                    {{ teacher.name }}
                  </option>
                </select>
              </div>
            </div>

            <button
              type="button"
              class="w-full px-4 py-3 text-white bg-(--accent-terracotta) rounded-lg hover:bg-(--accent-terracotta-dark) transition-all font-medium disabled:bg-gray-300 disabled:cursor-not-allowed"
              :disabled="!aiSubjectId || !aiBranchId || aiTimeSlots.length === 0"
              @click="handleGetAISuggestions"
            >
              Get AI Suggestions
            </button>
          </div>
          <BookingResults
            :suggestions="suggestions"
            :show-detailed-results="showDetailedResults"
            :is-evaluating="isEvaluating"
            @confirm-booking="addToCartDirectly"
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
