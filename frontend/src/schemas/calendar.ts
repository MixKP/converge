import { z } from 'zod'
import { toMinutes } from '../utils/dateValidation'
import { timeHHMM, requiredId } from './common'

export const weeklySlotSchema = z
  .object({
    day_of_week: z.coerce.number().int().min(0).max(6),
    start: timeHHMM,
    end: timeHHMM,
  })
  .refine((s) => toMinutes(s.start) < toMinutes(s.end), {
    message: 'Start time must be before end time',
    path: ['end'],
  })

export type WeeklySlotInput = z.infer<typeof weeklySlotSchema>

export const noPerDayOverlap = (slots: WeeklySlotInput[], ctx: z.RefinementCtx) => {
  for (let i = 0; i < slots.length; i++) {
    for (let j = i + 1; j < slots.length; j++) {
      if (slots[i].day_of_week !== slots[j].day_of_week) continue
      const overlaps =
        toMinutes(slots[i].start) < toMinutes(slots[j].end) &&
        toMinutes(slots[i].end) > toMinutes(slots[j].start)
      if (overlaps) {
        ctx.addIssue({
          code: 'custom',
          message: 'Time slots on the same day must not overlap',
          path: [j],
        })
      }
    }
  }
}

export const availabilityPayloadSchema = z.object({
  teacher_id: requiredId('a teacher'),
  weekly: z.array(weeklySlotSchema).min(1, 'Add at least one time slot').superRefine(noPerDayOverlap),
})
