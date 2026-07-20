import { computed, watch, type Ref } from 'vue'
import type FullCalendar from '@fullcalendar/vue3'
import { useScreenSize } from '../useScreenSize'

type CalendarRef = Ref<InstanceType<typeof FullCalendar> | null>

export function useCalendarResponsive(calendarRef: CalendarRef) {
  const { screenWidth, isMobile, isDesktop } = useScreenSize()

  const dayHeaderFormat = computed(() => {
    if (isMobile.value) {
      return { weekday: 'long' as const, day: 'numeric' as const }
    }
    if (!isDesktop.value) {
      return { weekday: 'short' as const, day: 'numeric' as const }
    }
    return { weekday: 'long' as const, day: 'numeric' as const }
  })

  const initialView = computed(() => (isMobile.value ? 'timeGridDay' : 'timeGridWeek'))

  const longPressDelay = computed(() => (isMobile.value ? 400 : 50))

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
