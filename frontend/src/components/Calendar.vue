<script setup lang="ts">
  import { ref, computed } from 'vue'
  import FullCalendar from '@fullcalendar/vue3'
  import timeGridPlugin from '@fullcalendar/timegrid'
  import interactionPlugin from '@fullcalendar/interaction'
  import { useCalendarResponsive } from '../composables/useCalendarResponsive'
  import { useCalendarEventSync } from '../composables/useCalendarEventSync'
  import { useBusinessHoursHeaders } from '../composables/useBusinessHoursHeaders'
  import { useCalendarInteraction } from '../composables/useCalendarInteraction'
  import type {
    EventInput,
    EventClickArg,
    BusinessHoursInput,
    CalendarOptions,
  } from '@fullcalendar/core'

  interface Props {
    editable?: boolean
    businessHours?: BusinessHoursInput
    constraint?: string
    modelValue?: EventInput[]
    additionalEvents?: EventInput[]
  }

  const props = withDefaults(defineProps<Props>(), {
    editable: false,
    businessHours: undefined,
    constraint: undefined,
    modelValue: () => [],
    additionalEvents: () => [],
  })
  const emit = defineEmits<{
    'update:modelValue': [value: EventInput[]]
    'event-click': [info: EventClickArg]
  }>()

  const calendarRef = ref<InstanceType<typeof FullCalendar> | null>(null)
  // Bound via the template; the interaction composable focuses it when the
  // delete dialog opens.
  const cancelBtnRef = ref<HTMLButtonElement | null>(null)

  // Screen-size dependent options (view, header format, long-press delays).
  const { isMobile, dayHeaderFormat, initialView, longPressDelay } =
    useCalendarResponsive(calendarRef)

  // Availability decoration on day headers.
  const { handleDayHeaderDidMount } = useBusinessHoursHeaders(
    calendarRef,
    () => props.businessHours
  )

  // All editable-event interaction: create, move/resize, select, delete, touch.
  const {
    selectedEventId,
    showDeleteDialog,
    handleContainerClick,
    handleEscapeKey,
    handleDeleteButtonClick,
    closeDeleteDialog,
    confirmDelete,
    handleDateSelect,
    handleSelectAllow,
    handleEventAllow,
    handleSlotClick,
    handleEventClick,
    handleEventDrop,
    handleEventResizeStart,
    handleEventResize,
    handleEventClassNames,
    handleEventDidMount,
    handleEventWillUnmount,
  } = useCalendarInteraction({
    calendarRef,
    isMobile,
    cancelBtnRef,
    isEditable: () => props.editable,
    getModelValue: () => props.modelValue ?? [],
    getAdditionalEvents: () => props.additionalEvents ?? [],
    onUpdate: (events) => emit('update:modelValue', events),
    onEventClick: (info) => emit('event-click', info),
  })

  // Keep the calendar in sync with the editable model events and the read-only
  // suggestion events (the latter are added as non-editable).
  useCalendarEventSync(calendarRef, () => props.modelValue)
  useCalendarEventSync(
    calendarRef,
    () => props.additionalEvents,
    (event) => ({
      ...event,
      editable: false,
    })
  )

  const formatTimeRange = (start: Date, end: Date): string => {
    const startHours = start.getHours()
    const endHours = end.getHours()
    const startAmpm = startHours >= 12 ? 'PM' : 'AM'
    const endAmpm = endHours >= 12 ? 'PM' : 'AM'

    const startFormatted = `${startHours % 12 || 12}:${start.getMinutes().toString().padStart(2, '0')}`
    const endFormatted = `${endHours % 12 || 12}:${end.getMinutes().toString().padStart(2, '0')}`

    if (startAmpm === endAmpm) {
      return `${startFormatted} - ${endFormatted} ${startAmpm}`
    }
    return `${startFormatted} ${startAmpm} - ${endFormatted} ${endAmpm}`
  }

  const handleEventContent = (arg: { event: { start: Date | null; end: Date | null } }) => {
    const start = arg.event.start
    const end = arg.event.end
    if (!start || !end) return {}

    const timeRange = formatTimeRange(new Date(start), new Date(end))
    return {
      html: `
        <div class="custom-event-content">
          <div class="event-time-range">${timeRange}</div>
        </div>
      `,
    }
  }

  const CALENDAR_DEFAULT_OPTIONS = {
    headerToolbar: {
      left: 'prev',
      center: 'title',
      right: 'next',
    },
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
    dayHeaderDidMount: handleDayHeaderDidMount,
    selectMinDistance: 5,
    eventOverlap: false,
  } as const

  const calendarOptions = computed(() => ({
    plugins: [timeGridPlugin, interactionPlugin],
    ...CALENDAR_DEFAULT_OPTIONS,
    initialView: initialView.value,
    editable: props.editable,
    selectable: props.editable,
    eventDurationEditable: props.editable,
    eventResizableFromStart: props.editable,
    displayEventTime: false,
    eventContent: handleEventContent,
    eventClassNames: handleEventClassNames,
    eventDidMount: handleEventDidMount,
    eventWillUnmount: handleEventWillUnmount,
    businessHours: props.businessHours,
    eventConstraint: props.constraint,
    selectConstraint: props.constraint,
    selectAllow: handleSelectAllow,
    eventAllow: handleEventAllow,
    select: handleDateSelect,
    dateClick: handleSlotClick,
    eventClick: handleEventClick,
    eventDrop: handleEventDrop,
    eventResizeStart: handleEventResizeStart,
    eventResize: handleEventResize,
    dayHeaderFormat: dayHeaderFormat.value,
    longPressDelay: longPressDelay.value,
    eventLongPressDelay: longPressDelay.value,
    selectLongPressDelay: longPressDelay.value,
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
  <div class="calendar-container" @click="handleContainerClick">
    <FullCalendar ref="calendarRef" :options="calendarOptions" />

    <!-- Mobile delete button (appears when event is selected on mobile) -->
    <Transition
      enter-active-class="transition-opacity duration-200"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition-opacity duration-150"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <button
        v-if="selectedEventId && isMobile"
        type="button"
        class="mobile-delete-btn"
        @click="handleDeleteButtonClick($event, selectedEventId)"
      >
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
          />
        </svg>
        Delete
      </button>
    </Transition>

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
        class="fixed inset-0 z-50 flex items-center justify-center bg-(--ink-primary)/30 backdrop-blur-md"
        role="dialog"
        aria-modal="true"
        aria-labelledby="deleteDialogTitle"
        @keydown.esc="handleEscapeKey"
      >
        <div
          class="transition-all duration-300 ease-out scale-in bg-(--paper-white) rounded-2xl shadow-2xl border border-(--border-subtle) p-8 w-full max-w-sm mx-4"
        >
          <div class="flex items-start gap-4 mb-5">
            <div
              class="shrink-0 w-12 h-12 rounded-sm bg-red-50 flex items-center justify-center border border-red-100"
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
              <h3
                id="deleteDialogTitle"
                class="text-lg font-semibold text-(--ink-primary) mb-1.5 tracking-tight"
              >
                Delete Time Slot
              </h3>
              <p class="text-sm text-(--text-secondary) leading-relaxed">
                Are you sure you want to delete this time slot? This action cannot be undone.
              </p>
            </div>
          </div>
          <div class="flex gap-3 justify-end">
            <button
              ref="cancelBtnRef"
              type="button"
              class="px-5 py-2.5 text-sm font-semibold text-(--ink-primary) bg-white border-2 border-(--border-subtle) rounded-xl hover:bg-(--paper-cream) focus:ring-2 focus:ring-(--accent-sage) transition-all"
              @click="closeDeleteDialog"
            >
              Cancel
            </button>
            <button
              type="button"
              class="px-5 py-2.5 text-sm font-semibold text-white rounded-xl hover:opacity-90 focus:ring-2 focus:ring-(--accent-sage) transition-all shadow-md tracking-wide"
              style="background: linear-gradient(135deg, #e8a598 0%, #f5c7bf 100%)"
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

<style scoped>
  .calendar-container {
    border: 1px solid var(--border-subtle);
    border-radius: 16px;
    overflow: hidden;
    height: 100%;
    background: var(--paper-white, #fff);
    box-shadow: 0 4px 16px rgba(45, 74, 62, 0.06);
    -webkit-tap-highlight-color: transparent;
  }

  .calendar-container * {
    -webkit-tap-highlight-color: transparent;
  }

  @media (max-width: 767px) {
    .calendar-container {
      overflow-y: auto;
    }

    /* Visual feedback during drag/resize on mobile */
    :deep(.fc-event.fc-event-selected) {
      opacity: 1;
      box-shadow: 0 0 0 3px var(--accent-sage);
    }

    /* Make resize handles very visible on mobile when selected */
    :deep(.fc-event.fc-event-selected .fc-event-resizer) {
      width: 20px !important;
      height: 20px !important;
      background: rgba(157, 180, 160, 0.9) !important;
      border: 2px solid white !important;
      opacity: 1 !important;
    }
  }

  /* Mobile delete button */
  .mobile-delete-btn {
    position: fixed;
    bottom: 2rem;
    left: 50%;
    transform: translateX(-50%);
    z-index: 30;
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.875rem 1.5rem;
    font-size: 0.9375rem;
    font-weight: 600;
    font-family: 'Inter', sans-serif;
    color: white;
    background: linear-gradient(135deg, #e8a598 0%, #f5c7bf 100%);
    border: none;
    border-radius: 16px;
    box-shadow: 0 6px 20px rgba(232, 165, 152, 0.4);
    cursor: pointer;
    transition: all 0.2s ease;
    -webkit-tap-highlight-color: transparent;
  }

  .mobile-delete-btn:active {
    transform: translateX(-50%) scale(0.96);
  }

  /* Show resize handles on selected events */
  :deep(.fc-event-selected .fc-event-resizer) {
    background: rgba(157, 180, 160, 0.4);
    width: 12px;
    height: 12px;
  }
</style>
