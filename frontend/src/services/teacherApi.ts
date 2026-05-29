import type { Teacher } from '../types'
import { createApiGetAll, fetchApi } from '../utils/api'
export const teacherApi = {
  getAll: createApiGetAll<Teacher>('teachers'),
  getBySubject: (subjectId: number) => fetchApi<Teacher[]>(`teachers?subject_id=${subjectId}`),
}
