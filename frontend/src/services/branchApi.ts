import type { Branch } from '../types'
import { API_ENDPOINTS } from '../config/endpoints'
import { createApiGetAll } from '../utils/api'

export const branchApi = {
  getAll: createApiGetAll<Branch>(API_ENDPOINTS.BRANCHES),
}
