import { watch, type Ref } from 'vue'
import type FullCalendar from '@fullcalendar/vue3'
import type { EventInput } from '@fullcalendar/core'

type CalendarRef = Ref<InstanceType<typeof FullCalendar> | null>

const collectIds = (events: EventInput[] | undefined): Set<string> =>
  new Set((events ?? []).map((e) => e.id).filter((id): id is string => id != null))

/**
 * Keeps the FullCalendar instance in sync with a reactive array of events by
 * diffing ids: events dropped from the source are removed, new ones are added.
 * Used for both the editable model events and the read-only suggestion events;
 * `transform` lets the caller adjust each event before it is added (e.g. to
 * mark suggestions as non-editable).
 */
export function useCalendarEventSync(
  calendarRef: CalendarRef,
  source: () => EventInput[] | undefined,
  transform?: (event: EventInput) => EventInput
) {
  watch(
    source,
    (newEvents, oldEvents) => {
      const api = calendarRef.value?.getApi()
      if (!api) return

      const oldIds = collectIds(oldEvents)
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
