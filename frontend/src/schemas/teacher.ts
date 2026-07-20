import { z } from 'zod'

/** Teacher status DB CHECK constraint (teachers.status). */
export const teacherStatusSchema = z.enum(['active', 'deactivated'])

/**
 * Mirrors backend teacher.Teacher. `email` marshals as an empty string when
 * null on the backend, so it stays a plain string here. Teachers are only ever
 * API-sourced in the frontend, so requiring email/status is safe.
 */
export const teacherSchema = z.object({
  id: z.number(),
  name: z.string(),
  email: z.string(),
  status: teacherStatusSchema,
})
