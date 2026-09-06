import assert from 'node:assert/strict'
import { fullCapacitySpans, overlapsFullCapacity } from '../src/utils/branchCapacity.ts'

const bookings = [
  { branch_id: 1, start_time: '2026-06-01T09:00:00Z', end_time: '2026-06-01T11:00:00Z' },
  { branch_id: 1, start_time: '2026-06-01T10:00:00Z', end_time: '2026-06-01T12:00:00Z' },
  { branch_id: 2, start_time: '2026-06-01T10:00:00Z', end_time: '2026-06-01T12:00:00Z' },
]

assert.deepEqual(fullCapacitySpans(1, -1, bookings), [])

const full = fullCapacitySpans(1, 2, bookings)
assert.deepEqual(full, [
  {
    start: Date.parse('2026-06-01T10:00:00Z'),
    end: Date.parse('2026-06-01T11:00:00Z'),
    occupancy: 2,
  },
])
assert.equal(overlapsFullCapacity(full, '2026-06-01T10:30:00Z', '2026-06-01T11:30:00Z'), true)
assert.equal(overlapsFullCapacity(full, '2026-06-01T11:00:00Z', '2026-06-01T12:00:00Z'), false)

console.log('branch capacity checks passed')
