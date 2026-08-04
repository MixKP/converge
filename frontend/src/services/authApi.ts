import { API_ENDPOINTS } from '../config/endpoints'
import { authResponseSchema, userSchema } from '../schemas/auth'
import type { AuthSession, AuthUser, RegisterRequest } from '../types'
import { postApi } from '../utils/api'

export const authApi = {
  login: async (name: string, password: string): Promise<AuthSession> => {
    const raw = await postApi<unknown>(API_ENDPOINTS.LOGIN, { name, password })
    return authResponseSchema.parse(raw)
  },
  register: async (payload: RegisterRequest): Promise<AuthUser> => {
    const raw = await postApi<unknown>(API_ENDPOINTS.REGISTER, payload)
    return userSchema.parse(raw)
  },
}
