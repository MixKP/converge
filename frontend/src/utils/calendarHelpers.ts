import type { EventInput } from '@fullcalendar/core'
import type { AvailabilityPayload } from '../types'

export function generateAvailabilityPayload(
  events: EventInput[],
  teacherId: number
): AvailabilityPayload {
  if (teacherId <= 0) {
    throw new Error('teacherId must be positive')
  }

  const weeklyMap = new Map<string, { day_of_week: number; start: string; end: string }>()
  for (const event of events) {
    const startDate = event.start as Date | undefined
    const endDate = event.end as Date | undefined
    if (!startDate || !endDate) continue

    const dayOfWeek = startDate.getDay()
    const startTime = startDate.toTimeString().slice(0, 5)
    const endTime = endDate.toTimeString().slice(0, 5)

    weeklyMap.set(`${dayOfWeek}|${startTime}|${endTime}`, {
      day_of_week: dayOfWeek,
      start: startTime,
      end: endTime,
    })
  }
  const result = Array.from(weeklyMap.values())
  return { teacher_id: teacherId, weekly: result }
}

export function formatEventSlot(event: EventInput): { day: string; start: string; end: string } {
  const start = new Date(event.start as string)
  const end = new Date(event.end as string)
  const dayName = start.toLocaleDateString('en-US', { weekday: 'long' })
  const startTime = start.toLocaleTimeString('en-US', {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  })
  const endTime = end.toLocaleTimeString('en-US', {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  })
  return { day: dayName, start: startTime, end: endTime }
}
