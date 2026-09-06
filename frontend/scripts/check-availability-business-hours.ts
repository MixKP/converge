import assert from 'node:assert/strict'
import { businessHoursForTeachers } from '../src/utils/availabilityTransform.ts'

const availability = new Map([
  [1, [{ day_of_week: 1, start: '09:00', end: '11:00' }]],
  [2, [{ day_of_week: 3, start: '13:00', end: '15:00' }]],
])

assert.deepEqual(businessHoursForTeachers(availability, [2, 1]), [
  { daysOfWeek: [3], startTime: '13:00', endTime: '15:00' },
  { daysOfWeek: [1], startTime: '09:00', endTime: '11:00' },
])
assert.deepEqual(businessHoursForTeachers(availability, [99]), [])

console.log('availability business-hours checks passed')
