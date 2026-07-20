import { z } from 'zod'

const teacherStatusSchema = z.enum(['active', 'deactivated'])

export const teacherSchema = z.object({
  id: z.number(),
  name: z.string(),
  email: z.string(),
  status: teacherStatusSchema,
})
