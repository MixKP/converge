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
    DatesSetArg,
  } from '@fullcalendar/core'

  interface Props {
    editable?: boolean
    businessHours?: BusinessHoursInput
    constraint?: string
    modelValue?: EventInput[]
    additionalEvents?: EventInput[]

    showHeader?: boolean
  }

  const props = withDefaults(defineProps<Props>(), {
    editable: false,
    businessHours: undefined,
    constraint: undefined,
    modelValue: () => [],
    additionalEvents: () => [],
    showHeader: true,
  })
  const emit = defineEmits<{
    'update:modelValue': [value: EventInput[]]
    'event-click': [info: EventClickArg]

    'dates-set': [range: { start: Date; end: Date }]
  }>()

  const calendarRef = ref<InstanceType<typeof FullCalendar> | null>(null)

  const { isMobile, dayHeaderFormat, initialView, longPressDelay } =
    useCalendarResponsive(calendarRef)

  const { handleDayHeaderDidMount } = useBusinessHoursHeaders(
    calendarRef,
    () => props.businessHours
  )

  const minutesOfDay = (time: string): number => {
    const [hours, mins] = time.split(':').map(Number)
    return (hours || 0) * 60 + (mins || 0)
  }

  const isWithinConstraint = (start: Date, end: Date): boolean => {
    if (props.constraint !== 'businessHours') return true

    const windows = props.businessHours

    if (!Array.isArray(windows)) return windows === true

    const startMinutes = start.getHours() * 60 + start.getMinutes()
    const endMinutes = end.getHours() * 60 + end.getMinutes()

    return windows.some((window) => {
      const days = window.daysOfWeek
      if (Array.isArray(days) && !days.includes(start.getDay())) return false
      if (!window.startTime || !window.endTime) return false
      return (
        startMinutes >= minutesOfDay(String(window.startTime)) &&
        endMinutes <= minutesOfDay(String(window.endTime))
      )
    })
  }

  const {
    selectedEventId,
    handleContainerClick,
    handleDeleteButtonClick,
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
    isEditable: () => props.editable,
    getModelValue: () => props.modelValue ?? [],
    getAdditionalEvents: () => props.additionalEvents ?? [],
    onUpdate: (events) => emit('update:modelValue', events),
    onEventClick: (info) => emit('event-click', info),
    isSlotAllowed: isWithinConstraint,
  })

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

  const escapeHtml = (value: string): string =>
    value.replace(
      /[&<>"']/g,
      (char) =>
        ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[char] ?? char
    )

  const handleEventContent = (arg: {
    event: { start: Date | null; end: Date | null; extendedProps?: Record<string, unknown> }
  }) => {
    const start = arg.event.start
    const end = arg.event.end
    if (!start || !end) return {}

    const timeRange = formatTimeRange(new Date(start), new Date(end))
    const p = arg.event.extendedProps

    if (p?.isBrowse) {
      return {
        html: `
          <div class="custom-event-content browse-event-content">
            <div class="browse-event-label">${escapeHtml(String(p.browseLabel ?? ''))}</div>
            <div class="browse-event-time">${timeRange}</div>
          </div>
        `,
      }
    }

    const isRemovable = !p?.isCartItem && !p?.isBooked && !p?.isSuggestion

    const deleteButton = isRemovable
      ? `<button type="button" class="event-delete-btn" aria-label="Remove this time slot" title="Remove this time slot">
           <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true">
             <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
           </svg>
         </button>`
      : ''

    return {
      html: `
        <div class="custom-event-content">
          <div class="event-time-range">${timeRange}</div>
          ${deleteButton}
        </div>
      `,
    }
  }

  const CALENDAR_DEFAULT_OPTIONS = {
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
    headerToolbar: props.showHeader
      ? ({ left: 'prev', center: 'title', right: 'next' } as const)
      : (false as const),
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
    datesSet: (arg: DatesSetArg) => emit('dates-set', { start: arg.start, end: arg.end }),
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
    class="calendar-container h-full overflow-x-hidden overflow-y-auto rounded-2xl border border-(--border-subtle) bg-(--paper-white) shadow-[0_4px_16px_rgba(45,74,62,0.06)] [-webkit-tap-highlight-color:transparent] **:[-webkit-tap-highlight-color:transparent] md:overflow-y-hidden"
    @click="handleContainerClick"
  >
    <FullCalendar ref="calendarRef" :options="calendarOptions" />
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
  </div>
</template>

<style scoped>
  @media (max-width: 767px) {
    :deep(.fc-event.fc-event-selected) {
      opacity: 1;
      box-shadow: 0 0 0 3px var(--accent-sage);
    }

    :deep(.fc-event.fc-event-selected .fc-event-resizer) {
      width: 20px !important;
      height: 20px !important;
      background: rgba(157, 180, 160, 0.9) !important;
      border: 2px solid white !important;
      opacity: 1 !important;
    }
  }

  :deep(.fc-event-selected .fc-event-resizer) {
    background: rgba(157, 180, 160, 0.4);
    width: 12px;
    height: 12px;
  }

  :deep(.browse-event) {
    cursor: pointer;
    border-width: 1px;
    border-style: solid;
  }

  :deep(.custom-event-content) {
    max-width: 100%;
  }

  :deep(.event-time-range) {
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  :deep(.browse-event-content) {
    display: flex;
    flex-direction: column;
    align-items: stretch;
    justify-content: flex-start;
    gap: 1px;
    width: 100%;
    overflow: hidden;
  }

  :deep(.fc-event.browse-event .fc-event-main) {
    padding: 4px 5px;
    text-align: left;
    font-family: 'Inter', sans-serif;
  }

  :deep(.browse-event-label),
  :deep(.browse-event-time) {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    line-height: 1.25;
    font-family: 'Inter', sans-serif;
  }

  :deep(.browse-event-label) {
    font-size: 0.6875rem;
    font-weight: 600;
  }

  :deep(.browse-event-time) {
    font-size: 0.625rem;
    font-weight: 400;
    opacity: 0.75;
  }

  :deep(.custom-event-content) {
    position: relative;
    height: 100%;
  }

  :deep(.event-delete-btn) {
    position: absolute;
    top: 2px;
    right: 2px;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 18px;
    height: 18px;
    padding: 0;
    border: none;
    border-radius: 50%;
    background: rgba(255, 255, 255, 0.9);
    color: var(--accent-coral);
    cursor: pointer;
    opacity: 0;
    transition:
      opacity 0.15s ease,
      transform 0.15s ease;
  }

  :deep(.event-delete-btn svg) {
    width: 11px;
    height: 11px;
  }

  :deep(.event-delete-btn:hover) {
    background: #fff;
    transform: scale(1.1);
  }

  :deep(.event-delete-btn:focus-visible) {
    opacity: 1;
    outline: 2px solid var(--accent-sage);
    outline-offset: 1px;
  }

  @media (hover: hover) {
    :deep(.fc-event:hover .event-delete-btn),
    :deep(.fc-event-selected .event-delete-btn) {
      opacity: 1;
    }
  }

  @media (hover: none) {
    :deep(.event-delete-btn) {
      opacity: 1;
      width: 22px;
      height: 22px;
    }

    :deep(.event-delete-btn svg) {
      width: 13px;
      height: 13px;
    }
  }
</style>
