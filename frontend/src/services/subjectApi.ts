import type { Subject } from '../types'
import { API_ENDPOINTS } from '../config/endpoints'
import { createApiGetAll } from '../utils/api'
export const subjectApi = {
  getAll: createApiGetAll<Subject>(API_ENDPOINTS.SUBJECTS),
}
