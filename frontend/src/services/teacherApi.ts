import type { Teacher } from '../types'
import { createApiGetAll } from '../utils/api'
export const teacherApi = {
  getAll: () => fetchApi<Teacher[]>(`${API_BASE}/teachers`),
}
