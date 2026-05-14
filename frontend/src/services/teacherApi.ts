import type { Teacher } from '../types/teacher'

const API_BASE = 'http://localhost:8080/api'

export const teacherApi = {
  async getAll(): Promise<Teacher[]> {
    const response = await fetch(`${API_BASE}/teachers`)
    if (!response.ok) {
      throw new Error(`Failed to fetch teachers: ${response.statusText}`)
    }
    return response.json()
  },
}
