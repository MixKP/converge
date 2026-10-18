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

export interface AuditEvent {
  id: number
  occurred_at: string
  resource: string
  operation: string
  row_key: string
  changed_fields: string[]
}

export interface PhysicalBackup {
  label: string
  completed_at: string
}

export interface PhysicalStatus {
  backups: PhysicalBackup[]
  archive_segment: string
  last_archived_at: string | null
  last_failed_at: string | null
  archive_status_error?: string
  repository_error?: string
}

export interface RecoveryTimeline {
  events: AuditEvent[]
  physical: PhysicalStatus | null
  physical_error: string
}

export interface RecoveryPreview {
  id?: string
  state: 'idle' | 'running' | 'ready' | 'failed'
  target?: string
  started_at?: string
  finished_at?: string
  counts?: Record<string, number>
  absent_events?: AuditEvent[]
  comparison_error?: string
  error?: string
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
  timeline: (beforeId?: number, after?: string) => {
    const params = new URLSearchParams()
    if (beforeId) params.set('before_id', String(beforeId))
    if (after) params.set('after', after)
    const query = params.toString()
    return fetchApi<RecoveryTimeline>(`${API_ENDPOINTS.RECOVERY_TIMELINE}${query ? `?${query}` : ''}`)
  },
  previewStatus: () => fetchApi<RecoveryPreview>(API_ENDPOINTS.RECOVERY_PREVIEWS),
  requestPreview: (target: string) =>
    fetchApi<RecoveryPreview>(API_ENDPOINTS.RECOVERY_PREVIEWS, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ target }),
    }),
  applyPreview: (id: string, confirmation: string) =>
    fetchApi<{ message: string }>(`${API_ENDPOINTS.RECOVERY_PREVIEWS}/apply`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id, confirmation }),
    }),
}
