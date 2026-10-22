import type { BookingRequest, BookingResponse, ConfirmBookingRequest, Booking } from '../types'
import { API_ENDPOINTS } from '../config/endpoints'
import { fetchApi, fetchList, postApi, putApi } from '../utils/api'

export const bookingApi = {
  evaluate: (request: BookingRequest) => postApi<BookingResponse>(API_ENDPOINTS.BOOKINGS, request),

  confirm: (request: ConfirmBookingRequest) =>
    postApi<Booking>(`${API_ENDPOINTS.BOOKINGS}/confirm`, request),

  list: () => fetchList<Booking>(API_ENDPOINTS.BOOKINGS),

  cancel: (id: number) => fetchApi<void>(`${API_ENDPOINTS.BOOKINGS}/${id}`, { method: 'DELETE' }),

  // Only the time moves; teacher, branch, subject and student stay as booked.
  reschedule: (id: number, startTime: string, endTime: string) =>
    putApi<Booking>(`${API_ENDPOINTS.BOOKINGS}/${id}`, {
      start_time: startTime,
      end_time: endTime,
    }),
}
