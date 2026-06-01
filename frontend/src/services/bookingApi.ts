import type {
  BookingRequest,
  BookingResponse,
  ConfirmBookingRequest,
  Booking,
} from '../types'
import { API_ENDPOINTS } from '../config/endpoints'
import { fetchApi, postApi } from '../utils/api'

export const bookingApi = {
  evaluate: (request: BookingRequest) =>
    postApi<BookingResponse>(API_ENDPOINTS.BOOKINGS, request),

  confirm: (request: ConfirmBookingRequest) =>
    postApi<Booking>(`${API_ENDPOINTS.BOOKINGS}/confirm`, request),

  listAll: () => fetchApi<Booking[]>(API_ENDPOINTS.BOOKINGS),

  cancel: (id: number) =>
    fetchApi<void>(`${API_ENDPOINTS.BOOKINGS}/${id}`, { method: 'DELETE' }),
}
