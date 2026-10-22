import { sortDeactivatedLast } from './status.ts'
import { startOfWeek } from './calendarHelpers.ts'
import type { Booking, Teacher } from '../types'

export type RosterMode = 'teachers' | 'students'

export interface RosterSession {
  id: number
  startTime: string
  endTime: string
  subject: string
  branch: string
}

// The other side of the roster: the students a teacher teaches, or the teachers
// a student learns from. Bookings are grouped into one entry per person so the
// roster answers "who does this teacher teach" instead of listing every session.
export interface RosterPerson {
  id: number
  name: string
  subjects: string[]
  branches: string[]
  sessions: RosterSession[]
}

// Only the teacher side has a status; students carry none, so the field is
// optional rather than a third "n/a" state the view would have to filter out.
export type RosterStatus = Teacher['status']

export interface RosterEntry {
  id: number
  name: string
  subjects: string[]
  people: RosterPerson[]
  classCount: number
  status?: RosterStatus
}

export const timeOf = (iso: string): number => {
  const ms = new Date(iso).getTime()
  return Number.isNaN(ms) ? Number.POSITIVE_INFINITY : ms
}

export const orFallback = (value: string | undefined, label: string, id: number): string =>
  value || `${label} #${id}`

const pushUnique = (list: string[], value: string): void => {
  if (!list.includes(value)) list.push(value)
}

const byName = (a: { name: string }, b: { name: string }): number => a.name.localeCompare(b.name)

const counterpartOf = (booking: Booking, mode: RosterMode): { id: number; name: string } =>
  mode === 'teachers'
    ? {
        id: booking.student_id,
        name: orFallback(booking.student_name, 'Student', booking.student_id),
      }
    : {
        id: booking.teacher_id,
        name: orFallback(booking.teacher_name, 'Teacher', booking.teacher_id),
      }

export function groupByCounterpart(bookings: Booking[], mode: RosterMode): RosterPerson[] {
  const map = new Map<number, RosterPerson>()

  for (const booking of bookings) {
    const { id, name } = counterpartOf(booking, mode)
    const subject = orFallback(booking.subject_name, 'Subject', booking.subject_id)
    const branch = orFallback(booking.branch_name, 'Branch', booking.branch_id)

    let person = map.get(id)
    if (!person) {
      person = { id, name, subjects: [], branches: [], sessions: [] }
      map.set(id, person)
    }
    pushUnique(person.subjects, subject)
    pushUnique(person.branches, branch)
    person.sessions.push({
      id: booking.id,
      startTime: booking.start_time,
      endTime: booking.end_time,
      subject,
      branch,
    })
  }

  for (const person of map.values()) {
    person.sessions.sort((a, b) => timeOf(a.startTime) - timeOf(b.startTime))
    person.subjects.sort((a, b) => a.localeCompare(b))
    person.branches.sort((a, b) => a.localeCompare(b))
  }

  return [...map.values()].sort(byName)
}

// A student has no assigned-subject list of their own, so their subjects are the
// distinct subjects they are actually booked for.
export function subjectsFromBookings(bookings: Booking[]): string[] {
  const names: string[] = []
  for (const booking of bookings) {
    pushUnique(names, orFallback(booking.subject_name, 'Subject', booking.subject_id))
  }
  return names.sort((a, b) => a.localeCompare(b))
}

export interface BuildEntryInput {
  id: number
  name: string
  subjects: string[]
  bookings: Booking[]
  mode: RosterMode
  status?: RosterStatus
}

export function buildEntry({
  id,
  name,
  subjects,
  bookings,
  mode,
  status,
}: BuildEntryInput): RosterEntry {
  const people = groupByCounterpart(bookings, mode)
  return {
    id,
    name,
    subjects,
    people,
    classCount: bookings.length,
    status,
  }
}

export function bookingsInWeekOf(bookings: Booking[], date: Date): Booking[] {
  const start = startOfWeek(date)
  const end = new Date(start)
  end.setDate(end.getDate() + 7)
  return bookings.filter((booking) => {
    const time = new Date(booking.start_time).getTime()
    return time >= start.getTime() && time < end.getTime()
  })
}

export function groupBookingsBy(
  bookings: Booking[],
  key: (booking: Booking) => number
): Map<number, Booking[]> {
  const map = new Map<number, Booking[]>()
  for (const booking of bookings) {
    const id = key(booking)
    const existing = map.get(id)
    if (existing) existing.push(booking)
    else map.set(id, [booking])
  }
  return map
}

// Searching the counterpart names too is what lets the page answer the reverse
// question ("which teacher teaches Bob?") without a second view.
export function matchesSearch(entry: RosterEntry, term: string): boolean {
  const needle = term.trim().toLowerCase()
  if (!needle) return true
  const hit = (value: string) => value.toLowerCase().includes(needle)
  return (
    hit(entry.name) || entry.subjects.some(hit) || entry.people.some((person) => hit(person.name))
  )
}

export const sortEntriesForDisplay = (entries: RosterEntry[]): RosterEntry[] =>
  sortDeactivatedLast(entries)
