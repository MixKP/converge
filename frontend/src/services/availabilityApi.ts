import type { AvailabilityPayload, SubmitAvailabilityResponse } from '../types'
import type { BackendTeacherAvailability } from '../utils/availabilityTransform'
import { API_ENDPOINTS } from '../config/endpoints'
import { fetchList, putApi } from '../utils/api'

export const availabilityApi = {
  getAll: () => fetchList<BackendTeacherAvailability>(API_ENDPOINTS.AVAILABILITY),
  submitAvailability: ({ teacher_id, weekly }: AvailabilityPayload) =>
    putApi<SubmitAvailabilityResponse>(`${API_ENDPOINTS.TEACHERS}/${teacher_id}/availability`, {
      weekly,
    }),
}
