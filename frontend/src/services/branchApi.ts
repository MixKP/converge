import type { Branch } from '../types'
import { API_ENDPOINTS } from '../config/endpoints'
import { createApiGetAll, patchApi } from '../utils/api'

export const branchApi = {
  getAll: createApiGetAll<Branch>(API_ENDPOINTS.BRANCHES),
  setCapacity: (branchId: number, capacity: number) =>
    patchApi<{ message: string }>(`${API_ENDPOINTS.BRANCHES}/${branchId}/capacity`, { capacity }),
}
