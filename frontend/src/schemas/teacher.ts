import { z } from 'zod'

const teacherStatusSchema = z.enum(['active', 'deactivated'])
const genderSchema = z.enum(['male', 'female', 'lgbtq+'])

export const teacherSchema = z.object({
  id: z.number(),
  name: z.string(),
  email: z.string(),
  status: teacherStatusSchema,
  gender: genderSchema,
})
