import { ref, onUnmounted, type Ref } from 'vue'
import type FullCalendar from '@fullcalendar/vue3'
import type { EventInput, DateSelectArg, EventClickArg, EventDropArg } from '@fullcalendar/core'
import {
  isSameDay,
  isSameDaySelection,
  createEvent,
  createOneHourEvent,
} from '../../utils/calendarHelpers'
import { isValidDate, rangesOverlap } from '../../utils/dateValidation'

type CalendarRef = Ref<InstanceType<typeof FullCalendar> | null>

interface InteractionOptions {
  calendarRef: CalendarRef
  isMobile: Ref<boolean>
  isEditable: () => boolean
  getModelValue: () => EventInput[]
  getAdditionalEvents: () => EventInput[]
  onUpdate: (events: EventInput[]) => void
  onEventClick: (info: EventClickArg) => void

  isSlotAllowed?: (start: Date, end: Date) => boolean
}

const DOUBLE_CLICK_DELAY = 300

export function useCalendarInteraction(options: InteractionOptions) {
  const {
    calendarRef,
    isMobile,
    isEditable,
    getModelValue,
    getAdditionalEvents,
    onUpdate,
    onEventClick,
    isSlotAllowed = () => true,
  } = options

  const overlapsWithCartEvent = (start: Date, end: Date): boolean => {
    const allEvents = [...getModelValue(), ...getAdditionalEvents()]
    const cartEvents = allEvents.filter((e) => e.extendedProps?.isCartItem === true)
    for (const cartEvent of cartEvents) {
      if (!cartEvent.start || !cartEvent.end) continue

      const cartStart = new Date(cartEvent.start as Date)
      const cartEnd = new Date(cartEvent.end as Date)
      if (!isValidDate(cartStart) || !isValidDate(cartEnd)) continue

      if (rangesOverlap(start, end, cartStart, cartEnd)) return true
    }
    return false
  }

  const editableModelValue = (): EventInput[] => getModelValue().filter((e) => e.editable !== false)

  const handleDateSelect = (selectInfo: DateSelectArg) => {
    if (!isEditable()) return
    if (overlapsWithCartEvent(selectInfo.start, selectInfo.end)) return

    selectInfo.view.calendar.unselect()
    onUpdate([...editableModelValue(), createEvent(selectInfo.start, selectInfo.end)])
  }

  const handleSelectAllow = (selectInfo: { start: Date; end: Date }) => {
    if (!isSameDaySelection(selectInfo.start, selectInfo.end)) return false
    if (overlapsWithCartEvent(selectInfo.start, selectInfo.end)) return false
    if (!isSlotAllowed(selectInfo.start, selectInfo.end)) return false
    return true
  }

  const handleEventAllow = (dropInfo: { start: Date | null; end: Date | null }) => {
    const { start, end } = dropInfo
    if (!start || !end) return false
    if (!isSameDay(start, end)) return false
    if (overlapsWithCartEvent(start, end)) return false
    return true
  }

  const handleSlotClick = (arg: { dateStr: string }) => {
    if (!isEditable()) return

    const clickDate = new Date(arg.dateStr)
    const endDate = new Date(clickDate)
    endDate.setHours(clickDate.getHours() + 1)
    if (overlapsWithCartEvent(clickDate, endDate)) return
    if (!isSlotAllowed(clickDate, endDate)) return

    onUpdate([...editableModelValue(), createOneHourEvent(new Date(arg.dateStr))])
  }

  const lastDragTime = ref(0)

  const getCurrentEvents = (): EventInput[] => {
    const api = calendarRef.value?.getApi()
    if (!api) return []
    return api
      .getEvents()
      .filter((e) => e.start != null && e.end != null)
      .filter(
        (e) =>
          !e.extendedProps?.isCartItem &&
          !e.extendedProps?.isBooked &&
          !e.extendedProps?.isSuggestion
      )
      .map((e) => ({ id: e.id, start: e.start!, end: e.end!, title: e.title }))
  }

  const handleEventChange = () => {
    onUpdate(getCurrentEvents())
  }

  const handleEventDrop = (info: EventDropArg) => {
    const oldStart = info.oldEvent.start
    const newStart = info.event.start
    if (!oldStart || !newStart || !isSameDay(oldStart, newStart)) {
      info.revert()
      return
    }
    lastDragTime.value = Date.now()
    handleEventChange()
  }

  const handleEventResize = (info: {
    event: { start: Date | null; end: Date | null }
    revert: () => void
  }) => {
    const { start, end } = info.event
    if (!start || !end || !isSameDay(start, end)) {
      info.revert()
      return
    }
    lastDragTime.value = Date.now()
    handleEventChange()
  }

  const SELECTED_CLASS = 'fc-event-selected'
  const selectedEventId = ref<string | null>(null)
  const selectedEventEl = ref<HTMLElement | null>(null)

  const deselectEvent = () => {
    selectedEventEl.value?.classList.remove(SELECTED_CLASS)
    selectedEventId.value = null
    selectedEventEl.value = null
  }

  const selectEvent = (id: string, el: HTMLElement) => {
    deselectEvent()
    selectedEventId.value = id
    selectedEventEl.value = el
    el.classList.add(SELECTED_CLASS)
  }

  const handleContainerClick = (e: MouseEvent) => {
    if ((e.target as HTMLElement).classList.contains('calendar-container')) {
      deselectEvent()
    }
  }

  const handleEventClassNames = (arg: { event: { id: string } }): string[] =>
    arg.event.id === selectedEventId.value ? [SELECTED_CLASS] : []

  const isUserEvent = (id: string): boolean => getModelValue().some((e) => e.id === id)

  const deleteEvent = (id: string) => {
    deselectEvent()
    onUpdate(editableModelValue().filter((e) => e.id !== id))
  }

  const handleDeleteButtonClick = (e: MouseEvent, eventId: string) => {
    e.stopPropagation()
    deleteEvent(eventId)
  }

  const lastClickTime = ref(0)
  const lastClickedEventId = ref<string | null>(null)

  const handleEventClick = (clickInfo: EventClickArg) => {
    const id = clickInfo.event.id
    if (!id) return

    const now = Date.now()
    if (now - lastDragTime.value < 200) return

    const target = clickInfo.jsEvent.target as HTMLElement
    if (target.closest('.fc-event-resizer')) return

    if (target.closest('.event-delete-btn')) {
      clickInfo.jsEvent.stopPropagation()
      deleteEvent(id)
      return
    }

    const isSuggestionEvent = id.startsWith('suggestion-') || !isUserEvent(id)
    if (isSuggestionEvent) {
      onEventClick(clickInfo)
      return
    }

    const timeSinceLastClick = now - lastClickTime.value
    const isDoubleClick = timeSinceLastClick < DOUBLE_CLICK_DELAY && lastClickedEventId.value === id

    if (!isMobile.value && isDoubleClick) {
      deleteEvent(id)
    } else {
      selectEvent(id, clickInfo.el as HTMLElement)
    }

    lastClickTime.value = now
    lastClickedEventId.value = id
  }

  const eventListeners = new Map<
    HTMLElement,
    { touchstart: (e: Event) => void; touchend: (e: Event) => void }
  >()

  const cleanupEventListeners = (el: HTMLElement) => {
    const listeners = eventListeners.get(el)
    if (listeners) {
      el.removeEventListener('touchstart', listeners.touchstart)
      el.removeEventListener('touchend', listeners.touchend)
      eventListeners.delete(el)
    }
  }

  const handleEventDidMount = (info: { event: { id: string }; el: HTMLElement }) => {
    const eventId = info.event.id
    if (!eventId || eventId.startsWith('suggestion-') || !isUserEvent(eventId)) return

    const el = info.el

    if (eventId === selectedEventId.value) {
      selectedEventEl.value = el
    }

    let touchStartTime = 0

    const touchStartHandler = () => {
      touchStartTime = Date.now()
    }

    const touchEndHandler = (e: Event) => {
      if (Date.now() - touchStartTime >= 300) return

      const touch = (e as TouchEvent).changedTouches[0]
      const target = touch.target as HTMLElement
      if (target.closest('.fc-event-resizer') || target.closest('.event-delete-btn')) return

      selectEvent(eventId, el)
    }

    el.addEventListener('touchstart', touchStartHandler, { passive: true })
    el.addEventListener('touchend', touchEndHandler, { passive: true })
    eventListeners.set(el, { touchstart: touchStartHandler, touchend: touchEndHandler })
  }

  const handleEventWillUnmount = (info: { el: HTMLElement }) => {
    cleanupEventListeners(info.el)
  }

  onUnmounted(() => {
    eventListeners.forEach((_, el) => cleanupEventListeners(el))
    eventListeners.clear()
  })

  return {
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
  }
}
