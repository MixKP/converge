import { API_ENDPOINTS } from '../config/endpoints'
import { fetchApi, patchApi } from '../utils/api'

export interface CommuteResponse {
  commute_time: number
  source_branch?: number
  destination_branch?: number
}

export const commuteApi = {
  get: () => fetchApi<CommuteResponse>(API_ENDPOINTS.COMMUTE),
  set: (commuteTime: number) =>
    patchApi<CommuteResponse>(API_ENDPOINTS.COMMUTE, { commute_time: commuteTime }),
  getForBranches: (sourceBranch: number, destinationBranch: number) =>
    fetchApi<CommuteResponse>(
      `${API_ENDPOINTS.COMMUTE}?source_branch=${sourceBranch}&destination_branch=${destinationBranch}`
    ),
}
