import type { AvailabilityPayload } from '../types'
import { API_BASE } from '../config/api'
import { fetchApi } from '../utils/api'

export const availabilityApi = {
  submitAvailability: (payload: AvailabilityPayload) =>
    fetchApi(`${API_BASE}/availability`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    }),
}
