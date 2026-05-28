import type { Teacher } from '../types'
import { API_BASE } from '../config/api'
import { fetchApi } from '../utils/api'

export const teacherApi = {
  getAll: () => fetchApi<Teacher[]>(`${API_BASE}/teachers`),
}
