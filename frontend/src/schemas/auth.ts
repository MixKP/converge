import { z } from 'zod'

// Mirrors shared.Role on the backend. 'guest' exists in the Go enum but is never
// issued by /login, so it is deliberately absent here.
export const roleSchema = z.enum(['admin', 'student', 'parent'])

export const userSchema = z.object({
  id: z.number(),
  name: z.string(),
  role: roleSchema,
})

export const parentWithStudentsSchema = z.object({
  id: z.number(),
  name: z.string(),
  role: roleSchema,
  students: z
    .array(userSchema)
    .nullable()
    .transform((v) => v ?? []),
})

export const authResponseSchema = z.object({
  token: z.string().min(1),
  user: userSchema,
})

// Persisted shape. Kept separate from authResponseSchema so a future change to
// the wire format does not silently invalidate every stored session.
export const authSessionSchema = z.object({
  token: z.string().min(1),
  user: userSchema,
})

// Backend limits: names are capped at 100 runes, and bcrypt silently truncates
// anything past 72 bytes, so the password limit is measured in bytes not chars.
const MAX_NAME_CHARS = 100
const MAX_PASSWORD_BYTES = 72

const byteLength = (value: string) => new TextEncoder().encode(value).length

export const nameField = z
  .string()
  .trim()
  .min(1, 'Username is required')
  .refine(
    (v) => [...v].length <= MAX_NAME_CHARS,
    `Username must be at most ${MAX_NAME_CHARS} characters`
  )

export const passwordField = z
  .string()
  .min(1, 'Password is required')
  .refine(
    (v) => byteLength(v) <= MAX_PASSWORD_BYTES,
    `Password must be at most ${MAX_PASSWORD_BYTES} bytes`
  )

export const loginRequestSchema = z.object({
  name: nameField,
  password: passwordField,
})

export const registerRequestSchema = z
  .object({
    name: nameField,
    password: passwordField,
    role: roleSchema,
    student_ids: z.array(z.number().int().positive()).optional(),
  })
  .refine((v) => v.role !== 'parent' || (v.student_ids?.length ?? 0) > 0, {
    message: 'A parent must be linked to at least one student',
    path: ['student_ids'],
  })
