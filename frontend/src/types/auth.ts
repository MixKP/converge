import type { z } from 'zod'
import type {
  authSessionSchema,
  parentWithStudentsSchema,
  registerRequestSchema,
  roleSchema,
  userSchema,
} from '../schemas/auth'

export type Role = z.infer<typeof roleSchema>
export type AuthUser = z.infer<typeof userSchema>
export type AuthSession = z.infer<typeof authSessionSchema>
export type ParentWithStudents = z.infer<typeof parentWithStudentsSchema>
export type RegisterRequest = z.infer<typeof registerRequestSchema>
