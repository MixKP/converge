import type { AvailabilityPayload } from '../types'
import { API_BASE } from '../config/api'
import { fetchApi, postApi } from '../utils/api'
export const availabilityApi = {
  submitAvailability: (payload: AvailabilityPayload) =>
    postApi(`${API_BASE}/availability`, payload),
}
