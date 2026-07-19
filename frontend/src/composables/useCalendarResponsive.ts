import { computed, watch, type Ref } from 'vue'
import type FullCalendar from '@fullcalendar/vue3'
import { useScreenSize } from './useScreenSize'

type CalendarRef = Ref<InstanceType<typeof FullCalendar> | null>

/**
 * Derives the screen-size dependent FullCalendar options (day-header format,
 * initial view and long-press delays) and imperatively re-applies them to the
 * live calendar whenever the viewport changes. Breakpoint decisions come from
 * useScreenSize so this never disagrees with the page-level layout.
 */
export function useCalendarResponsive(calendarRef: CalendarRef) {
  const { screenWidth, isMobile, isDesktop } = useScreenSize()

  const dayHeaderFormat = computed(() => {
    if (isMobile.value) {
      // Mobile: S 7
      return { weekday: 'narrow' as const, day: 'numeric' as const }
    }
    if (!isDesktop.value) {
      // Tablet: Sun 7
      return { weekday: 'short' as const, day: 'numeric' as const }
    }
    // Desktop: Sunday 7
    return { weekday: 'long' as const, day: 'numeric' as const }
  })

  // Mobile: show day view, Tablet/Desktop: show week view
  const initialView = computed(() => (isMobile.value ? 'timeGridDay' : 'timeGridWeek'))

  // Touch-oriented long-press delays are mobile-only; tablet and desktop share
  // the shorter "pointer" delay.
  const longPressDelay = computed(() => (isMobile.value ? 400 : 50))

  // FullCalendar's imperative API options need to be explicitly re-applied when
  // the breakpoint-derived values above change.
  watch(screenWidth, () => {
    const api = calendarRef.value?.getApi()
    if (!api) return
    api.setOption('dayHeaderFormat', dayHeaderFormat.value)
    api.setOption('longPressDelay', longPressDelay.value)
    api.setOption('eventLongPressDelay', longPressDelay.value)
    api.setOption('selectLongPressDelay', longPressDelay.value)
    if (api.view.type !== initialView.value) {
      api.changeView(initialView.value)
    }
  })

  return { isMobile, dayHeaderFormat, initialView, longPressDelay }
}
