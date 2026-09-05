import type { Branch } from '../types'
import { API_ENDPOINTS } from '../config/endpoints'
import { createApiGetAll, patchApi, postApi } from '../utils/api'

export const branchApi = {
  getAll: createApiGetAll<Branch>(API_ENDPOINTS.BRANCHES),
  create: (name: string, capacity: number) =>
    postApi<Branch>(API_ENDPOINTS.BRANCHES, { name, capacity }),
  setCapacity: (branchId: number, capacity: number) =>
    patchApi<{ message: string }>(`${API_ENDPOINTS.BRANCHES}/${branchId}/capacity`, { capacity }),
  setStatus: (branchId: number, status: Branch['status']) =>
    patchApi<{ message: string }>(`${API_ENDPOINTS.BRANCHES}/${branchId}/status`, { status }),
}
