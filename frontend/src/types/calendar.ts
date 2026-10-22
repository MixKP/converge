import type { z } from 'zod'
import type { weeklySlotSchema, availabilityPayloadSchema } from '../schemas/calendar'

export type WeeklySlot = z.infer<typeof weeklySlotSchema>
export type AvailabilityPayload = z.infer<typeof availabilityPayloadSchema>

export interface SubmitAvailabilityResponse {
  message: string
  deleted_bookings: number
}
