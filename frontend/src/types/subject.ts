import type { z } from 'zod'
import type { subjectSchema } from '../schemas/subject'

export type Subject = z.infer<typeof subjectSchema>
