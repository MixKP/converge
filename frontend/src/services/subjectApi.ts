import type { Subject } from '../types'
import { createApiGetAll } from '../utils/api'
export const subjectApi = {
  getAll: createApiGetAll<Subject>('subjects'),
}
