<script setup lang="ts">
  import { ref, computed } from 'vue'
  import FullCalendar from '@fullcalendar/vue3'
  import timeGridPlugin from '@fullcalendar/timegrid'
  import interactionPlugin from '@fullcalendar/interaction'
  import { useCalendarResponsive } from '../composables/cart/useCalendarResponsive'
  import { useCalendarEventSync } from '../composables/cart/useCalendarEventSync'
  import { useBusinessHoursHeaders } from '../composables/cart/useBusinessHoursHeaders'
  import { useCalendarInteraction } from '../composables/cart/useCalendarInteraction'
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
    handleDeleteButtonClick,
    closeDeleteDialog,
    confirmDelete,
    handleDateSelect,
    handleSelectAllow,
    handleEventAllow,
    handleSlotClick,
    handleEventClick,
    handleEventDrop,
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
  <div
    class="calendar-container h-full overflow-x-hidden overflow-y-auto rounded-2xl border border-(--border-subtle) bg-(--paper-white) shadow-[0_4px_16px_rgba(45,74,62,0.06)] [-webkit-tap-highlight-color:transparent] md:overflow-y-hidden **:[-webkit-tap-highlight-color:transparent]"
    @click="handleContainerClick"
  >
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
        class="fixed bottom-8 left-1/2 z-30 flex -translate-x-1/2 cursor-pointer items-center gap-2 rounded-2xl px-6 py-3.5 text-[0.9375rem] font-semibold text-white shadow-[0_6px_20px_rgba(232,165,152,0.4)] transition-all duration-200 [-webkit-tap-highlight-color:transparent] active:scale-[0.96]"
        style="background: linear-gradient(135deg, #e8a598 0%, #f5c7bf 100%)"
        @click="handleDeleteButtonClick($event, selectedEventId)"
      >
        <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
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
      >
        <div
          class="scale-in mx-4 w-full max-w-sm rounded-2xl border border-(--border-subtle) bg-(--paper-white) p-8 shadow-2xl transition-all duration-300 ease-out"
        >
          <div class="mb-5 flex items-start gap-4">
            <div
              class="flex h-12 w-12 shrink-0 items-center justify-center rounded-sm border border-red-100 bg-red-50"
            >
              <svg
                class="h-6 w-6 text-red-600"
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
                class="mb-1.5 text-lg font-semibold tracking-tight text-(--ink-primary)"
              >
                Delete Time Slot
              </h3>
              <p class="text-sm leading-relaxed text-(--text-secondary)">
                Are you sure you want to delete this time slot? This action cannot be undone.
              </p>
            </div>
          </div>
          <div class="flex justify-end gap-3">
            <button
              ref="cancelBtnRef"
              type="button"
              class="rounded-xl border-2 border-(--border-subtle) bg-white px-5 py-2.5 text-sm font-semibold text-(--ink-primary) transition-all hover:bg-(--paper-cream) focus:ring-2 focus:ring-(--accent-sage)"
              @click="closeDeleteDialog"
            >
              Cancel
            </button>
            <button
              type="button"
              class="rounded-xl px-5 py-2.5 text-sm font-semibold tracking-wide text-white shadow-md transition-all hover:opacity-90 focus:ring-2 focus:ring-(--accent-sage)"
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
  /* FullCalendar renders its own DOM at runtime, so these overrides target its
     internal elements via :deep() and must stay as CSS — they cannot be moved
     onto utility classes in a template we don't own. */

  @media (max-width: 767px) {
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

  /* Show resize handles on selected events */
  :deep(.fc-event-selected .fc-event-resizer) {
    background: rgba(157, 180, 160, 0.4);
    width: 12px;
    height: 12px;
  }
</style>
