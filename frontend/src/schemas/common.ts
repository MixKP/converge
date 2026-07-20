import { z } from 'zod'

/** "HH:MM" 24-hour string, matching the backend TimeHHMM layout ("15:04"). */
export const timeHHMM = z
  .string()
  .regex(/^([01]\d|2[0-3]):[0-5]\d$/, 'Use HH:MM format (e.g., 09:30)')

/**
 * A required positive integer id sourced from a `<select>`.
 * Selects use `null` as their placeholder value; `null` fails the number
 * check and surfaces the custom "Please select ..." message as the field error.
 */
export const requiredId = (label: string) =>
  z
    .number({ error: `Please select ${label}` })
    .int()
    .positive(`Please select ${label}`)

/**
 * A required id bound to a `<select>` whose unselected placeholder value is
 * `null`. Accepts `null` as input (so form initial values / v-model type-check)
 * but rejects it with the "Please select ..." message during validation.
 */
export const selectId = (label: string) =>
  z
    .number({ error: `Please select ${label}` })
    .int()
    .positive(`Please select ${label}`)
    .nullable()
    .refine((v): v is number => v !== null, `Please select ${label}`)

/** RFC3339 / ISO datetime string as used by Booking start_time/end_time. */
export const isoDateTime = z
  .string()
  .refine((s) => !Number.isNaN(new Date(s).getTime()), 'Invalid date format')

/** Datetime range where end must come after start (mirrors the DB CHECK). */
export const dateRangeSchema = z
  .object({ start_time: isoDateTime, end_time: isoDateTime })
  .refine((v) => new Date(v.start_time) < new Date(v.end_time), {
    message: 'Start time must be before end time',
    path: ['end_time'],
  })
