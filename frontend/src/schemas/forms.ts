import { z } from 'zod'
import { toMinutes } from '../utils/dateValidation'
import { selectId } from './common'
import { weeklySlotSchema, noPerDayOverlap } from './calendar'

/**
 * AI-suggestions form (SmartSuggestionsPanel). Ports the validation previously
 * living in useAISuggestions.ts (missing selection, per-slot time format,
 * start<end, and the "all slots share one duration" rule).
 */
export const aiSuggestionsFormSchema = z.object({
  subject_id: selectId('a subject'),
  branch_id: selectId('a branch'),
  // Teacher is optional here; null means "any teacher".
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

export type AiSuggestionsForm = z.infer<typeof aiSuggestionsFormSchema>

/** Manual booking selection (BookingView): all three ids required. */
export const manualBookingFormSchema = z.object({
  subject_id: selectId('a subject'),
  branch_id: selectId('a branch'),
  teacher_id: selectId('a teacher'),
})

export type ManualBookingForm = z.infer<typeof manualBookingFormSchema>

/** Teacher availability submission (SubmitAvailability). */
export const availabilityFormSchema = z.object({
  teacher_id: selectId('a teacher'),
  weekly: z
    .array(weeklySlotSchema)
    .min(1, 'Add at least one time slot on the calendar')
    .superRefine(noPerDayOverlap),
})

export type AvailabilityForm = z.infer<typeof availabilityFormSchema>
