import type { EventInput } from '@fullcalendar/core'
import type { AvailabilityPayload } from '../types'

export function generateAvailabilityPayload(
  events: EventInput[],
  teacherId: number
): AvailabilityPayload {
  const weeklyMap = new Map<string, { day_of_week: number; start: string; end: string }>()

  for (const event of events) {
    if (!event.start || !event.end) continue

    const start = new Date(event.start as Date)
    const dayOfWeek = start.getDay()
    const startTime = start.toTimeString().slice(0, 5)

    const end = new Date(event.end as Date)
    const endTime = end.toTimeString().slice(0, 5)

    const key = `${dayOfWeek}-${startTime}-${endTime}`
    if (!weeklyMap.has(key)) {
      weeklyMap.set(key, { day_of_week: dayOfWeek, start: startTime, end: endTime })
    }
  }

  return { teacher_id: teacherId, weekly: Array.from(weeklyMap.values()) }
}
