import { API_ENDPOINTS } from '../config/endpoints'
import { parentWithStudentsSchema, userSchema } from '../schemas/auth'
import type { AuthUser, ParentWithStudents } from '../types'
import { deleteApi, fetchList, postApi } from '../utils/api'

export const userApi = {
  listStudents: async (): Promise<AuthUser[]> => {
    const raw = await fetchList<unknown>(API_ENDPOINTS.STUDENTS)
    return userSchema.array().parse(raw)
  },
  listParents: async (): Promise<ParentWithStudents[]> => {
    const raw = await fetchList<unknown>(API_ENDPOINTS.PARENTS)
    return parentWithStudentsSchema.array().parse(raw)
  },
  linkStudent: (parentId: number, studentId: number) =>
    postApi<{ message: string }>(`${API_ENDPOINTS.PARENTS}/${parentId}/students`, {
      student_id: studentId,
    }),
  unlinkStudent: (parentId: number, studentId: number) =>
    deleteApi<{ message: string }>(`${API_ENDPOINTS.PARENTS}/${parentId}/students/${studentId}`),
}
