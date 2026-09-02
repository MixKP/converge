import type { Booking } from '../types'

export type ScheduleScope = 'upcoming' | 'past'

export type ScheduleFilters = {
  scope: ScheduleScope
  teacherId: number | null
  studentId: number | null
}

export type PersonOption = { id: number; name: string }

export type ScheduleDay = { key: string; items: Booking[] }

function timeOf(iso: string): number {
  const ms = new Date(iso).getTime()
  return Number.isNaN(ms) ? Number.POSITIVE_INFINITY : ms
}

function people(bookings: Booking[], pick: (booking: Booking) => [number, string]): PersonOption[] {
  const byId = new Map<number, string>()
  for (const booking of bookings) {
    const [id, name] = pick(booking)
    if (!byId.has(id)) byId.set(id, name)
  }
  return [...byId].map(([id, name]) => ({ id, name })).sort((a, b) => a.name.localeCompare(b.name))
}

export function teacherOptions(bookings: Booking[]): PersonOption[] {
  return people(bookings, (b) => [b.teacher_id, b.teacher_name || `Teacher #${b.teacher_id}`])
}

export function studentOptions(bookings: Booking[]): PersonOption[] {
  return people(bookings, (b) => [b.student_id, b.student_name || `Student #${b.student_id}`])
}

export function filterBookings(
  bookings: Booking[],
  filters: ScheduleFilters,
  now: number = Date.now()
): Booking[] {
  const matched = bookings.filter((b) => {
    const isPast = timeOf(b.end_time) < now
    if (filters.scope === 'past' ? !isPast : isPast) return false
    if (filters.teacherId !== null && b.teacher_id !== filters.teacherId) return false
    if (filters.studentId !== null && b.student_id !== filters.studentId) return false
    return true
  })

  return matched.sort((a, b) => {
    const delta = timeOf(a.start_time) - timeOf(b.start_time)
    if (Number.isNaN(delta)) return 0
    return filters.scope === 'past' ? -delta : delta
  })
}

export function groupByDay(bookings: Booking[]): ScheduleDay[] {
  const out: ScheduleDay[] = []
  for (const booking of bookings) {
    const key = new Date(booking.start_time).toDateString()
    const last = out.at(-1)
    if (last?.key === key) last.items.push(booking)
    else out.push({ key, items: [booking] })
  }
  return out
}
