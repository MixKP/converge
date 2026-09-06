import assert from 'node:assert/strict'
import { commuteSpansForBranch } from '../src/utils/commuteAvailability.ts'

const hour = (value: number) => Date.parse(`2026-06-01T${String(value).padStart(2, '0')}:00:00Z`)

const engagement = [{ branchId: 1, start: hour(10), end: hour(11) }]

assert.deepEqual(commuteSpansForBranch(engagement, 2, 30), [
  { start: hour(9) + 30 * 60 * 1000, end: hour(10) },
  { start: hour(11), end: hour(11) + 30 * 60 * 1000 },
])
assert.deepEqual(commuteSpansForBranch(engagement, 1, 30), [])
assert.deepEqual(commuteSpansForBranch(engagement, 2, 0), [])

console.log('commute availability checks passed')
