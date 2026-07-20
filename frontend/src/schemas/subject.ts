import { z } from 'zod'

/** Mirrors backend shared.Subject. */
export const subjectSchema = z.object({
  id: z.number(),
  name: z.string(),
})
