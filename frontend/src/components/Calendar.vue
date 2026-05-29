<script setup lang="ts">
  import { ref, computed, watch } from 'vue'
  import FullCalendar from '@fullcalendar/vue3'
  import timeGridPlugin from '@fullcalendar/timegrid'
  import interactionPlugin from '@fullcalendar/interaction'
  import type {
    EventInput,
    DateSelectArg,
    EventClickArg,
    BusinessHoursInput,
    CalendarOptions,
  } from '@fullcalendar/core'

  interface Props {
    editable?: boolean
    businessHours?: BusinessHoursInput
    constraint?: string | 'businessHours'
    modelValue?: EventInput[]
  }

  const props = withDefaults(defineProps<Props>(), {
    editable: false,
    businessHours: undefined,
    constraint: undefined,
    modelValue: () => [],
  })

  const emit = defineEmits<{
    'update:modelValue': [value: EventInput[]]
  }>()

  const calendarRef = ref<InstanceType<typeof FullCalendar> | null>(null)

  const showDeleteDialog = ref(false)
  const eventToDelete = ref<string | null>(null)

  const generateEventId = (): string => {
    return `event-${Date.now()}-${Math.random().toString(36).slice(2, 11)}`
  }

  const handleDateSelect = (selectInfo: DateSelectArg) => {
    if (!props.editable) return

    const calendar = selectInfo.view.calendar
    const start = selectInfo.start
    const end = selectInfo.end

    calendar.unselect()

    const newEvent = { id: generateEventId(), start, end, title: '' }
    const newEvents = [...(props.modelValue || []), newEvent]
    emit('update:modelValue', newEvents)
  }

  const handleEventClick = (clickInfo: EventClickArg) => {
    const id = clickInfo.event.id
    if (!id) return

    eventToDelete.value = id
    showDeleteDialog.value = true
  }

  const confirmDelete = () => {
    if (eventToDelete.value) {
      const filteredEvents = (props.modelValue || []).filter((e) => e.id !== eventToDelete.value)
      emit('update:modelValue', filteredEvents)
    }
    closeDeleteDialog()
  }

  const closeDeleteDialog = () => {
    showDeleteDialog.value = false
    eventToDelete.value = null
  }

  const CALENDAR_DEFAULT_OPTIONS = {
    initialView: 'timeGridWeek',
    headerToolbar: false,
    height: '100%',
    weekends: true,
    allDaySlot: false,
    selectMirror: true,
    dayMaxEvents: true,
    nowIndicator: true,
    slotMinTime: '08:00',
    slotMaxTime: '19:00',
    slotDuration: '00:30:00',
    snapDuration: '01:00:00',
  } as const

  watch(
    () => props.modelValue,
    (newEvents, oldEvents) => {
      const api = calendarRef.value?.getApi()
      if (!api) return

      const oldIds = new Set(
        (oldEvents ?? []).map((e) => e.id).filter((id): id is string => id != null)
      )
      const newIds = new Set(
        (newEvents ?? []).map((e) => e.id).filter((id): id is string => id != null)
      )

      // Diff-based sync: avoid full re-render which causes flicker
      for (const id of oldIds) {
        if (!newIds.has(id)) {
          api.getEventById(id)?.remove()
        }
      }

      for (const event of newEvents ?? []) {
        if (!event.id) continue

        const existing = api.getEventById(event.id)
        if (existing) {
          existing.setDates(event.start as Date, event.end as Date)
        } else {
          api.addEvent(event)
        }
      }
    }
  )

  const calendarOptions = computed(() => ({
    plugins: [timeGridPlugin, interactionPlugin],
    ...CALENDAR_DEFAULT_OPTIONS,
    editable: props.editable,
    selectable: props.editable,
    businessHours: props.businessHours,
    eventConstraint: props.constraint,
    selectConstraint: props.constraint,
    select: handleDateSelect,
    eventClick: handleEventClick,
  }))

  const setOption = <K extends keyof CalendarOptions>(key: K, value: CalendarOptions[K]) => {
    const api = calendarRef.value?.getApi()
    if (api) {
      api.setOption(key, value)
    }
  }

  defineExpose({ setOption })
</script>

<template>
  <div class="border border-slate-200 rounded-xl overflow-hidden h-full bg-white shadow-sm">
    <FullCalendar ref="calendarRef" :options="calendarOptions" />

    <Transition
      enter-active-class="transition-opacity duration-200"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition-opacity duration-150"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="showDeleteDialog"
        class="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/50 backdrop-blur-sm"
      >
        <div
          class="transition-all duration-300 ease-out scale-in bg-white rounded-2xl shadow-2xl border border-slate-200 p-6 w-full max-w-sm mx-4"
        >
          <div class="flex items-start gap-4 mb-4">
            <div
              class="shrink-0 w-12 h-12 rounded-full bg-red-100 flex items-center justify-center"
            >
              <svg
                class="w-6 h-6 text-red-600"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                />
              </svg>
            </div>
            <div class="flex-1">
              <h3 class="text-lg font-semibold text-slate-900 mb-1">Delete Time Slot</h3>
              <p class="text-sm text-slate-600">
                Are you sure you want to delete this time slot? This action cannot be undone.
              </p>
            </div>
          </div>

          <div class="flex gap-3 justify-end">
            <button
              type="button"
              class="px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-lg hover:bg-slate-50 focus:ring-4 focus:ring-slate-100 transition-all"
              @click="closeDeleteDialog"
            >
              Cancel
            </button>
            <button
              type="button"
              class="px-4 py-2 text-sm font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 focus:ring-4 focus:ring-red-100 transition-all shadow-sm shadow-red-600/20"
              @click="confirmDelete"
            >
              Delete
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>
