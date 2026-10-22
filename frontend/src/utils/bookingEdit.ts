import type { EventInput } from '@fullcalendar/vue3'
import type { Booking } from '../types'

// Every teacher, subject and branch: the edit list starts from the student,
// not from the subject and branch picked for a new booking.
export function upcomingForStudent(bookings: Booking[], studentId: number, now: number): Booking[] {
  return bookings
    .filter(
      (booking) => booking.student_id === studentId && new Date(booking.start_time).getTime() > now
    )
    .sort((a, b) => new Date(a.start_time).getTime() - new Date(b.start_time).getTime())
}

export interface EditBlocks {
  ownClasses: Booking[]
  teacherBusy: Booking[]
}

// What sits on the grid around the class being moved: the student's other
// upcoming classes (clickable, and in the way, since the student cannot be in
// two places), and the rest of that teacher's classes that are not over yet. A
// booking that is both is drawn once, as the student's. A class in progress
// stays: it still holds the teacher until it ends.
export function editBlocks(
  bookings: Booking[],
  studentId: number,
  selected: Booking | null,
  now: number
): EditBlocks {
  const ownClasses = upcomingForStudent(bookings, studentId, now).filter(
    (booking) => booking.id !== selected?.id
  )
  if (!selected) return { ownClasses, teacherBusy: [] }
  const ownIds = new Set(ownClasses.map((booking) => booking.id))
  const teacherBusy = bookings.filter(
    (booking) =>
      booking.teacher_id === selected.teacher_id &&
      booking.id !== selected.id &&
      !ownIds.has(booking.id) &&
      new Date(booking.end_time).getTime() > now
  )
  return { ownClasses, teacherBusy }
}

// The calendar edits a list, but a reschedule moves one booking. A slot the
// admin drew or clicked is read as "move the class to start here", keeping its
// length: a click always draws one hour, and a 90-minute class should not
// shrink because it was moved. Changing the length is the resize handle's job,
// which edits the class itself. With nothing left (the class was removed on the
// grid) there is no move and the caller decides what that means.
export function foldToSingleEvent(
  events: EventInput[],
  current: EventInput & { id: string }
): EventInput | null {
  const drawn = events.filter((event) => event.id !== current.id)
  if (drawn.length > 0) {
    const start = new Date(drawn[drawn.length - 1].start as string | Date)
    const length =
      new Date(current.end as string | Date).getTime() -
      new Date(current.start as string | Date).getTime()
    return { ...current, start, end: new Date(start.getTime() + length) }
  }
  return events.find((event) => event.id === current.id) ?? null
}

export function sameTimes(event: EventInput, booking: Booking): boolean {
  const start = new Date(event.start as string | Date).getTime()
  const end = new Date(event.end as string | Date).getTime()
  return (
    start === new Date(booking.start_time).getTime() && end === new Date(booking.end_time).getTime()
  )
}
