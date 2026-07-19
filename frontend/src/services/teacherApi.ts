import type { Teacher } from '../types'
import { API_ENDPOINTS } from '../config/endpoints'
import { createApiGetAll, fetchList } from '../utils/api'
export const teacherApi = {
  getAll: createApiGetAll<Teacher>(API_ENDPOINTS.TEACHERS),
  getBySubject: (subjectId: number) => {
    const url = new URLSearchParams({ subject_id: subjectId.toString() })
    return fetchList<Teacher>(`${API_ENDPOINTS.TEACHERS}?${url}`)
  },
}
