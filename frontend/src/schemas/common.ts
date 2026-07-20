import { z } from 'zod'

export const timeHHMM = z
  .string()
  .regex(/^([01]\d|2[0-3]):[0-5]\d$/, 'Use HH:MM format (e.g., 09:30)')

export const requiredId = (label: string) =>
  z
    .number({ error: `Please select ${label}` })
    .int()
    .positive(`Please select ${label}`)

    export const selectId = (label: string) =>
  z
    .number({ error: `Please select ${label}` })
    .int()
    .positive(`Please select ${label}`)
    .nullable()
    .refine((v): v is number => v !== null, `Please select ${label}`)

export const isoDateTime = z
  .string()
  .refine((s) => !Number.isNaN(new Date(s).getTime()), 'Invalid date format')

export const dateRangeSchema = z
  .object({ start_time: isoDateTime, end_time: isoDateTime })
  .refine((v) => new Date(v.start_time) < new Date(v.end_time), {
    message: 'Start time must be before end time',
    path: ['end_time'],
  })
