import { API_ENDPOINTS } from '../config/endpoints'
import { fetchApi } from '../utils/api'

export interface BackupStatus {
  last_success: string | null
  last_failure: string | null
  latest: string
  queued: boolean
  running: boolean
  restoring: boolean
  restore_backup: string
  restore_error: string
  restored_at: string | null
}

export const recoveryApi = {
  status: () => fetchApi<BackupStatus>(API_ENDPOINTS.RECOVERY_BACKUPS),
  requestBackup: () =>
    fetchApi<{ message: string }>(API_ENDPOINTS.RECOVERY_BACKUPS, { method: 'POST' }),
  requestRestore: (backupName: string, confirmation: string) =>
    fetchApi<{ message: string }>(API_ENDPOINTS.RECOVERY_RESTORES, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ backup_name: backupName, confirmation }),
    }),
}
