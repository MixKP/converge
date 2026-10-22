// Admin roster grouping guard.
//
// The dashboard answers "which students does this teacher teach" by grouping
// bookings per counterpart. The failure that matters is a student appearing once
// per session instead of once per person, which is what the page showed before.
//
// Run: npm run check:roster

import assert from 'node:assert/strict'
import {
  bookingsInWeekOf,
  buildEntry,
  groupByCounterpart,
  matchesSearch,
  sortEntriesForDisplay,
  subjectsFromBookings,
} from '../src/utils/roster.ts'
import type { Booking } from '../src/types/booking.ts'
import type { RosterMode } from '../src/utils/roster.ts'

const booking = (over: Partial<Booking>): Booking =>
  ({
    id: 1,
    teacher_id: 10,
    teacher_name: 'Alice',
    branch_id: 1,
    branch_name: 'Siam',
    subject_id: 5,
    subject_name: 'Math',
    start_time: '2026-10-05T09:00:00Z',
    end_time: '2026-10-05T10:00:00Z',
    student_id: 100,
    student_name: 'Bob',
    created_at: '2026-09-01T00:00:00Z',
    ...over,
  }) as Booking

// --- happy path: repeat sessions collapse into one person ------------------
{
  const people = groupByCounterpart(
    [
      booking({ id: 1, start_time: '2026-10-07T09:00:00Z', end_time: '2026-10-07T10:00:00Z' }),
      booking({ id: 2, start_time: '2026-10-05T09:00:00Z', end_time: '2026-10-05T10:00:00Z' }),
      booking({ id: 3, student_id: 200, student_name: 'Carol', subject_name: 'Physics' }),
    ],
    'teachers'
  )

  assert.equal(people.length, 2, 'two distinct students, not three sessions')
  const bob = people.find((p) => p.name === 'Bob')!
  assert.equal(bob.sessions.length, 2, 'Bob keeps both sessions')
  assert.deepEqual(
    bob.sessions.map((s) => s.id),
    [2, 1],
    'sessions sorted chronologically, not insertion order'
  )
  assert.deepEqual(bob.subjects, ['Math'], 'repeated subject is not duplicated')
  assert.deepEqual(
    people.map((p) => p.name),
    ['Bob', 'Carol'],
    'people sorted by name'
  )
}

// --- mirror: students mode groups by teacher, subjects come from bookings --
{
  const rows = [
    booking({ id: 1, teacher_id: 10, teacher_name: 'Alice', subject_name: 'Math' }),
    booking({ id: 2, teacher_id: 11, teacher_name: 'Zoe', subject_name: 'Physics' }),
  ]
  const people = groupByCounterpart(rows, 'students' satisfies RosterMode)

  assert.deepEqual(
    people.map((p) => p.name),
    ['Alice', 'Zoe'],
    'students mode groups by teacher'
  )
  assert.deepEqual(
    subjectsFromBookings(rows),
    ['Math', 'Physics'],
    'a student’s subjects are derived from their bookings'
  )
}

// --- edge: missing names fall back instead of rendering blank --------------
{
  const [person] = groupByCounterpart(
    [booking({ student_name: undefined, subject_name: undefined, branch_name: undefined })],
    'teachers'
  )
  assert.equal(person.name, 'Student #100')
  assert.deepEqual(person.subjects, ['Subject #5'])
  assert.deepEqual(person.branches, ['Branch #1'])
}

// --- edge: a teacher with no bookings is still an entry --------------------
{
  const entry = buildEntry({
    id: 10,
    name: 'Alice',
    subjects: ['Math'],
    bookings: [],
    mode: 'teachers',
    status: 'active',
  })
  assert.equal(entry.classCount, 0)
  assert.deepEqual(entry.people, [])
  assert.deepEqual(entry.subjects, ['Math'], 'assigned subjects survive with zero bookings')
}

// --- status rides along for teachers and stays absent for students ---------
{
  const deactivated = buildEntry({
    id: 11,
    name: 'Bob',
    subjects: [],
    bookings: [booking({})],
    mode: 'teachers',
    status: 'deactivated',
  })
  assert.equal(deactivated.status, 'deactivated', 'the dot needs the status on the entry')
  assert.equal(deactivated.classCount, 1, 'a deactivated teacher keeps their existing bookings')

  const student = buildEntry({
    id: 100,
    name: 'Bob',
    subjects: [],
    bookings: [],
    mode: 'students',
  })
  assert.equal(student.status, undefined, 'students have no status, so no dot is rendered')
}

// --- search reaches the counterpart, which is the reverse lookup -----------
{
  const entry = buildEntry({
    id: 10,
    name: 'Alice',
    subjects: ['Math'],
    bookings: [booking({})],
    mode: 'teachers',
  })
  assert.equal(matchesSearch(entry, ''), true, 'empty term matches everything')
  assert.equal(matchesSearch(entry, 'ali'), true, 'matches entry name')
  assert.equal(matchesSearch(entry, 'MATH'), true, 'match is case-insensitive')
  assert.equal(matchesSearch(entry, 'bob'), true, 'matches a student of this teacher')
  assert.equal(matchesSearch(entry, 'zzz'), false)
}

// --- deactivated teachers sink below active ones, alphabetical within ------
{
  const entry = (name: string, status?: 'active' | 'deactivated') =>
    buildEntry({ id: name.length, name, subjects: [], bookings: [], mode: 'teachers', status })

  const sorted = sortEntriesForDisplay([
    entry('Zoe', 'active'),
    entry('Bob', 'deactivated'),
    entry('Alice', 'deactivated'),
    entry('Carol', 'active'),
  ])
  assert.deepEqual(
    sorted.map((e) => e.name),
    ['Carol', 'Zoe', 'Alice', 'Bob'],
    'active first (A-Z), then deactivated (A-Z)'
  )

  // Students have no status, so the sort must collapse to plain alphabetical.
  const students = sortEntriesForDisplay([
    buildEntry({ id: 2, name: 'Zed', subjects: [], bookings: [], mode: 'students' }),
    buildEntry({ id: 1, name: 'Ann', subjects: [], bookings: [], mode: 'students' }),
  ])
  assert.deepEqual(
    students.map((e) => e.name),
    ['Ann', 'Zed'],
    'statusless entries sort alphabetically'
  )
}

// --- the dashboard shows the current Sunday–Saturday week only -------------
{
  const at = (day: number, hour: number, minute = 0) => new Date(2026, 9, day, hour, minute)
  const row = (id: number, start: Date) =>
    booking({ id, start_time: start.toISOString(), end_time: start.toISOString() })
  const rows = [
    row(1, at(17, 23, 59)),
    row(2, at(18, 0)),
    row(3, at(22, 9)),
    row(4, at(24, 23)),
    row(5, at(25, 0)),
    booking({ id: 6, start_time: 'not-a-date' }),
  ]

  assert.deepEqual(
    bookingsInWeekOf(rows, at(22, 15)).map((b) => b.id),
    [2, 3, 4],
    'Sunday 00:00 is in, the next Sunday 00:00 and last Saturday are out'
  )
  assert.deepEqual(
    bookingsInWeekOf(rows, at(18, 0)).map((b) => b.id),
    [2, 3, 4],
    'on Sunday at midnight the week has already turned over'
  )
  assert.deepEqual(
    bookingsInWeekOf(rows, at(17, 12)).map((b) => b.id),
    [1],
    'the Saturday before belongs to the previous week'
  )
}

console.log('✓ admin roster grouping behaves')
