import type { z } from 'zod'
import type {
  cartItemSchema,
  bookingRequestSchema,
  bookingAlternativeSchema,
  slotResultSchema,
  bookingResponseSchema,
  confirmBookingRequestSchema,
  bookingSchema,
  branchSchema,
} from '../schemas/booking'

export type CartItem = z.infer<typeof cartItemSchema>
export type BookingRequest = z.infer<typeof bookingRequestSchema>
export type BookingAlternative = z.infer<typeof bookingAlternativeSchema>
export type SlotResult = z.infer<typeof slotResultSchema>
export type BookingResponse = z.infer<typeof bookingResponseSchema>
export type ConfirmBookingRequest = z.infer<typeof confirmBookingRequestSchema>
export type Booking = z.infer<typeof bookingSchema>
export type Branch = z.infer<typeof branchSchema>
