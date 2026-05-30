import type { AvailabilityPayload } from '../types'
import type { BackendTeacherAvailability } from '../utils/availabilityTransform'
import { API_ENDPOINTS } from '../config/endpoints'
import { fetchApi, postApi } from '../utils/api'

export const availabilityApi = {
  getAll: () => fetchApi<BackendTeacherAvailability[]>(API_ENDPOINTS.AVAILABILITY),
  submitAvailability: (payload: AvailabilityPayload) =>
    postApi(API_ENDPOINTS.AVAILABILITY, payload),
}
