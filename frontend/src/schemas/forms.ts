import { z } from 'zod'
import { selectId } from './common'
import { weeklySlotSchema, noPerDayOverlap } from './calendar'

export const availabilityFormSchema = z.object({
  teacher_id: selectId('a teacher'),
  weekly: z
    .array(weeklySlotSchema)
    .min(1, 'Add at least one time slot on the calendar')
    .superRefine(noPerDayOverlap),
})
