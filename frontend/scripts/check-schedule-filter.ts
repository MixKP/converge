// Runnable check for the pure schedule logic in src/utils/scheduleFilter.ts.
// Run: npm run check:schedule   (node --experimental-strip-types, no test framework)
import {
  filterBookings,
  groupByDay,
  studentOptions,
  teacherOptions,
} from '../src/utils/scheduleFilter.ts'
import type { Booking } from '../src/types/booking.ts'

let failures = 0

function expect(label: string, actual: unknown, expected: unknown) {
  const a = JSON.stringify(actual)
  const e = JSON.stringify(expected)
  if (a === e) {
    console.log(`  ok   ${label}`)
  } else {
    failures++
    console.error(`  FAIL ${label}\n         expected ${e}\n         actual   ${a}`)
  }
}

const NOW = Date.parse('2026-08-19T12:00:00Z')

const booking = (over: Partial<Booking> & Pick<Booking, 'id' | 'start_time'>): Booking => {
  const start = Date.parse(over.start_time)
  return {
    teacher_id: 1,
    teacher_name: 'Ada',
    branch_id: 1,
    branch_name: 'Central',
    subject_id: 1,
    subject_name: 'Math',
    student_id: 10,
    student_name: 'Kim',
    end_time: Number.isNaN(start) ? over.start_time : new Date(start + 3600_000).toISOString(),
    created_at: '2026-08-01T00:00:00Z',
    ...over,
  }
}

const pastAda = booking({ id: 1, start_time: '2026-08-18T09:00:00Z' })
const soonAda = booking({ id: 2, start_time: '2026-08-20T09:00:00Z' })
const laterAda = booking({ id: 3, start_time: '2026-08-21T09:00:00Z' })
const soonBob = booking({
  id: 4,
  start_time: '2026-08-20T11:00:00Z',
  teacher_id: 2,
  teacher_name: 'Bob',
  student_id: 11,
  student_name: 'Lee',
})
const all = [laterAda, pastAda, soonBob, soonAda]

const ids = (list: Booking[]) => list.map((b) => b.id)
const noFilter = { teacherId: null, studentId: null }

console.log('filterBookings — happy path')
expect(
  'upcoming excludes finished classes and sorts soonest-first',
  ids(filterBookings(all, { scope: 'upcoming', ...noFilter }, NOW)),
  [2, 4, 3],
)
expect(
  'past keeps only finished classes, most recent first',
  ids(filterBookings(all, { scope: 'past', ...noFilter }, NOW)),
  [1],
)
expect(
  'teacher filter narrows to that teacher',
  ids(filterBookings(all, { scope: 'upcoming', teacherId: 2, studentId: null }, NOW)),
  [4],
)
expect(
  'student filter narrows to that student',
  ids(filterBookings(all, { scope: 'upcoming', teacherId: null, studentId: 10 }, NOW)),
  [2, 3],
)
expect(
  'teacher and student filters combine',
  ids(filterBookings(all, { scope: 'upcoming', teacherId: 2, studentId: 10 }, NOW)),
  [],
)

console.log('filterBookings — boundary and malformed data')
// A class ending exactly now is still "upcoming", matching MyClassesView so
// the same booking never appears under both scopes across the two pages.
const endingNow = booking({
  id: 5,
  start_time: '2026-08-19T11:00:00Z',
  end_time: new Date(NOW).toISOString(),
})
expect(
  'a class ending exactly now is still upcoming',
  ids(filterBookings([endingNow], { scope: 'upcoming', ...noFilter }, NOW)),
  [5],
)
expect(
  'and is absent from past',
  ids(filterBookings([endingNow], { scope: 'past', ...noFilter }, NOW)),
  [],
)
// The critical failure mode: a malformed timestamp must not drop the row from
// the admin's view — it stays visible in upcoming and sorts last.
const broken = booking({ id: 6, start_time: 'not-a-date', end_time: 'not-a-date' })
expect(
  'an unparsable timestamp stays visible and sorts last',
  ids(filterBookings([...all, broken], { scope: 'upcoming', ...noFilter }, NOW)),
  [2, 4, 3, 6],
)
expect('empty input yields an empty list', filterBookings([], { scope: 'upcoming', ...noFilter }, NOW), [])

console.log('option lists')
expect('teachers are de-duplicated and sorted by name', teacherOptions(all), [
  { id: 1, name: 'Ada' },
  { id: 2, name: 'Bob' },
])
expect('students are de-duplicated and sorted by name', studentOptions(all), [
  { id: 10, name: 'Kim' },
  { id: 11, name: 'Lee' },
])
// teacher_name / student_name are omitempty server-side.
const nameless = booking({ id: 7, start_time: '2026-08-20T09:00:00Z', teacher_name: undefined })
expect('a missing teacher name falls back to the id', teacherOptions([nameless]), [
  { id: 1, name: 'Teacher #1' },
])

console.log('groupByDay')
const upcoming = filterBookings(all, { scope: 'upcoming', ...noFilter }, NOW)
expect(
  'consecutive same-day bookings share one group',
  groupByDay(upcoming).map((day) => ids(day.items)),
  [[2, 4], [3]],
)
expect('grouping an empty list yields no groups', groupByDay([]), [])

if (failures > 0) {
  console.error(`\n${failures} check(s) failed`)
  process.exit(1)
}
console.log('\nAll schedule filter checks passed')
