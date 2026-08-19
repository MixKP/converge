import type { Booking } from '../types'

export type ScheduleScope = 'upcoming' | 'past'

export type ScheduleFilters = {
  scope: ScheduleScope
  teacherId: number | null
  studentId: number | null
}

export type PersonOption = { id: number; name: string }

export type ScheduleDay = { key: string; items: Booking[] }

/**
 * An unparsable timestamp sorts to the end and never counts as past, so a
 * malformed booking stays visible to the admin instead of silently vanishing.
 */
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

// The name fields are omitempty server-side, so every option falls back to the
// id rather than rendering a blank entry that cannot be told apart from another.
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

  // Upcoming reads best soonest-first; past reads best most-recent-first.
  return matched.sort((a, b) => {
    const delta = timeOf(a.start_time) - timeOf(b.start_time)
    if (Number.isNaN(delta)) return 0
    return filters.scope === 'past' ? -delta : delta
  })
}

/** Groups an already-sorted list into consecutive calendar days. */
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
