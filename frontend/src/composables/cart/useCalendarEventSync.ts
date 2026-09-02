import { watch, type Ref } from 'vue'
import type FullCalendar from '@fullcalendar/vue3'
import type { EventInput } from '@fullcalendar/core'

type CalendarRef = Ref<InstanceType<typeof FullCalendar> | null>

const collectIds = (events: EventInput[] | undefined): Set<string> =>
  new Set((events ?? []).map((e) => e.id).filter((id): id is string => id != null))

export function useCalendarEventSync(
  calendarRef: CalendarRef,
  source: () => EventInput[] | undefined,
  transform?: (event: EventInput) => EventInput
) {
  watch(
    [calendarRef, source] as const,
    ([, newEvents], previous) => {
      const api = calendarRef.value?.getApi()
      if (!api) return

      const oldIds = collectIds(previous?.[1])
      const newIds = collectIds(newEvents)

      for (const id of oldIds) {
        if (!newIds.has(id)) {
          api.getEventById(id)?.remove()
        }
      }
      for (const event of newEvents ?? []) {
        if (!event.id) continue
        if (!api.getEventById(event.id)) {
          api.addEvent(transform ? transform(event) : event)
        }
      }
    },
    { immediate: true }
  )
}
