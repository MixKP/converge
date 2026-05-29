import type { EventInput } from '@fullcalendar/core'
import type { AvailabilityPayload } from '../types'
export function generateAvailabilityPayload(
  events: EventInput[],
  teacherId: number
): AvailabilityPayload {
  if (teacherId <= 0) {
    throw new Error('teacherId must be positive')
  }
  const weeklyMap = new Map<string, { dayOfWeek: number; start: string; end: string }>()
  for (const event of events) {
    if (!event.start || !event.end) continue
    const startDate = event.start as Date
    const endDate = event.end as Date
    const dayOfWeek = startDate.getDay()
    const startTime = startDate.toTimeString().slice(0, 5)
    const endTime = endDate.toTimeString().slice(0, 5)
    weeklyMap.set(`${dayOfWeek}-${startTime}-${endTime}`, {
      dayOfWeek,
      start: startTime,
      end: endTime,
    })
  }
  return { teacherId, weekly: Array.from(weeklyMap.values()) }
}
