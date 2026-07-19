import { ref, watch, onUnmounted, type Ref } from 'vue'
import type FullCalendar from '@fullcalendar/vue3'
import type { EventInput, DateSelectArg, EventClickArg, EventDropArg } from '@fullcalendar/core'
import {
  isSameDay,
  isSameDaySelection,
  createEvent,
  createOneHourEvent,
} from '../utils/calendarHelpers'
import { isValidDate, rangesOverlap } from '../utils/dateValidation'

type CalendarRef = Ref<InstanceType<typeof FullCalendar> | null>

interface InteractionOptions {
  calendarRef: CalendarRef
  isMobile: Ref<boolean>
  /** Cancel button in the delete dialog, focused when the dialog opens. */
  cancelBtnRef: Ref<HTMLButtonElement | null>
  isEditable: () => boolean
  getModelValue: () => EventInput[]
  getAdditionalEvents: () => EventInput[]
  onUpdate: (events: EventInput[]) => void
  onEventClick: (info: EventClickArg) => void
}

const DOUBLE_CLICK_DELAY = 300

/**
 * Owns all user interaction with editable calendar events: creating slots via
 * select/click, constraining moves/resizes to the same day and away from cart
 * events, event selection, and the delete-confirmation dialog (including the
 * touch handlers that make selection work on mobile).
 */
export function useCalendarInteraction(options: InteractionOptions) {
  const {
    calendarRef,
    isMobile,
    cancelBtnRef,
    isEditable,
    getModelValue,
    getAdditionalEvents,
    onUpdate,
    onEventClick,
  } = options

  // ---- Cart overlap -------------------------------------------------------

  // Check if a time range overlaps with any cart event. Cart events can live in
  // either the model events or the suggestion events, so both are considered.
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

  // ---- Event creation & constraints ---------------------------------------

  const handleDateSelect = (selectInfo: DateSelectArg) => {
    if (!isEditable()) return
    if (overlapsWithCartEvent(selectInfo.start, selectInfo.end)) return

    selectInfo.view.calendar.unselect()
    onUpdate([...getModelValue(), createEvent(selectInfo.start, selectInfo.end)])
  }

  const handleSelectAllow = (selectInfo: { start: Date; end: Date }) => {
    if (!isSameDaySelection(selectInfo.start, selectInfo.end)) return false
    if (overlapsWithCartEvent(selectInfo.start, selectInfo.end)) return false
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

    onUpdate([...getModelValue(), createOneHourEvent(new Date(arg.dateStr))])
  }

  // ---- Move / resize ------------------------------------------------------

  const lastDragTime = ref(0)

  const getCurrentEvents = (): EventInput[] => {
    const api = calendarRef.value?.getApi()
    if (!api) return []
    return api
      .getEvents()
      .filter((e) => e.start != null && e.end != null)
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

  const handleEventResizeStart = () => {
    // Don't deselect when entering resize mode - keeps selection visible
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

  // ---- Selection ----------------------------------------------------------

  const SELECTED_CLASS = 'fc-event-selected'
  const selectedEventId = ref<string | null>(null)
  const selectedEventEl = ref<HTMLElement | null>(null)

  const deselectEvent = () => {
    // Remove the class imperatively so the resize handles disappear right away;
    // FullCalendar does not re-render the event just because the ref changes.
    selectedEventEl.value?.classList.remove(SELECTED_CLASS)
    selectedEventId.value = null
    selectedEventEl.value = null
  }

  const selectEvent = (id: string, el: HTMLElement) => {
    deselectEvent()
    selectedEventId.value = id
    selectedEventEl.value = el
    // Apply the class now (not only on the next render) so the resize handles
    // become grabbable on the very first touch, without a throwaway drag first.
    el.classList.add(SELECTED_CLASS)
  }

  const handleContainerClick = (e: MouseEvent) => {
    // Clicking the container background (not an event) deselects.
    if ((e.target as HTMLElement).classList.contains('calendar-container')) {
      deselectEvent()
    }
  }

  const handleEventClassNames = (arg: { event: { id: string } }): string[] =>
    arg.event.id === selectedEventId.value ? [SELECTED_CLASS] : []

  const isUserEvent = (id: string): boolean => getModelValue().some((e) => e.id === id)

  // ---- Deletion dialog ----------------------------------------------------

  const showDeleteDialog = ref(false)
  const eventToDelete = ref<string | null>(null)
  const previouslyFocused = ref<HTMLElement | null>(null)

  const openDeleteDialog = (id: string) => {
    eventToDelete.value = id
    showDeleteDialog.value = true
    deselectEvent()
  }

  const closeDeleteDialog = () => {
    showDeleteDialog.value = false
    eventToDelete.value = null
  }

  const confirmDelete = () => {
    if (eventToDelete.value) {
      onUpdate(getModelValue().filter((e) => e.id !== eventToDelete.value))
    }
    closeDeleteDialog()
  }

  const handleDeleteButtonClick = (e: MouseEvent, eventId: string) => {
    e.stopPropagation()
    openDeleteDialog(eventId)
  }

  const handleEscapeKey = () => {
    if (showDeleteDialog.value) {
      closeDeleteDialog()
    } else if (selectedEventId.value) {
      deselectEvent()
    }
  }

  watch(showDeleteDialog, (isOpen) => {
    if (isOpen) {
      cancelBtnRef.value?.focus()
    } else {
      previouslyFocused.value?.focus()
      previouslyFocused.value = null
    }
  })

  // ---- Click handling -----------------------------------------------------

  const lastClickTime = ref(0)
  const lastClickedEventId = ref<string | null>(null)

  const handleEventClick = (clickInfo: EventClickArg) => {
    const id = clickInfo.event.id
    if (!id) return

    // Ignore clicks fired right after a drag/resize.
    const now = Date.now()
    if (now - lastDragTime.value < 200) return

    // Ignore clicks on the resize handle or delete button.
    const target = clickInfo.jsEvent.target as HTMLElement
    if (target.closest('.fc-event-resizer') || target.closest('.event-delete-btn')) return

    const isSuggestionEvent = id.startsWith('suggestion-') || !isUserEvent(id)
    if (isSuggestionEvent) {
      onEventClick(clickInfo)
      return
    }

    const timeSinceLastClick = now - lastClickTime.value
    const isDoubleClick = timeSinceLastClick < DOUBLE_CLICK_DELAY && lastClickedEventId.value === id

    if (!isMobile.value && isDoubleClick) {
      // Double click → delete dialog
      openDeleteDialog(id)
    } else {
      // Single click → select event (show resize handles)
      selectEvent(id, clickInfo.el as HTMLElement)
    }

    lastClickTime.value = now
    lastClickedEventId.value = id
  }

  // ---- Touch selection (mobile) -------------------------------------------

  // Store touch listener references per element for cleanup.
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

    // A re-render (e.g. after a resize) replaces the DOM node while keeping the
    // selection. Point selectedEventEl at the fresh node so a later deselect
    // clears the class off the element the user actually sees.
    if (eventId === selectedEventId.value) {
      selectedEventEl.value = el
    }

    let touchStartTime = 0

    const touchStartHandler = () => {
      touchStartTime = Date.now()
    }

    const touchEndHandler = (e: Event) => {
      // Quick tap = select event
      if (Date.now() - touchStartTime >= 300) return

      const touch = (e as TouchEvent).changedTouches[0]
      const target = touch.target as HTMLElement
      // Don't select when tapping a resize handle or the delete button.
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
    // template state
    selectedEventId,
    showDeleteDialog,
    handleContainerClick,
    handleEscapeKey,
    handleDeleteButtonClick,
    closeDeleteDialog,
    confirmDelete,
    // calendar option handlers
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
  }
}
