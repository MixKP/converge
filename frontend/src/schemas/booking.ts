import { z } from 'zod'
import { weeklySlotSchema } from './calendar'
import { isoDateTime } from './common'

export const branchSchema = z.object({
  id: z.number(),
  name: z.string(),
})

export const bookingRequestSchema = z.object({
  subject_id: z.number().int().positive(),
  branch_id: z.number().int().positive(),
  preferred_slots: z.array(weeklySlotSchema).min(1),
  duration_minutes: z.number().int().positive().optional(),
  // preferred_teacher_id is *int in Go: optional & nullable.
  preferred_teacher_id: z.number().int().positive().nullish(),
})

export const confirmBookingRequestSchema = z
  .object({
    teacher_id: z.number().int().positive(),
    branch_id: z.number().int().positive(),
    subject_id: z.number().int().positive(),
    start_time: isoDateTime,
    end_time: isoDateTime,
    client_name: z.string(),
  })
  .refine((v) => new Date(v.start_time) < new Date(v.end_time), {
    message: 'Start time must be before end time',
    path: ['end_time'],
  })

export const bookingAlternativeSchema = z.object({
  teacher_id: z.number(),
  teacher_name: z.string(),
  branch_id: z.number(),
  subject_id: z.number(),
  start_time: z.string(),
  end_time: z.string(),
  score: z.number(),
  reasons: z.array(z.string()),
  room_available: z.boolean().optional(),
  commute_minutes: z.number().optional(),
})

const slotResultSchema = z.object({
  slot: weeklySlotSchema,
  exact_match: bookingAlternativeSchema.optional(),
  alternatives: z.array(bookingAlternativeSchema).optional(),
  message: z.string(),
})

export const bookingResponseSchema = z.object({
  results: z.array(slotResultSchema),
})

export const bookingSchema = z.object({
  id: z.number(),
  teacher_id: z.number(),
  branch_id: z.number(),
  subject_id: z.number(),
  start_time: z.string(),
  end_time: z.string(),
  client_name: z.string(),
  created_at: z.string(),
})

export const cartItemSchema = z.object({
  id: z.number(),
  teacher_id: z.number(),
  teacher_name: z.string(),
  branch_id: z.number(),
  branch_name: z.string(),
  subject_id: z.number(),
  subject_name: z.string(),
  start_time: z.string(),
  end_time: z.string(),
  client_name: z.string(),
  status: z.enum(['pending', 'confirmed']),
})

export const cartItemInputSchema = cartItemSchema.omit({ id: true, status: true })
