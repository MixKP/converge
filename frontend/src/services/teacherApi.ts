import type { Teacher } from '../types'
import { API_ENDPOINTS } from '../config/endpoints'
import { fetchApi, fetchList, postApi, createApiGetAll } from '../utils/api'
export const teacherApi = {
  getAll: createApiGetAll<Teacher>(API_ENDPOINTS.TEACHERS),
  getBySubject: (subjectId: number) => {
    const url = new URLSearchParams({ subject_id: subjectId.toString() })
    return fetchList<Teacher>(`${API_ENDPOINTS.TEACHERS}?${url}`)
  },
  create: (name: string, email: string, gender: string) =>
    postApi<Teacher>(API_ENDPOINTS.TEACHERS, { name, email, gender }),
  setStatus: (id: number, status: string) =>
    fetchApi<{ message: string }>(`${API_ENDPOINTS.TEACHERS}/${id}/status`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ status }),
    }),
  setGender: (id: number, gender: string) =>
    fetchApi<{ message: string }>(`${API_ENDPOINTS.TEACHERS}/${id}/gender`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ gender }),
    }),
}
