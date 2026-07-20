import type { z } from 'zod'
import type { teacherSchema } from '../schemas/teacher'

export type Teacher = z.infer<typeof teacherSchema>
