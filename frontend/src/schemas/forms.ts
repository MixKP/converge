import { z } from 'zod'
import { toMinutes } from '../utils/dateValidation'
import { selectId } from './common'
import { weeklySlotSchema, noPerDayOverlap } from './calendar'

const selectGender = () =>
  z
    .enum(['male', 'female', 'lgbtq+'])
    .nullable()
    .refine((v): v is 'male' | 'female' | 'lgbtq+' => v !== null, 'Please select a gender preference')

export const aiSuggestionsFormSchema = z.object({
  student_id: selectId('a student'),
  subject_id: selectId('a subject'),
  branch_id: selectId('a branch'),
  required_gender: selectGender(),
  teacher_id: z.number().int().positive().nullable(),
  slots: z
    .array(weeklySlotSchema)
    .min(1, 'Add at least one time slot')
    .superRefine((slots, ctx) => {
      const durations = new Set(slots.map((s) => toMinutes(s.end) - toMinutes(s.start)))
      if (durations.size > 1) {
        ctx.addIssue({ code: 'custom', message: 'All time slots must use the same duration' })
      }
    }),
})

export const manualBookingFormSchema = z.object({
  student_id: selectId('a student'),
  subject_id: selectId('a subject'),
  branch_id: selectId('a branch'),
  teacher_id: selectId('a teacher'),
  required_gender: z.enum(['male', 'female', 'lgbtq+']).nullable(),
})

export const availabilityFormSchema = z.object({
  teacher_id: selectId('a teacher'),
  weekly: z
    .array(weeklySlotSchema)
    .min(1, 'Add at least one time slot on the calendar')
    .superRefine(noPerDayOverlap),
})
