import type { Teacher } from '../types'
import { API_ENDPOINTS } from '../config/endpoints'
import { createApiGetAll, fetchApi } from '../utils/api'
export const teacherApi = {
  getAll: createApiGetAll<Teacher>(API_ENDPOINTS.TEACHERS),
  getBySubject: (subjectId: number) => {
    const url = new URLSearchParams({ subject_id: subjectId.toString() })
    return fetchApi<Teacher[]>(`${API_ENDPOINTS.TEACHERS}?${url}`)
  },
}
