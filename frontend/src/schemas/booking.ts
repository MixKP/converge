import { z } from 'zod'
import { weeklySlotSchema } from './calendar'
import { isoDateTime } from './common'

const genderSchema = z.enum(['male', 'female', 'lgbtq+'])

export const branchSchema = z.object({
  id: z.number(),
  name: z.string(),
  capacity: z.number().int().min(0),
})

export const bookingRequestSchema = z.object({
  subject_id: z.number().int().positive(),
  branch_id: z.number().int().positive(),
  preferred_slots: z.array(weeklySlotSchema).min(1),
  duration_minutes: z.number().int().positive().optional(),
  required_gender: genderSchema,
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
    student_id: z.number().int().positive(),
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

// The *_name fields are omitempty in Go: the list endpoint joins them in, but
// the confirm response returns the booking without them.
export const bookingSchema = z.object({
  id: z.number(),
  teacher_id: z.number(),
  teacher_name: z.string().optional(),
  branch_id: z.number(),
  branch_name: z.string().optional(),
  subject_id: z.number(),
  subject_name: z.string().optional(),
  start_time: z.string(),
  end_time: z.string(),
  student_id: z.number(),
  student_name: z.string(),
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
  // The booking is made on behalf of this student; the backend attributes and
  // scopes it by student_id, so a cart item is meaningless without one.
  student_id: z.number().int().positive(),
  student_name: z.string(),
  status: z.enum(['pending', 'confirmed']),
})

export const cartItemInputSchema = cartItemSchema.omit({ id: true, status: true })
