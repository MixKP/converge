<script setup lang="ts">
  import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
  import FullCalendar from '@fullcalendar/vue3'
  import timeGridPlugin from '@fullcalendar/timegrid'
  import interactionPlugin from '@fullcalendar/interaction'
  import { isSameDaySelection, createEvent, createOneHourEvent } from '../utils/calendarHelpers'
  import type {
    EventInput,
    DateSelectArg,
    EventClickArg,
    EventDropArg,
    BusinessHoursInput,
    CalendarOptions,
  } from '@fullcalendar/core'

  interface BusinessHourItem {
    daysOfWeek: number[]
    startTime?: string
    endTime?: string
    start?: string
    end?: string
  }

  interface Props {
    editable?: boolean
    businessHours?: BusinessHoursInput
    constraint?: string | 'businessHours'
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
  const screenWidth = ref(0)
  const updateScreenWidth = () => {
    screenWidth.value = globalThis.innerWidth
  }
  onMounted(() => {
    updateScreenWidth()
    globalThis.addEventListener('resize', updateScreenWidth)
  })
  onUnmounted(() => {
    globalThis.removeEventListener('resize', updateScreenWidth)
  })
  const dayHeaderFormat = computed(() => {
    if (screenWidth.value < 640) {
      // Mobile: S 7
      return { weekday: 'narrow' as const, day: 'numeric' as const }
    }
    if (screenWidth.value < 1024) {
      // Tablet: Sun 7
      return { weekday: 'short' as const, day: 'numeric' as const }
    }
    // Desktop: Sunday 7
    return { weekday: 'long' as const, day: 'numeric' as const }
  })

  const initialView = computed(() => {
    // Mobile: show day view, Tablet/Desktop: show week view
    return screenWidth.value < 768 ? 'timeGridDay' : 'timeGridWeek'
  })
  const showDeleteDialog = ref(false)
  const eventToDelete = ref<string | null>(null)
  const cancelBtnRef = ref<HTMLButtonElement | null>(null)
  const previouslyFocused = ref<HTMLElement | null>(null)
  const selectedEventId = ref<string | null>(null)
  const selectedEventEl = ref<HTMLElement | null>(null)
  const lastDragTime = ref(0)
  const handleEscapeKey = () => {
    if (showDeleteDialog.value) {
      closeDeleteDialog()
    } else if (selectedEventId.value) {
      deselectEvent()
    }
  }
  const getCurrentEvents = (): EventInput[] => {
    const api = calendarRef.value?.getApi()
    if (!api) return []
    return api
      .getEvents()
      .filter((e) => e.start != null && e.end != null)
      .map((e) => ({
        id: e.id,
        start: e.start!,
        end: e.end!,
        title: e.title,
      }))
  }
  const handleDateSelect = (selectInfo: DateSelectArg) => {
    if (!props.editable) return
    const calendar = selectInfo.view.calendar
    calendar.unselect()

    const newEvent = createEvent(selectInfo.start, selectInfo.end)
    emit('update:modelValue', [...(props.modelValue || []), newEvent])
  }

  const handleSelectAllow = (selectInfo: { start: Date; end: Date }) => {
    return isSameDaySelection(selectInfo.start, selectInfo.end)
  }

  const handleEventAllow = (dropInfo: { start: Date | null; end: Date | null }) => {
    const start = dropInfo.start
    const end = dropInfo.end
    if (!start || !end) return false
    return isSameDay(start, end)
  }

  const handleSlotClick = (arg: { dateStr: string }) => {
    if (!props.editable) return
    const newEvent = createOneHourEvent(new Date(arg.dateStr))
    emit('update:modelValue', [...(props.modelValue || []), newEvent])
  }

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

  const handleEventContent = (arg: {
    event: { start: Date | null; end: Date | null; id: string }
  }) => {
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
  const lastClickTime = ref(0)
  const lastClickedEventId = ref<string | null>(null)
  const DOUBLE_CLICK_DELAY = 300

  const handleEventClick = (clickInfo: EventClickArg) => {
    const id = clickInfo.event.id
    if (!id) return

    // Ignore clicks right after drag/resize
    const now = Date.now()
    if (now - lastDragTime.value < 200) {
      return
    }

    // Check if clicking on resize handle or delete button
    const target = clickInfo.jsEvent.target as HTMLElement
    if (target.closest('.fc-event-resizer') || target.closest('.event-delete-btn')) {
      return
    }

    const isSuggestionEvent =
      id.startsWith('suggestion-') ||
      !Array.isArray(props.modelValue) ||
      !props.modelValue.some((e) => e.id === id)
    if (isSuggestionEvent) {
      emit('event-click', clickInfo)
      return
    }

    // Check for double click (desktop only)
    const timeSinceLastClick = now - lastClickTime.value
    const isDoubleClick = timeSinceLastClick < DOUBLE_CLICK_DELAY && lastClickedEventId.value === id
    const isDesktop = screenWidth.value >= 768

    if (isDesktop && isDoubleClick) {
      // Double click → show delete dialog
      eventToDelete.value = id
      showDeleteDialog.value = true
      deselectEvent()
    } else {
      // Single click → select event (show resize handles)
      deselectEvent()
      selectedEventId.value = id
      selectedEventEl.value = clickInfo.el as HTMLElement
    }

    lastClickTime.value = now
    lastClickedEventId.value = id
  }

  const handleDeleteButtonClick = (e: MouseEvent, eventId: string) => {
    e.stopPropagation()
    eventToDelete.value = eventId
    showDeleteDialog.value = true
    deselectEvent()
  }

  const deselectEvent = () => {
    selectedEventId.value = null
    selectedEventEl.value = null
  }

  const handleEventClassNames = (arg: { event: { id: string } }): string[] => {
    const classes: string[] = []
    if (arg.event.id === selectedEventId.value) {
      classes.push('fc-event-selected')
    }
    return classes
  }

  const handleEventDidMount = (info: { event: { id: string }; el: HTMLElement }) => {
    const eventId = info.event.id
    if (!eventId || eventId.startsWith('suggestion-')) return

    // Check if this is a user event (editable)
    const isUserEvent =
      Array.isArray(props.modelValue) && props.modelValue.some((e) => e.id === eventId)
    if (!isUserEvent) return

    // Add click handler for touch devices
    const el = info.el
    let touchStartTime = 0

    el.addEventListener(
      'touchstart',
      (_e: Event) => {
        touchStartTime = Date.now()
      },
      { passive: true }
    )

    el.addEventListener(
      'touchend',
      (e: Event) => {
        const touchDuration = Date.now() - touchStartTime

        // Quick tap = select event
        if (touchDuration < 300) {
          const touch = (e as TouchEvent).changedTouches[0]
          const target = touch.target as HTMLElement

          // Don't select if tapping on resize handle
          if (target.closest('.fc-event-resizer') || target.closest('.event-delete-btn')) {
            return
          }

          // Select the event
          deselectEvent()
          selectedEventId.value = eventId
          selectedEventEl.value = el
        }
      },
      { passive: true }
    )
  }

  const handleContainerClick = (e: MouseEvent) => {
    // If clicking on the container (not an event), deselect
    if ((e.target as HTMLElement).classList.contains('calendar-container')) {
      deselectEvent()
    }
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
  const isSameDay = (date1: Date, date2: Date): boolean => {
    return (
      date1.getFullYear() === date2.getFullYear() &&
      date1.getMonth() === date2.getMonth() &&
      date1.getDate() === date2.getDate()
    )
  }

  const handleEventChange = () => {
    emit('update:modelValue', getCurrentEvents())
  }
  const handleEventDrop = (info: EventDropArg) => {
    const oldStart = info.oldEvent.start
    const newStart = info.event.start
    if (!oldStart || !newStart) {
      info.revert()
      return
    }
    if (!isSameDay(oldStart, newStart)) {
      info.revert()
      return
    }
    lastDragTime.value = Date.now()
    handleEventChange()
  }
  const handleEventResizeStart = () => {
    // Don't deselect when entering resize mode - keeps selection visible
  }
  const handleEventResize = (info: {
    event: { start: Date | null; end: Date | null }
    revert: () => void
  }) => {
    const start = info.event.start
    const end = info.event.end
    if (!start || !end) {
      info.revert()
      return
    }
    if (!isSameDay(start, end)) {
      info.revert()
      return
    }
    lastDragTime.value = Date.now()
    handleEventChange()
  }
  const handleDayHeaderDidMount = (info: { el: HTMLElement }) => {
    if (!props.businessHours) return
    const api = calendarRef.value?.getApi()
    if (!api) return

    const headerDate = new Date(info.el.getAttribute('data-date') || '')
    const dayOfWeek = headerDate.getDay()

    const businessHoursArray = (
      Array.isArray(props.businessHours) ? props.businessHours : []
    ) as BusinessHourItem[]
    const hasAvailability = businessHoursArray.some((hour: BusinessHourItem) => {
      if (Array.isArray(hour.daysOfWeek)) {
        return hour.daysOfWeek.includes(dayOfWeek)
      }
      return false
    })

    if (hasAvailability) {
      info.el.classList.add('has-availability')
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
    longPressDelay: screenWidth.value < 768 ? 400 : 50,
    eventLongPressDelay: screenWidth.value < 768 ? 400 : 50,
    selectLongPressDelay: screenWidth.value < 768 ? 400 : 50,
    selectMinDistance: 5,
    eventOverlap: false,
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
      for (const id of oldIds) {
        if (!newIds.has(id)) {
          api.getEventById(id)?.remove()
        }
      }
      for (const event of newEvents ?? []) {
        if (!event.id) continue
        const existing = api.getEventById(event.id)
        if (!existing) {
          api.addEvent(event)
        }
      }
    },
    { immediate: true }
  )
  watch(
    () => props.additionalEvents,
    (newEvents, oldEvents) => {
      const api = calendarRef.value?.getApi()
      if (!api) return
      const oldIds = new Set(
        (oldEvents ?? []).map((e) => e.id).filter((id): id is string => id != null)
      )
      const newIds = new Set(
        (newEvents ?? []).map((e) => e.id).filter((id): id is string => id != null)
      )
      for (const id of oldIds) {
        if (!newIds.has(id)) {
          api.getEventById(id)?.remove()
        }
      }
      for (const event of newEvents ?? []) {
        if (!event.id) continue
        const existing = api.getEventById(event.id)
        if (!existing) {
          api.addEvent({ ...event, editable: false })
        }
      }
    },
    { immediate: true }
  )
  watch(
    () => props.businessHours,
    (newBusinessHours) => {
      const api = calendarRef.value?.getApi()
      if (api) {
        api.setOption('businessHours', newBusinessHours)
        // Update day header classes based on availability
        const headerEls = api.el.querySelectorAll('.fc-col-header-cell') as NodeListOf<HTMLElement>
        headerEls.forEach((headerEl) => {
          const dateStr = headerEl.getAttribute('data-date')
          if (!dateStr) return
          const headerDate = new Date(dateStr)
          const dayOfWeek = headerDate.getDay()

          const businessHoursArray = (
            Array.isArray(newBusinessHours) ? newBusinessHours : []
          ) as BusinessHourItem[]
          const hasAvailability = businessHoursArray.some((hour: BusinessHourItem) => {
            if (Array.isArray(hour.daysOfWeek)) {
              return hour.daysOfWeek.includes(dayOfWeek)
            }
            return false
          })

          if (hasAvailability) {
            headerEl.classList.add('has-availability')
          } else {
            headerEl.classList.remove('has-availability')
          }
        })
      }
    }
  )
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
  }))

  const setOption = <K extends keyof CalendarOptions>(key: K, value: CalendarOptions[K]) => {
    const api = calendarRef.value?.getApi()
    if (api) {
      api.setOption(key, value)
    }
  }

  // Watch for screen width changes to update day header format and view
  watch(screenWidth, () => {
    setOption('dayHeaderFormat', dayHeaderFormat.value)
    // Change view based on screen size
    const api = calendarRef.value?.getApi()
    if (api) {
      const newView = initialView.value
      if (api.view.type !== newView) {
        api.changeView(newView)
      }
    }
  })

  watch(showDeleteDialog, (isOpen) => {
    if (isOpen) {
      cancelBtnRef.value?.focus()
    } else {
      previouslyFocused.value?.focus()
      previouslyFocused.value = null
    }
  })
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
        v-if="selectedEventId && screenWidth < 768"
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

  /* Event action menu */
  .event-action-menu {
    position: absolute;
    bottom: 1rem;
    left: 50%;
    transform: translateX(-50%);
    z-index: 30;
    width: calc(100% - 2rem);
    max-width: 320px;
  }

  .event-action-card {
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: 16px;
    padding: 1rem 1.25rem;
    box-shadow: var(--shadow-elevated);
  }

  .event-action-title {
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--text-secondary);
    margin-bottom: 0.75rem;
    text-align: center;
  }

  .event-action-buttons {
    display: flex;
    gap: 0.75rem;
  }

  .event-action-btn {
    flex: 1;
    padding: 0.75rem 1rem;
    font-size: 0.875rem;
    font-weight: 600;
    font-family: 'Inter', sans-serif;
    border-radius: 12px;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .event-action-btn--cancel {
    color: var(--text-primary);
    background: var(--bg-subtle);
    border: 2px solid var(--border-subtle);
  }

  .event-action-btn--cancel:hover {
    background: var(--border-subtle);
  }

  .event-action-btn--delete {
    color: white;
    background: linear-gradient(135deg, #e8a598 0%, #f5c7bf 100%);
    border: none;
    box-shadow: 0 4px 12px rgba(232, 165, 152, 0.3);
  }

  .event-action-btn--delete:hover {
    transform: translateY(-1px);
    box-shadow: 0 6px 16px rgba(232, 165, 152, 0.4);
  }

  .event-action-btn--resize {
    color: white;
    background: linear-gradient(135deg, var(--accent-sage) 0%, var(--accent-mint) 100%);
    border: none;
    box-shadow: 0 4px 12px rgba(157, 180, 160, 0.3);
  }

  .event-action-btn--resize:hover {
    transform: translateY(-1px);
    box-shadow: 0 6px 16px rgba(157, 180, 160, 0.4);
  }

  /* Show resize handles on selected events */
  :deep(.fc-event-selected .fc-event-resizer) {
    background: rgba(157, 180, 160, 0.4);
    width: 12px;
    height: 12px;
  }
</style>
