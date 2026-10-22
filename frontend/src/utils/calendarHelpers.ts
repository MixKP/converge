import type { EventInput } from '@fullcalendar/vue3'
import type { AvailabilityPayload } from '../types'

export function startOfWeek(date: Date): Date {
  const start = new Date(date)
  start.setHours(0, 0, 0, 0)
  start.setDate(start.getDate() - start.getDay())
  return start
}

function generateEventId(): string {
  return `event-${Date.now()}-${Math.random().toString(36).slice(2, 11)}`
}

export function isSameDaySelection(start: Date, end: Date): boolean {
  const startDay = start.getDate()
  const endDay = end.getDate()
  const startMonth = start.getMonth()
  const endMonth = end.getMonth()
  return startDay === endDay && startMonth === endMonth
}

export function isSameDay(date1: Date, date2: Date): boolean {
  return (
    date1.getFullYear() === date2.getFullYear() &&
    date1.getMonth() === date2.getMonth() &&
    date1.getDate() === date2.getDate()
  )
}

export function createEvent(start: Date, end: Date): EventInput {
  return {
    id: generateEventId(),
    start,
    end,
    title: '',
    // Set here rather than left to the theme default: an event with no colour
    // falls back to --fc-classic-event, which is a solid fill meant to carry
    // white text. Every other block on this grid is a tint with an edge.
    color: 'var(--draft-fill)',
    contrastColor: 'var(--text-primary)',
    className: 'draft-event',
  }
}

export function createOneHourEvent(date: Date): EventInput {
  const start = new Date(date)
  const end = new Date(start.getTime() + 60 * 60 * 1000)
  return createEvent(start, end)
}

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
