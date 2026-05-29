<script setup lang="ts">
  import { ref, computed, watch } from 'vue'
  import type { EventInput, BusinessHoursInput } from '@fullcalendar/core'
  import { useTeacherStore } from '../stores/teacherStore'
  import { availabilityApi } from '../services/availabilityApi'
  import type { WeeklySlot } from '../types/calendar'

  const teacherStore = useTeacherStore()
  const isLoading = ref(false)
  const calendarApi = ref<any>(null)

  // Subject mock data
  interface Subject {
    id: number
    name: string
  }

  const subjects = ref<Subject[]>([
    { id: 1, name: 'Mathematics' },
    { id: 2, name: 'Physics' },
    { id: 3, name: 'Chemistry' },
    { id: 4, name: 'Biology' },
    { id: 5, name: 'English' },
    { id: 6, name: 'Thai Language' },
  ])
  const selectedSubjectId = ref<number | null>(null)

  const events = ref<EventInput[]>([])
  const businessHours = ref<any>({})

  const selectedTeacherId = computed({
    get: () => teacherStore.selectedTeacherId,
    set: (val) => teacherStore.setSelectedTeacherById(val),
  })

  const canSubmit = computed(
    () =>
      !!selectedSubjectId.value &&
      !!selectedTeacherId.value &&
      !isLoading.value &&
      events.value.length > 0
  )

  const canSelectTeacher = computed(() => !!selectedSubjectId.value)

  const MOCK_AVAILABILITY: WeeklySlot[] = [
    { dayOfWeek: 1, start: '09:00', end: '12:00' },
    { dayOfWeek: 1, start: '14:00', end: '17:00' },
    { dayOfWeek: 3, start: '10:00', end: '12:00' },
    { dayOfWeek: 5, start: '09:00', end: '12:00' },
    { dayOfWeek: 0, start: '10:00', end: '12:00' },
  ] as const

  const updateBusinessHours = (availability: WeeklySlot[]) => {
    businessHours.value = availability.map((avail) => ({
      daysOfWeek: [avail.dayOfWeek],
      startTime: avail.start,
      endTime: avail.end,
    }))
    calendarRef.value?.setOption('businessHours', businessHours.value)
  }

  const calendarOptions = computed(() => ({
    plugins: [timeGridPlugin, interactionPlugin],
    initialView: 'timeGridWeek',
    headerToolbar: false,
    height: '100%', // <-- เพิ่มตรงนี้ เพื่อให้ Calendar ไม่ดันปุ่มตกลงไปข้างล่าง
    weekends: true,
    allDaySlot: false,
    editable: !!selectedTeacherId.value,
    selectable: !!selectedTeacherId.value,
    selectMirror: true,
    dayMaxEvents: true,
    businessHours: businessHours.value,
    slotMinTime: '08:00',
    slotMaxTime: '19:00',
    slotDuration: '00:30:00',
    snapDuration: '00:15:00',
    eventConstraint: 'businessHours',
    selectConstraint: 'businessHours',
    events: events.value,
    select: handleDateSelect,
    eventClick: handleEventClick,
    didMount: (info: any) => {
      calendarApi.value = info.view.calendar
    },
  }))

  const generatePayload = (eventsList: EventInput[], teacherId: number) => {
    const weeklyMap = new Map<string, { day_of_week: number; start: string; end: string }>()
    for (const event of eventsList) {
      if (!event.start || !event.end) continue

      const start = new Date(event.start as Date)
      const jsDay = start.getDay()
      const dayOfWeek = jsDay === 0 ? 7 : jsDay
      const startTime = start.toTimeString().slice(0, 5)

      const end = new Date(event.end as Date)
      const endTime = end.toTimeString().slice(0, 5)

      const key = `${dayOfWeek}-${startTime}-${endTime}`
      if (!weeklyMap.has(key)) {
        weeklyMap.set(key, { day_of_week: dayOfWeek, start: startTime, end: endTime })
      }
    }
    return { teacher_id: teacherId, weekly: Array.from(weeklyMap.values()) }
  }

  const handleSubmit = async () => {
    if (!selectedTeacherId.value || isLoading.value) return

    isLoading.value = true
    try {
      const payload = generatePayload(events.value, selectedTeacherId.value)
      await availabilityApi.submitAvailability(payload)

      events.value = []
      teacherStore.setSelectedTeacherById(null)
    } finally {
      isLoading.value = false
    }
  }

  watch(selectedSubjectId, () => {
    events.value = []
    teacherStore.setSelectedTeacherById(null)
  })

  watch(selectedTeacherId, (newId) => {
    events.value = []
    if (newId) {
      updateBusinessHours(MOCK_AVAILABILITY)
    } else {
      businessHours.value = {}
    }
  })

  watch(
    businessHours,
    () => {
      if (calendarApi.value) {
        calendarApi.value.setOption('businessHours', businessHours.value)
      }
    },
    { deep: true }
  )
</script>

<template>
  <PageLayout title="Booking Course">
    <form class="flex flex-col gap-3 sm:gap-4 flex-1 min-h-0" @submit.prevent="handleSubmit">
      <div class="flex flex-col sm:flex-row gap-3 sm:gap-5 shrink-0 stagger-in">
        <div
          class="bg-(--paper-white) p-4 sm:p-5 rounded-sm border border-(--border-subtle) flex-1 shadow-card"
        >
          <label
            class="font-semibold text-(--ink-primary) text-sm block mb-2 sm:mb-3 tracking-tight"
          >
            <span
              class="font-mono text-xs border-2 border-(--ink-primary) bg-(--paper-cream) px-2.5 sm:px-3 py-1 rounded-sm tracking-widest mr-2 sm:mr-3 shadow-badge inline-block"
              >01</span
            >
            Select Subject
          </label>
          <select
            v-model="selectedSubjectId"
            class="w-full bg-white border border-(--border-strong) rounded-sm px-3 sm:px-4 py-2.5 sm:py-3 text-sm focus:ring-2 focus:ring-(--accent-terracotta)/10 focus:border-(--accent-terracotta) outline-none transition-all text-(--ink-primary) cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed hover:border-(--text-secondary)"
          >
            <option :value="null" disabled>-- Choose Subject --</option>
            <option v-for="subject in subjects" :key="subject.id" :value="subject.id">
              {{ subject.name }}
            </option>
          </select>
        </div>

        <div
          class="bg-(--paper-white) p-4 sm:p-5 rounded-sm border border-(--border-subtle) flex-1 shadow-card"
        >
          <label
            class="font-semibold text-(--ink-primary) text-sm block mb-2 sm:mb-3 tracking-tight"
          >
            <span
              class="font-mono text-xs border-2 border-(--ink-primary) bg-(--paper-cream) px-2.5 sm:px-3 py-1 rounded-sm tracking-widest mr-2 sm:mr-3 shadow-badge inline-block"
              >02</span
            >
            Select Teacher
          </label>
          <select
            v-model="selectedTeacherId"
            :disabled="!canSelectTeacher"
            class="w-full bg-white border border-(--border-strong) rounded-sm px-3 sm:px-4 py-2.5 sm:py-3 text-sm focus:ring-2 focus:ring-(--accent-terracotta)/10 focus:border-(--accent-terracotta) outline-none transition-all text-(--ink-primary) cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed hover:border-(--text-secondary)"
          >
            <option :value="null" disabled>
              {{ canSelectTeacher ? '-- Choose Teacher --' : '-- Select subject first --' }}
            </option>
            <option v-for="teacher in teacherStore.teachers" :key="teacher.id" :value="teacher.id">
              {{ teacher.name }}
            </option>
          </select>
        </div>
      </div>

      <div class="flex-1 flex flex-col min-h-0 stagger-in">
        <div class="flex items-center justify-between mb-3 sm:mb-4 shrink-0">
          <label class="font-semibold text-(--ink-primary) text-sm tracking-tight">
            <span
              class="font-mono text-xs border-2 border-(--ink-primary) bg-(--paper-cream) px-2.5 sm:px-3 py-1 rounded-sm tracking-widest mr-2 sm:mr-3 shadow-badge inline-block"
              >03</span
            >
            Select Time Slots
          </label>
          <span
            v-if="selectedTeacherId"
            class="text-xs font-medium text-(--accent-terracotta) bg-(--accent-terracotta-soft) px-3 sm:px-4 py-1 sm:py-1.5 rounded-sm border border-(--accent-terracotta)/25 tracking-wide hidden sm:inline-block"
          >
            Click and drag to select
          </span>
        </div>

        <div
          class="relative border border-(--border-subtle) rounded-sm overflow-hidden shadow-card bg-(--paper-white) flex-1 min-h-0"
        >
          <div
            :class="{
              'opacity-45 grayscale-30 pointer-events-none transition-opacity duration-300':
                !selectedTeacherId,
            }"
            class="h-full"
          >
            <Calendar
              ref="calendarRef"
              v-model="events"
              :editable="!!selectedTeacherId"
              :business-hours="businessHours"
              constraint="businessHours"
              class="h-full"
            />
          </div>

          <div class="bg-slate-50/50 p-3 rounded-lg border border-slate-100 flex-1">
            <label class="font-semibold text-slate-700 text-sm block mb-1">
              <span class="bg-blue-100 text-blue-700 px-2 py-0.5 rounded text-xs mr-1">2</span>
              Select Teacher
            </label>
            <select
              v-model="selectedTeacherId"
              :disabled="!canSelectTeacher"
              class="w-full bg-white border border-slate-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 outline-none transition-all text-slate-700 shadow-sm cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <option :value="null" disabled>
                {{ canSelectTeacher ? '-- Choose Teacher --' : '-- Select subject first --' }}
              </option>
              <option
                v-for="teacher in teacherStore.teachers"
                :key="teacher.id"
                :value="teacher.id"
              >
                {{ teacher.name }}
              </option>
            </select>
          </div>
        </div>

      <div
        class="flex justify-end pt-4 sm:pt-5 border-t border-(--border-subtle) shrink-0 stagger-in"
      >
        <SubmitButton
          :is-disabled="!canSubmit"
          :is-loading="isLoading"
          loading-text="Submitting..."
          normal-text="Confirm Booking"
        />
      </div>
    </form>
  </PageLayout>
</template>

<style>
  /* --- FullCalendar Modern Theme Overrides --- */

  .fc {
    font-family: inherit;
    --fc-border-color: #e2e8f0; /* Tailwind slate-200 */
    --fc-page-bg-color: #ffffff;
    --fc-today-bg-color: #f0f9ff; /* Tailwind sky-50 */
    --fc-highlight-color: rgba(59, 130, 246, 0.1);
  }

  /* Header styles */
  .fc-theme-standard th {
    padding: 12px 0;
    background-color: #f8fafc; /* Tailwind slate-50 */
    font-weight: 600;
    color: #475569; /* Tailwind slate-600 */
    text-transform: uppercase;
    font-size: 0.75rem;
    letter-spacing: 0.05em;
    border-bottom: 2px solid #e2e8f0;
  }

  /* Event Styles */
  .fc-event {
    cursor: pointer;
    background-color: #3b82f6;
    border: 1px solid #2563eb;
    border-radius: 6px;
    padding: 2px 4px;
    font-size: 0.75rem;
    box-shadow: 0 2px 4px rgba(59, 130, 246, 0.2);
    transition:
      transform 0.15s ease,
      box-shadow 0.15s ease;
  }

  .fc-event:hover {
    background-color: #2563eb;
    transform: translateY(-1px);
    box-shadow: 0 4px 6px rgba(59, 130, 246, 0.3);
    z-index: 5 !important;
  }

  .fc-v-event .fc-event-main {
    color: #ffffff;
    font-weight: 500;
  }

  /* Non-business hours (disabled areas) */
  .fc-non-business {
    background-color: #f1f5f9;
    background-image: repeating-linear-gradient(
      45deg,
      #f1f5f9,
      #f1f5f9 8px,
      #e2e8f0 8px,
      #e2e8f0 9px
    );
    opacity: 0.7;
  }

  /* Make slots slightly taller for better click/touch area */
  .fc-timegrid-slot {
    height: 34px;
  }

  /* Soften grid lines */
  .fc-theme-standard td,
  .fc-theme-standard th {
    border-color: #f1f5f9; /* Tailwind slate-100 */
  }

  /* Axis time text styling */
  .fc .fc-timegrid-axis-cushion,
  .fc .fc-timegrid-slot-label-cushion {
    color: #64748b; /* Tailwind slate-500 */
    font-size: 0.75rem;
    font-weight: 500;
  }

  /* Current time line */
  .fc-timegrid-now-indicator-line {
    border-color: #ef4444; /* Tailwind red-500 */
    border-width: 2px;
  }

  .fc-timegrid-now-indicator-arrow {
    border-color: #ef4444;
    border-width: 5px;
  }

  /* Custom animation for overlay */
  @keyframes bounce-slight {
    0%,
    100% {
      transform: translateY(0);
    }
    50% {
      transform: translateY(-5px);
    }
  }
  .animate-bounce-slight {
    animation: bounce-slight 3s ease-in-out infinite;
  }
</style>
