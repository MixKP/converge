<script setup lang="ts">
  import { ref, computed, onMounted, watch } from 'vue'
  import { useTeacherStore } from '../stores/teacherStore'
  import { availabilityApi } from '../services/availabilityApi'
  import { subjectApi } from '../services/subjectApi'
  import { teacherApi } from '../services/teacherApi'
  import { bookingApi } from '../services/bookingApi'
  import { branchApi } from '../services/branchApi'
  import type { Teacher, Branch } from '../types'
  import PageLayout from '../components/PageLayout.vue'
  import Calendar from '../components/Calendar.vue'
  import CalendarDisabledOverlay from '../components/CalendarDisabledOverlay.vue'
  import ClientNameDialog from '../components/ClientNameDialog.vue'
  import type { EventInput, BusinessHoursInput } from '@fullcalendar/core'
  import type { WeeklySlot, Subject, BookingResponse, ConfirmBookingRequest } from '../types'
  import { generateAvailabilityPayload } from '../utils/calendarHelpers'
  import { transformBackendAvailability } from '../utils/availabilityTransform'

  const teacherStore = useTeacherStore()
  const calendarRef = ref<InstanceType<typeof Calendar>>()
  const isLoading = ref(false)
  const isLoadingAvailability = ref(false)
  const isLoadingTeachers = ref(false)
  const isEvaluating = ref(false)
  const isConfirming = ref(false)
  const errorMessage = ref<string>('')
  const successMessage = ref<string>('')
  const events = ref<EventInput[]>([])
  const businessHours = ref<BusinessHoursInput>([])
  const availabilityCache = ref<Map<number, WeeklySlot[]>>(new Map())
  const subjects = ref<Subject[]>([])
  const branches = ref<Branch[]>([])
  const filteredTeachers = ref<Teacher[]>([])
  const selectedSubjectId = ref<number | null>(null)
  const selectedBranchId = ref<number | null>(null)
  const suggestions = ref<BookingResponse | null>(null)
  const showDetailedResults = ref(false)
  const currentView = ref<'details' | 'options'>('details')

  const manualSlots = ref<WeeklySlot[]>([])
  const newSlotDay = ref<number>(0)
  const newSlotStart = ref<string>('09:00')
  const newSlotEnd = ref<string>('10:00')

  // Dialog state
  const isDialogOpen = ref(false)
  const dialogTeacherName = ref('')
  const dialogClientName = ref('')
  const pendingBooking = ref<{
    teacherId: number
    teacherName: string
    startTime: string
    endTime: string
  } | null>(null)

  const dayNames = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']

  const timeOptions = computed(() => {
    const options: string[] = []
    for (let hour = 8; hour <= 18; hour++) {
      options.push(`${String(hour).padStart(2, '0')}:00`)
      options.push(`${String(hour).padStart(2, '0')}:30`)
    }
    options.push('19:00')
    return options
  })

  const fetchAvailability = async () => {
    try {
      isLoadingAvailability.value = true
      const data = await availabilityApi.getAll()
      availabilityCache.value = transformBackendAvailability(data)
    } finally {
      isLoadingAvailability.value = false
    }
  }

  onMounted(async () => {
    const [subjectsData, branchesData] = await Promise.all([
      subjectApi.getAll(),
      branchApi.getAll(),
    ])
    subjects.value = subjectsData
    branches.value = branchesData
    fetchAvailability()
  })

  const fetchTeachersBySubject = async (subjectId: number) => {
    try {
      isLoadingTeachers.value = true
      filteredTeachers.value = await teacherApi.getBySubject(subjectId)
    } finally {
      isLoadingTeachers.value = false
    }
  }

  const addManualSlot = () => {
    if (!canAddManualSlot.value) return
    const slot: WeeklySlot = {
      day_of_week: newSlotDay.value,
      start: newSlotStart.value,
      end: newSlotEnd.value,
    }
    manualSlots.value.push(slot)
    suggestions.value = null
    showDetailedResults.value = false
    newSlotDay.value = (newSlotDay.value + 1) % 7
  }

  const removeManualSlot = (index: number) => {
    manualSlots.value.splice(index, 1)
    suggestions.value = null
    showDetailedResults.value = false
  }

  const getSlotLabel = (slot: WeeklySlot) => {
    return `${dayNames[slot.day_of_week]} ${slot.start} - ${slot.end}`
  }

  const selectedTeacherId = computed({
    get: () => teacherStore.selectedTeacherId,
    set: (val) => teacherStore.setSelectedTeacherById(val),
  })

  const preferredTeacherId = ref<number | null>(null)
  const durationMinutes = ref<number>(60)

  const calculatedDuration = computed(() => {
    if (events.value.length === 0) return 60
    const event = events.value[0]
    if (!event.start || !event.end) return 60
    const start = new Date(event.start as Date)
    const end = new Date(event.end as Date)
    return Math.round((end.getTime() - start.getTime()) / (1000 * 60))
  })

  const canSelectTeacher = computed(() => !!selectedSubjectId.value)

  const canEvaluate = computed(() => {
    const slots =
      manualSlots.value.length > 0
        ? manualSlots.value
        : events.value.map((e) => ({
            day_of_week: new Date(e.start as Date).getDay(),
            start: new Date(e.start as Date).toTimeString().slice(0, 5),
            end: new Date(e.end as Date).toTimeString().slice(0, 5),
          }))
    return (
      !!selectedSubjectId.value &&
      !!selectedBranchId.value &&
      slots.length > 0 &&
      !isEvaluating.value
    )
  })

  const canAddManualSlot = computed(() => {
    return newSlotStart.value < newSlotEnd.value
  })

  const currentStep = computed(() => {
    if (!selectedSubjectId.value) return 1
    if (!selectedBranchId.value) return 2
    return 3
  })

  const suggestionEvents = computed<EventInput[]>(() => {
    if (!suggestions.value || !showDetailedResults.value) return []
    const result: EventInput[] = []
    let suggestionIndex = 0
    for (const slotResult of suggestions.value.results) {
      if (slotResult.exact_match) {
        result.push({
          id: `suggestion-exact-${suggestionIndex++}`,
          title: `⭐ ${slotResult.exact_match.teacher_name} (Score: ${slotResult.exact_match.score})`,
          start: slotResult.exact_match.start_time,
          end: slotResult.exact_match.end_time,
          backgroundColor: 'var(--accent-terracotta)',
          borderColor: 'var(--accent-terracotta-dark)',
          textColor: '#fff',
          extendedProps: {
            isSuggestion: true,
            teacherId: slotResult.exact_match.teacher_id,
            teacherName: slotResult.exact_match.teacher_name,
            score: slotResult.exact_match.score,
          },
        })
      }
      if (slotResult.alternatives) {
        for (const alt of slotResult.alternatives) {
          result.push({
            id: `suggestion-alt-${suggestionIndex++}`,
            title: `💡 ${alt.teacher_name} (Score: ${alt.score})`,
            start: alt.start_time,
            end: alt.end_time,
            backgroundColor: 'var(--accent-terracotta-soft)',
            borderColor: 'var(--accent-terracotta)',
            textColor: 'var(--ink-primary)',
            extendedProps: {
              isSuggestion: true,
              teacherId: alt.teacher_id,
              teacherName: alt.teacher_name,
              score: alt.score,
            },
          })
        }
      }
    }
    return result
  })

  const allEvents = computed<EventInput[]>(() => {
    return [...events.value, ...suggestionEvents.value]
  })

  const updateBusinessHours = (teacherId: number) => {
    const availability = availabilityCache.value.get(teacherId)
    if (!availability) {
      businessHours.value = []
      return
    }
    businessHours.value = availability.map((avail) => ({
      daysOfWeek: [avail.day_of_week + 1],
      startTime: avail.start,
      endTime: avail.end,
    }))
    calendarRef.value?.setOption('businessHours', businessHours.value)
  }

  const handleEvaluate = async () => {
    if (!canEvaluate.value) return
    isEvaluating.value = true
    showDetailedResults.value = false
    suggestions.value = null
    errorMessage.value = ''
    try {
      const slots =
        manualSlots.value.length > 0
          ? manualSlots.value
          : events.value.map((e) => ({
              day_of_week: new Date(e.start as Date).getDay(),
              start: new Date(e.start as Date).toTimeString().slice(0, 5),
              end: new Date(e.end as Date).toTimeString().slice(0, 5),
            }))
      const request = {
        subject_id: selectedSubjectId.value!,
        branch_id: selectedBranchId.value!,
        preferred_slots: slots,
        duration_minutes: durationMinutes.value,
        ...(preferredTeacherId.value && { preferred_teacher_id: preferredTeacherId.value }),
      }
      suggestions.value = await bookingApi.evaluate(request)
      showDetailedResults.value = true
      currentView.value = 'options'
    } catch (error) {
      errorMessage.value = error instanceof Error ? error.message : 'Failed to evaluate booking'
    } finally {
      isEvaluating.value = false
    }
  }

  const backToDetails = () => {
    currentView.value = 'details'
  }

  const handleSuggestionClick = (info: any) => {
    const props = info.event.extendedProps
    if (!props.isSuggestion) return
    pendingBooking.value = {
      teacherId: props.teacherId,
      teacherName: props.teacherName,
      startTime: info.event.startStr,
      endTime: info.event.endStr,
    }
    dialogTeacherName.value = props.teacherName
    isDialogOpen.value = true
  }

  const handleConfirmBooking = async (
    teacherId: number,
    teacherName: string,
    startTime: string,
    endTime: string
  ) => {
    pendingBooking.value = {
      teacherId,
      teacherName,
      startTime,
      endTime,
    }
    dialogTeacherName.value = teacherName
    isDialogOpen.value = true
  }

  const processBooking = async () => {
    if (!pendingBooking.value || !dialogClientName.value.trim()) return

    isConfirming.value = true
    errorMessage.value = ''
    successMessage.value = ''
    try {
      const request: ConfirmBookingRequest = {
        teacher_id: pendingBooking.value.teacherId,
        branch_id: selectedBranchId.value!,
        subject_id: selectedSubjectId.value!,
        start_time: pendingBooking.value.startTime,
        end_time: pendingBooking.value.endTime,
        client_name: dialogClientName.value.trim(),
      }
      await bookingApi.confirm(request)
      successMessage.value = 'Booking confirmed successfully!'
      setTimeout(() => (successMessage.value = ''), 3000)
      isDialogOpen.value = false
      showDetailedResults.value = false
      suggestions.value = null
      events.value = []
      manualSlots.value = []
      currentView.value = 'details'
    } catch (error) {
      errorMessage.value = error instanceof Error ? error.message : 'Failed to confirm booking'
    } finally {
      isConfirming.value = false
      pendingBooking.value = null
      dialogClientName.value = ''
    }
  }

  const cancelDialog = () => {
    isDialogOpen.value = false
    dialogClientName.value = ''
    pendingBooking.value = null
  }

  const handleSubmit = async () => {
    if (!selectedTeacherId.value || isLoading.value) return
    isLoading.value = true
    errorMessage.value = ''
    try {
      const payload = generateAvailabilityPayload(events.value, selectedTeacherId.value)
      await availabilityApi.submitAvailability(payload)
      successMessage.value = 'Availability submitted successfully!'
      setTimeout(() => (successMessage.value = ''), 3000)
      events.value = []
      teacherStore.setSelectedTeacherById(null)
    } catch (error) {
      errorMessage.value = error instanceof Error ? error.message : 'Failed to submit availability'
    } finally {
      isLoading.value = false
    }
  }

  watch(selectedSubjectId, (newSubjectId) => {
    events.value = []
    teacherStore.setSelectedTeacherById(null)
    if (newSubjectId) {
      fetchTeachersBySubject(newSubjectId)
    } else {
      filteredTeachers.value = []
    }
  })

  watch(selectedTeacherId, (newId) => {
    events.value = []
    if (newId) {
      updateBusinessHours(newId)
    } else {
      businessHours.value = []
    }
  })

  watch(
    () => events.value.length,
    () => {
      if (events.value.length > 0) {
        durationMinutes.value = calculatedDuration.value
      }
    }
  )
</script>

<template>
  <PageLayout title="Course Booking">
    <form class="flex flex-col lg:flex-row flex-1 min-h-0 gap-6" @submit.prevent="handleSubmit">
      <aside class="w-full lg:w-80 shrink-0 flex flex-col">
        <div class="flex items-center justify-between mb-6">
          <div class="flex items-center gap-2">
            <button
              v-if="currentView === 'options'"
              @click="backToDetails"
              class="p-1 hover:bg-gray-100 rounded-lg transition-colors"
              type="button"
            >
              <svg
                class="w-5 h-5 text-(--ink-primary)"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M15 19l-7-7 7-7"
                ></path>
              </svg>
            </button>
            <h2 class="text-lg font-semibold text-(--ink-primary)">
              {{ currentView === 'details' ? 'Booking Details' : 'Availability Options' }}
            </h2>
          </div>
          <div v-if="currentView === 'details'" class="flex items-center gap-1">
            <div
              v-for="i in 3"
              :key="i"
              class="w-8 h-1 rounded-full transition-colors duration-300"
              :class="currentStep >= i ? 'bg-(--accent-terracotta)' : 'bg-gray-200'"
            ></div>
          </div>
        </div>

        <div v-if="currentView === 'details'" class="space-y-5 flex-1 overflow-y-auto">
          <div class="space-y-3">
            <label class="text-sm font-semibold text-(--ink-primary) flex items-center gap-2">
              <span
                class="w-5 h-5 rounded-full bg-(--accent-terracotta) text-white text-xs flex items-center justify-center"
                >1</span
              >
              Subject
            </label>
            <select
              v-model="selectedSubjectId"
              class="w-full bg-white border border-(--border-strong) rounded-lg px-4 py-3 text-sm focus:outline-none focus:ring-2 focus:ring-(--accent-terracotta)/20 focus:border-(--accent-terracotta) transition-all"
            >
              <option :value="null">Choose your subject</option>
              <option v-for="subject in subjects" :key="subject.id" :value="subject.id">
                {{ subject.name }}
              </option>
            </select>
            <p class="text-xs text-(--text-secondary)">Select the subject you want to book</p>
          </div>

          <div class="space-y-3">
            <label class="text-sm font-semibold text-(--ink-primary) flex items-center gap-2">
              <span
                class="w-5 h-5 rounded-full border-2 border-(--border-strong) text-(--text-secondary) text-xs flex items-center justify-center"
                :class="
                  currentStep >= 2
                    ? 'bg-(--accent-terracotta) border-(--accent-terracotta) text-white'
                    : ''
                "
                >2</span
              >
              Branch
            </label>
            <select
              v-model="selectedBranchId"
              :disabled="!selectedSubjectId"
              class="w-full bg-white border border-(--border-strong) rounded-lg px-4 py-3 text-sm focus:outline-none focus:ring-2 focus:ring-(--accent-terracotta)/20 focus:border-(--accent-terracotta) transition-all disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <option :value="null">Choose your branch</option>
              <option v-for="branch in branches" :key="branch.id" :value="branch.id">
                {{ branch.name }}
              </option>
            </select>
            <p v-if="!selectedSubjectId" class="text-xs text-(--text-secondary)">
              Select a subject first
            </p>
          </div>

          <div class="space-y-3">
            <label class="text-sm font-semibold text-(--ink-primary) flex items-center gap-2">
              <span
                class="w-5 h-5 rounded-full border-2 border-(--border-strong) text-(--text-secondary) text-xs flex items-center justify-center"
                :class="
                  currentStep >= 3
                    ? 'bg-(--accent-terracotta) border-(--accent-terracotta) text-white'
                    : ''
                "
                >3</span
              >
              Preferred Time Slots
            </label>
            <div v-if="manualSlots.length > 0" class="flex flex-wrap gap-2 mb-2">
              <span
                v-for="(slot, index) in manualSlots"
                :key="index"
                class="inline-flex items-center gap-1 bg-(--accent-terracotta-soft) text-(--ink-primary) text-xs px-2 py-1 rounded-full"
              >
                {{ getSlotLabel(slot) }}
                <button
                  @click="removeManualSlot(index)"
                  class="text-(--accent-terracotta) hover:text-red-600"
                >
                  ×
                </button>
              </span>
            </div>
            <div class="flex gap-2">
              <select
                v-model="newSlotDay"
                class="flex-1 bg-white border border-(--border-strong) rounded-lg px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-(--accent-terracotta)/20 focus:border-(--accent-terracotta) disabled:opacity-50"
                :disabled="!selectedBranchId"
              >
                <option v-for="(day, index) in dayNames" :key="index" :value="index">
                  {{ day }}
                </option>
              </select>
              <select
                v-model="newSlotStart"
                class="flex-1 bg-white border border-(--border-strong) rounded-lg px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-(--accent-terracotta)/20 focus:border-(--accent-terracotta) disabled:opacity-50"
                :disabled="!selectedBranchId"
              >
                <option v-for="time in timeOptions" :key="time" :value="time">
                  {{ time }}
                </option>
              </select>
              <select
                v-model="newSlotEnd"
                class="flex-1 bg-white border border-(--border-strong) rounded-lg px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-(--accent-terracotta)/20 focus:border-(--accent-terracotta) disabled:opacity-50"
                :disabled="!selectedBranchId"
              >
                <option v-for="time in timeOptions" :key="time" :value="time">
                  {{ time }}
                </option>
              </select>
              <button
                @click="addManualSlot"
                :disabled="!canAddManualSlot || !selectedBranchId"
                class="bg-(--accent-terracotta) text-white px-3 py-2.5 rounded-lg text-sm hover:bg-(--accent-terracotta-dark) disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                +
              </button>
            </div>
            <p v-if="!selectedBranchId" class="text-xs text-(--text-secondary)">
              Select a branch first
            </p>
            <p v-else class="text-xs text-(--text-secondary)">Add the times you want to study</p>
          </div>

          <div class="space-y-3">
            <label class="text-sm font-semibold text-(--ink-primary) flex items-center gap-2">
              <span
                class="w-5 h-5 rounded-full border-2 border-dashed border-(--border-subtle) text-(--text-secondary) text-xs flex items-center justify-center"
                >?</span
              >
              Preferred Teacher
              <span class="text-xs text-(--text-secondary) font-normal">(optional)</span>
            </label>
            <select
              v-model="selectedTeacherId"
              :disabled="!canSelectTeacher || isLoadingTeachers"
              class="w-full bg-white border border-(--border-strong) rounded-lg px-4 py-3 text-sm focus:outline-none focus:ring-2 focus:ring-(--accent-terracotta)/20 focus:border-(--accent-terracotta) transition-all disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <option :value="null">
                {{ isLoadingTeachers ? 'Loading...' : 'Choose your teacher' }}
              </option>
              <option v-for="teacher in filteredTeachers" :key="teacher.id" :value="teacher.id">
                {{ teacher.name }}
              </option>
            </select>
            <p
              v-if="filteredTeachers.length === 0 && selectedSubjectId"
              class="text-xs text-(--text-secondary)"
            >
              No teachers available for this subject
            </p>
            <p v-if="!selectedSubjectId" class="text-xs text-(--text-secondary)">
              Select a subject first
            </p>
            <p v-else-if="filteredTeachers.length === 0" class="text-xs text-(--text-secondary)">
              No teachers available for this subject
            </p>
            <p v-else class="text-xs text-(--text-secondary)">
              Leave empty to let CLP find the best teacher
            </p>
          </div>

          <div class="space-y-3 pt-4 border-t border-(--border-subtle)">
            <button
              @click="handleEvaluate"
              :disabled="!canEvaluate || isEvaluating"
              class="w-full py-2.5 rounded-lg text-sm font-medium bg-(--accent-terracotta) text-white hover:bg-(--accent-terracotta-dark) transition-colors disabled:opacity-50 disabled:cursor-not-allowed shadow-sm"
            >
              {{ isEvaluating ? 'Searching...' : 'Find Available Slots' }}
            </button>
          </div>
        </div>

        <div v-else class="space-y-5 flex-1 overflow-y-auto">
          <div v-if="!suggestions || suggestions.results.length === 0" class="text-center py-8">
            <p class="text-(--text-secondary)">No availability options found</p>
          </div>
          <div v-else class="space-y-3">
            <div
              v-for="(result, index) in suggestions.results"
              :key="index"
              class="bg-gray-50 rounded-lg p-3"
            >
              <p class="text-xs font-medium text-(--text-secondary) mb-2">
                {{ getSlotLabel(result.slot) }}
              </p>
              <div
                v-if="result.exact_match"
                class="bg-green-50 rounded-lg p-3 mb-2 border border-green-200"
              >
                <div class="flex items-center justify-between">
                  <div>
                    <p class="text-sm font-semibold text-green-800">
                      {{ result.exact_match.teacher_name }}
                    </p>
                    <p class="text-xs text-green-600">Score: {{ result.exact_match.score }}</p>
                  </div>
                  <button
                    @click="
                      handleConfirmBooking(
                        result.exact_match!.teacher_id,
                        result.exact_match!.teacher_name,
                        result.exact_match!.start_time,
                        result.exact_match!.end_time
                      )
                    "
                    :disabled="isConfirming"
                    class="px-3 py-1.5 rounded-lg text-xs font-medium bg-green-600 text-white hover:bg-green-700 disabled:opacity-50 transition-colors"
                  >
                    Book
                  </button>
                </div>
              </div>
              <div v-if="result.alternatives?.length" class="space-y-2">
                <div
                  v-for="(alt, altIndex) in result.alternatives"
                  :key="altIndex"
                  class="bg-white rounded-lg p-2.5 border border-(--border-subtle) flex items-center justify-between"
                >
                  <div>
                    <p class="text-sm font-medium text-(--ink-primary)">{{ alt.teacher_name }}</p>
                    <p class="text-xs text-(--text-secondary)">Score: {{ alt.score }}</p>
                  </div>
                  <button
                    @click="
                      handleConfirmBooking(
                        alt.teacher_id,
                        alt.teacher_name,
                        alt.start_time,
                        alt.end_time
                      )
                    "
                    :disabled="isConfirming"
                    class="px-3 py-1.5 rounded-lg text-xs font-medium bg-(--accent-terracotta) text-white hover:bg-(--accent-terracotta-dark) disabled:opacity-50 transition-colors"
                  >
                    Select
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </aside>

      <main
        class="flex-1 flex flex-col min-h-0 bg-white rounded-xl border border-(--border-subtle) shadow-sm overflow-hidden"
      >
        <div class="p-4 border-b border-(--border-subtle) flex items-center justify-between">
          <h3 class="text-sm font-semibold text-(--ink-primary)">Calendar</h3>
          <div
            v-if="showDetailedResults"
            class="flex items-center gap-2 text-xs text-(--text-secondary)"
          >
            <span class="w-2 h-2 rounded-full bg-green-500"></span>
            {{ suggestions?.results.length || 0 }} slot(s) found
          </div>
        </div>
        <div class="relative flex-1">
          <Calendar
            ref="calendarRef"
            v-model="events"
            :editable="false"
            :business-hours="businessHours"
            :additional-events="allEvents"
            @event-click="handleSuggestionClick"
            class="h-full"
          />
          <CalendarDisabledOverlay
            v-if="currentView === 'details' && !showDetailedResults"
            :message="
              !selectedSubjectId
                ? 'Select a subject to begin'
                : !selectedBranchId
                  ? 'Select a branch'
                  : 'Add time slots and click Find Available Slots'
            "
          />
        </div>
      </main>

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
        <button @click="errorMessage = ''" class="ml-2 text-white/80 hover:text-white">×</button>
      </div>

      <ClientNameDialog
        v-model:client-name="dialogClientName"
        :is-open="isDialogOpen"
        :teacher-name="dialogTeacherName"
        :is-processing="isConfirming"
        @confirm="processBooking"
        @cancel="cancelDialog"
      />
    </form>
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
