import { toConfirmRequest } from '../src/utils/bookingPayload.ts'
import type { CartItem } from '../src/types/booking.ts'

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

const item: CartItem = {
  id: 1,
  teacher_id: 7,
  teacher_name: 'Somchai',
  branch_id: 2,
  branch_name: 'Silom',
  subject_id: 3,
  subject_name: 'Math',
  start_time: '2026-06-01T09:00:00.000Z',
  end_time: '2026-06-01T10:00:00.000Z',
  student_id: 42,
  student_name: 'Nara',
  status: 'pending',
}

console.log('toConfirmRequest — happy path')
expect('sends exactly the fields the backend binds', toConfirmRequest(item), {
  teacher_id: 7,
  branch_id: 2,
  subject_id: 3,
  start_time: '2026-06-01T09:00:00.000Z',
  end_time: '2026-06-01T10:00:00.000Z',
  student_id: 42,
})

console.log('toConfirmRequest — cart item written before student ownership')
expect(
  'refuses an item with no student',
  toConfirmRequest({ ...item, student_id: undefined as unknown as number }),
  null
)
expect('refuses a zero student id', toConfirmRequest({ ...item, student_id: 0 }), null)

console.log(failures === 0 ? '\nall checks passed' : `\n${failures} check(s) failed`)
process.exit(failures === 0 ? 0 : 1)
