import type { AvailabilityPayload } from '../types'
import type { BackendTeacherAvailability } from '../utils/availabilityTransform'
import { fetchApi, postApi } from '../utils/api'
export const availabilityApi = {
  getAll: () => fetchApi<BackendTeacherAvailability[]>('availability'),
  submitAvailability: (payload: AvailabilityPayload) => postApi('availability', payload),
}
