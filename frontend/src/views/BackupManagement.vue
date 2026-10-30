<script setup lang="ts">
  import { computed, onMounted, onUnmounted, ref } from 'vue'
  import { DatabaseBackup, Loader2 } from '@lucide/vue'
  import PageLayout from '../components/PageLayout.vue'
  import ManagementNav from '../components/ManagementNav.vue'
  import { recoveryApi, type BackupStatus, type RecoveryTimeline, type RecoveryPreview, type AuditEvent } from '../services/recoveryApi'
  import { useNotification } from '../composables/useNotification'
  import { Button } from '@/components/ui/button'
  import { Card, CardContent } from '@/components/ui/card'
  import { Dialog, DialogContent, DialogDescription, DialogTitle } from '@/components/ui/dialog'
  import { Input } from '@/components/ui/input'
  import { Label } from '@/components/ui/label'

  const { showSuccess, showError } = useNotification()
  const status = ref<BackupStatus | null>(null)
  const loadError = ref('')
  const isLoading = ref(true)
  const isRequesting = ref(false)
  const isRequestingRestore = ref(false)
  const showRestoreConfirm = ref(false)
  const restoreConfirmation = ref('')
  const timeline = ref<RecoveryTimeline | null>(null)
  const timelineError = ref('')
  const preview = ref<RecoveryPreview | null>(null)
  const previewTarget = ref('')
  const isRequestingPreview = ref(false)
  const isApplyingPreview = ref(false)
  const showApplyConfirm = ref(false)
  const applyConfirmation = ref('')
  const isLoadingMore = ref(false)
  const hasMoreEvents = ref(true)
  const hasLoadedOlder = ref(false)
  const selectedEvent = ref<AuditEvent | null>(null)
  const selectedPosition = ref<'before' | 'after'>('after')
  const previewError = ref('')
  let visiblePreviewID: string | null | undefined
  let lastRestoredAt: string | null = null
  let refreshTimer: ReturnType<typeof setInterval> | undefined

  const load = async () => {
    try {
      status.value = await recoveryApi.status()
      if (status.value.restored_at && status.value.restored_at !== lastRestoredAt) {
        lastRestoredAt = status.value.restored_at
        selectedEvent.value = null
        preview.value = null
        visiblePreviewID = null
        void loadRecovery(true)
      }
      loadError.value = ''
    } catch (err) {
      loadError.value = err instanceof Error ? err.message : 'Failed to load backup status'
    } finally {
      isLoading.value = false
    }
  }

  const loadRecovery = async (refreshEvents = true) => {
    try {
      const fresh = await recoveryApi.timeline()
      if (!refreshEvents && hasLoadedOlder.value && timeline.value) fresh.events = timeline.value.events
      else {
        hasLoadedOlder.value = false
        hasMoreEvents.value = fresh.events.length === 100
      }
      timeline.value = fresh
      timelineError.value = ''
      if (!timeline.value.physical_error) {
        const currentPreview = await recoveryApi.previewStatus()
        preview.value = visiblePreviewID === undefined || currentPreview.id === visiblePreviewID ? currentPreview : null
      }
    } catch (err) {
      timelineError.value = err instanceof Error ? err.message : 'Failed to load recovery timeline'
    }
  }

  onMounted(() => {
    void load()
    void loadRecovery()
    refreshTimer = setInterval(() => {
      void load()
      void loadRecovery(false)
    }, 10000)
  })
  onUnmounted(() => clearInterval(refreshTimer))

  const lastBackup = computed(() => {
    if (!status.value?.last_success) return 'No successful backup yet'
    return new Intl.DateTimeFormat('en-GB', {
      dateStyle: 'medium',
      timeStyle: 'short',
      timeZone: 'Asia/Bangkok',
    }).format(new Date(status.value.last_success)) + ' ICT'
  })

  const hasRecentFailure = computed(
    () =>
      !!status.value?.last_failure &&
      (!status.value.last_success || status.value.last_failure > status.value.last_success)
  )

  const requestBackup = async () => {
    isRequesting.value = true
    try {
      await recoveryApi.requestBackup()
      showSuccess('Backup requested. The time below updates when it finishes.')
      await load()
    } catch (err) {
      showError(err, err instanceof Error ? err.message : 'Failed to request backup')
    } finally {
      isRequesting.value = false
    }
  }

  const expectedConfirmation = computed(() => `RESTORE LIVE DATA from ${status.value?.latest ?? ''}`)
  const expectedApplyConfirmation = computed(() => `APPLY WAL PREVIEW ${preview.value?.id ?? ''}`)

  const localTime = (value: string) => new Intl.DateTimeFormat('en-GB', {
    dateStyle: 'medium', timeStyle: 'short', timeZone: 'Asia/Bangkok',
  }).format(new Date(value)) + ' ICT'

  const describeEvent = (event: AuditEvent) =>
    `${event.operation} ${event.resource.replaceAll('_', ' ')}${event.row_key ? ` #${event.row_key}` : ''}`

  const beforeFirstBackup = (event: AuditEvent) => {
    const first = timeline.value?.physical?.backups[0]?.completed_at
    return !first || new Date(event.occurred_at).getTime() < new Date(first).getTime()
  }

  const previewEvent = async (event: AuditEvent, position: 'before' | 'after') => {
    selectedEvent.value = event
    selectedPosition.value = position
    previewError.value = ''
    visiblePreviewID = null
    if (beforeFirstBackup(event)) {
      preview.value = null
      previewError.value = `This change is before the first physical backup (${timeline.value?.physical?.backups[0] ? localTime(timeline.value.physical.backups[0].completed_at) : 'none available'}). Choose a later change.`
      return
    }
    isRequestingPreview.value = true
    try {
      preview.value = await recoveryApi.requestEventPreview(event.id, position)
      visiblePreviewID = preview.value.id ?? null
      showSuccess('Recovery preview started. Live data is unchanged.')
    } catch (err) {
      preview.value = null
      previewError.value = err instanceof Error ? err.message : 'Failed to start preview'
    } finally {
      isRequestingPreview.value = false
    }
  }

  const requestPreview = async () => {
    if (!previewTarget.value) return
    selectedEvent.value = null
    previewError.value = ''
    visiblePreviewID = null
    isRequestingPreview.value = true
    try {
      preview.value = await recoveryApi.requestPreview(new Date(previewTarget.value).toISOString())
      visiblePreviewID = preview.value.id ?? null
      showSuccess('Recovery preview started. Live data is unchanged.')
    } catch (err) {
      preview.value = null
      previewError.value = err instanceof Error ? err.message : 'Failed to start preview'
    } finally {
      isRequestingPreview.value = false
    }
  }

  const applyPreview = async () => {
    if (!preview.value?.id || applyConfirmation.value !== expectedApplyConfirmation.value) return
    isApplyingPreview.value = true
    try {
      await recoveryApi.applyPreview(preview.value.id, applyConfirmation.value)
      showApplyConfirm.value = false
      applyConfirmation.value = ''
      showSuccess('Applying the recovered data. The app will pause while it finishes.')
      await load()
    } catch (err) {
      showError(err, err instanceof Error ? err.message : 'Failed to apply recovery preview')
    } finally {
      isApplyingPreview.value = false
    }
  }

  const loadMoreEvents = async () => {
    const oldest = timeline.value?.events.at(-1)
    if (!oldest || isLoadingMore.value) return
    isLoadingMore.value = true
    try {
      const next = await recoveryApi.timeline(oldest)
      if (timeline.value) timeline.value.events.push(...next.events)
      hasLoadedOlder.value = true
      hasMoreEvents.value = next.events.length === 100
    } catch (err) {
      showError(err, err instanceof Error ? err.message : 'Failed to load older changes')
    } finally {
      isLoadingMore.value = false
    }
  }

  const requestRestore = async () => {
    if (!status.value?.latest || restoreConfirmation.value !== expectedConfirmation.value) return
    isRequestingRestore.value = true
    try {
      await recoveryApi.requestRestore(status.value.latest, restoreConfirmation.value)
      showRestoreConfirm.value = false
      restoreConfirmation.value = ''
      showSuccess('Restore started. This page will show the result when it finishes.')
      await load()
    } catch (err) {
      showError(err, err instanceof Error ? err.message : 'Failed to start restore')
      await load()
    } finally {
      isRequestingRestore.value = false
    }
  }
</script>

<template>
  <PageLayout title="Manage Backups" :show-cart="false" back-to="/dashboard" back-label="Dashboard">
    <ManagementNav />

    <div class="mx-auto flex w-full max-w-6xl flex-col gap-4 px-4 py-6 sm:px-6 lg:px-8">
      <Card class="max-w-2xl">
        <CardContent class="flex flex-col gap-5">
          <div class="flex items-start gap-3">
            <DatabaseBackup class="text-primary mt-0.5 size-5" />
            <div>
              <h2 class="font-medium">Database backups</h2>
              <p class="text-muted-foreground mt-1 text-sm">
                Backups run automatically. You can also request one now before making major changes.
              </p>
            </div>
          </div>

          <p v-if="isLoading" class="text-muted-foreground flex items-center gap-2 text-sm">
            <Loader2 class="size-4 animate-spin" /> Loading backup status…
          </p>
          <div v-else-if="loadError" class="text-destructive text-sm">
            <p>{{ loadError }}</p>
            <Button class="mt-3" variant="outline" @click="load">Retry</Button>
          </div>
          <div v-else-if="status" class="flex flex-col gap-3">
            <div>
              <p class="text-muted-foreground text-xs">Last successful backup</p>
              <p class="mt-1 font-medium">{{ lastBackup }}</p>
            </div>
            <p v-if="status.running" class="text-sm">A backup is running now.</p>
            <p v-else-if="status.queued" class="text-sm">A backup is queued.</p>
            <p v-if="hasRecentFailure" class="text-destructive text-sm">
              The last backup attempt failed. The previous successful backup is still available.
            </p>
            <Button
              class="self-start"
              :disabled="isRequesting || status.queued || status.running || status.restoring"
              @click="requestBackup"
            >
              <Loader2 v-if="isRequesting" class="size-4 animate-spin" />
              Back up now
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card class="max-w-3xl">
        <CardContent class="flex flex-col gap-4">
          <div>
            <h2 class="font-medium">Recovery timeline</h2>
            <p class="text-muted-foreground mt-1 text-sm">
              Physical backups and archived WAL can restore the database to a chosen time. Recorded
              changes help you choose a time; they are not individual undo buttons.
            </p>
          </div>
          <p v-if="timelineError" class="text-destructive text-sm">{{ timelineError }}</p>
          <template v-else-if="timeline">
            <p v-if="timeline.physical_error" class="text-destructive text-sm">
              {{ timeline.physical_error }}. The logical backups above remain separate.
            </p>
            <template v-else-if="timeline.physical">
              <p v-if="timeline.physical.repository_error" class="text-destructive text-sm">
                Backup repository: {{ timeline.physical.repository_error }}
              </p>
              <div class="text-sm">
                <p>Physical backups: {{ timeline.physical.backups.length }}</p>
                <p v-if="timeline.physical.backups.length">
                  Latest base backup:
                  {{ localTime(timeline.physical.backups[timeline.physical.backups.length - 1]!.completed_at) }}
                </p>
                <p v-if="timeline.physical.archive_segment">
                  Latest archived WAL segment: {{ timeline.physical.archive_segment }}
                </p>
                <p v-if="timeline.physical.last_archived_at">
                  Last archive completed: {{ localTime(timeline.physical.last_archived_at) }}
                </p>
                <p v-if="timeline.physical.archive_status_error" class="text-destructive">
                  {{ timeline.physical.archive_status_error }}
                </p>
                <p v-if="timeline.physical.last_failed_at && (!timeline.physical.last_archived_at || timeline.physical.last_failed_at > timeline.physical.last_archived_at)" class="text-destructive">
                  WAL archiving has a recent failure. Check the backup store before recovery.
                </p>
              </div>
              <p class="text-muted-foreground text-xs">
                Backup and WAL markers indicate a possible recovery period. Only a successful isolated
                preview verifies the selected time. Live data is not changed.
              </p>
              <div class="border-border border-t pt-4">
                <div class="flex items-center justify-between gap-3">
                  <h3 class="font-medium">Recorded application changes</h3>
                  <Button variant="outline" size="sm" @click="loadRecovery(true)">Refresh changes</Button>
                </div>
                <p class="text-muted-foreground mt-1 text-xs">
                  Newest first. Select a change to preview the database just after it. These records
                  describe application changes; they are separate from archived WAL.
                </p>
                <p v-if="!timeline.events.length" class="text-muted-foreground mt-3 text-sm">No recorded changes yet.</p>
                <ul v-else class="mt-3 max-h-96 space-y-2 overflow-y-auto">
                  <li v-for="event in timeline.events" :key="event.id">
                    <button type="button" class="border-border hover:bg-muted/50 w-full rounded-md border p-3 text-left text-sm disabled:cursor-not-allowed disabled:opacity-60" :class="selectedEvent?.id === event.id ? 'border-primary bg-muted/50' : ''" :disabled="!!timeline.physical_error || isRequestingPreview || preview?.state === 'running'" @click="previewEvent(event, 'after')">
                      <span class="block capitalize">{{ describeEvent(event) }}</span>
                      <span class="text-muted-foreground block text-xs">{{ localTime(event.occurred_at) }}<span v-if="event.changed_fields.length"> · {{ event.changed_fields.join(', ') }}</span></span>
                      <span v-if="beforeFirstBackup(event)" class="text-destructive block text-xs">Before first physical backup</span>
                    </button>
                  </li>
                </ul>
                <Button v-if="hasMoreEvents && timeline.events.length >= 100" class="mt-3" variant="outline" :disabled="isLoadingMore" @click="loadMoreEvents">Load older changes</Button>
              </div>
              <div v-if="selectedEvent" class="border-border rounded-md border p-3 text-sm">
                <p class="font-medium capitalize">{{ describeEvent(selectedEvent) }}</p>
                <p class="text-muted-foreground text-xs">{{ localTime(selectedEvent.occurred_at) }}</p>
                <div class="mt-3 flex gap-2">
                  <Button size="sm" :variant="selectedPosition === 'after' ? 'default' : 'outline'" :disabled="isRequestingPreview || preview?.state === 'running'" @click="previewEvent(selectedEvent, 'after')">After this change</Button>
                  <Button size="sm" :variant="selectedPosition === 'before' ? 'default' : 'outline'" :disabled="isRequestingPreview || preview?.state === 'running'" @click="previewEvent(selectedEvent, 'before')">Before this change</Button>
                </div>
              </div>
              <details class="border-border border-t pt-3 text-sm">
                <summary class="cursor-pointer font-medium">Advanced: choose a time</summary>
                <div class="mt-3 flex flex-col gap-2 sm:flex-row sm:items-end">
                  <div class="space-y-1">
                    <Label for="preview-target">Preview database at this time (your local time)</Label>
                    <Input id="preview-target" v-model="previewTarget" type="datetime-local" />
                  </div>
                  <Button class="self-start" :disabled="!previewTarget || isRequestingPreview || preview?.state === 'running' || !timeline.physical.backups.length || !!timeline.physical.repository_error" @click="requestPreview">
                    <Loader2 v-if="isRequestingPreview" class="size-4 animate-spin" />
                    Preview recovery
                  </Button>
                </div>
              </details>
              <p v-if="previewError" class="text-destructive text-sm">{{ previewError }}</p>
              <div v-if="preview?.state === 'running'" class="text-sm">Restoring the preview database…</div>
              <div v-else-if="preview?.state === 'failed'" class="text-destructive text-sm">
                Preview failed: {{ preview.error }}
              </div>
              <div v-else-if="preview?.state === 'ready'" class="text-sm">
                <p class="font-medium">Preview ready for {{ preview.target ? localTime(preview.target) : '' }}</p>
                <p v-if="preview.event_id" class="mt-1">Verified {{ preview.position }} the selected change in the recovered database.</p>
                <p v-if="preview.counts">Accounts {{ preview.counts.users }}, teachers {{ preview.counts.teachers }}, branches {{ preview.counts.branches }}, bookings {{ preview.counts.bookings }}.</p>
                <p v-if="preview.comparison_error" class="text-destructive mt-2">{{ preview.comparison_error }}</p>
                <p v-else-if="preview.absent_events?.length" class="mt-2 font-medium">Changes absent from this preview, among the latest 100 live changes at preview time:</p>
                <ul v-if="preview.absent_events?.length" class="list-inside list-disc">
                  <li v-for="event in preview.absent_events" :key="event.id" class="capitalize">{{ describeEvent(event) }} · {{ localTime(event.occurred_at) }}</li>
                </ul>
                <p v-if="!preview.comparison_error && !preview.absent_events?.length" class="mt-2">No differences found among the latest 100 live changes at preview time.</p>
                <p class="text-muted-foreground text-xs">Older changes may also differ. The comparison is a snapshot taken when the preview finished.</p>
                <p class="text-muted-foreground mt-1">This is a read-only preview. It has not replaced the live database.</p>
                <Button class="mt-3" variant="destructive" :disabled="!preview.id || (preview.event_id && !preview.selection_matched) || !status || status.restoring || status.running || status.queued" @click="showApplyConfirm = true">
                  Apply this recovery to live data
                </Button>
              </div>
            </template>

          </template>
        </CardContent>
      </Card>

      <Card class="max-w-2xl">
        <CardContent>
          <h2 class="font-medium">Restore live data</h2>
          <p class="text-muted-foreground mt-2 text-sm">
            Replace the current database with a backup while the app is running. Changes made after
            that backup will be lost. The app pauses requests during restoration.
          </p>
          <p v-if="status?.restoring" class="mt-3 text-sm">
            Restoring {{ status.restore_backup }}. The app is temporarily unavailable.
          </p>
          <p v-else-if="status?.restore_error" class="text-destructive mt-3 text-sm">
            Restore failed: {{ status.restore_error }}
          </p>
          <p v-else-if="status?.restored_at" class="mt-3 text-sm">
            Restored from {{ status.restore_backup }}. Check your records before continuing work.
          </p>
          <Button
            class="mt-4"
            variant="destructive"
            :disabled="!status?.latest || status.running || status.queued || status.restoring || isRequestingRestore"
            @click="showRestoreConfirm = true"
          >
            Restore latest backup
          </Button>
          <p class="text-muted-foreground mt-3 text-xs">
            If the database is down and this page cannot load, use the separate recovery tool.
          </p>
        </CardContent>
      </Card>
    </div>

    <Dialog v-model:open="showRestoreConfirm">
      <DialogContent class="max-w-lg">
        <DialogTitle>Replace live data?</DialogTitle>
        <DialogDescription>
          This replaces current accounts, teachers, branches, and bookings with the backup from
          {{ lastBackup }}. Changes made since then will be lost. A safety backup of the current data
          is created first.
        </DialogDescription>
        <p class="break-all text-xs">Backup: {{ status?.latest }}</p>
        <div class="space-y-2">
          <Label for="restore-confirmation">Type this phrase to confirm:</Label>
          <p class="break-all font-mono text-sm">{{ expectedConfirmation }}</p>
          <Input id="restore-confirmation" v-model="restoreConfirmation" autocomplete="off" />
        </div>
        <div class="flex justify-end gap-2">
          <Button variant="outline" @click="showRestoreConfirm = false">Cancel</Button>
          <Button
            variant="destructive"
            :disabled="isRequestingRestore || restoreConfirmation !== expectedConfirmation"
            @click="requestRestore"
          >
            <Loader2 v-if="isRequestingRestore" class="size-4 animate-spin" />
            Replace live data
          </Button>
        </div>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="showApplyConfirm">
      <DialogContent class="max-w-lg">
        <DialogTitle>Apply recovered data?</DialogTitle>
        <DialogDescription>
          The previewed database at {{ preview?.target ? localTime(preview.target) : '' }} will replace
          live accounts, teachers, branches, and bookings. Later changes will be lost. A safety backup
          of the current database is created first. The app pauses during the restore.
        </DialogDescription>
        <div class="space-y-2">
          <Label for="apply-confirmation">Type this phrase to confirm:</Label>
          <p class="break-all font-mono text-sm">{{ expectedApplyConfirmation }}</p>
          <Input id="apply-confirmation" v-model="applyConfirmation" autocomplete="off" />
        </div>
        <div class="flex justify-end gap-2">
          <Button variant="outline" @click="showApplyConfirm = false">Cancel</Button>
          <Button variant="destructive" :disabled="isApplyingPreview || applyConfirmation !== expectedApplyConfirmation" @click="applyPreview">
            <Loader2 v-if="isApplyingPreview" class="size-4 animate-spin" />
            Replace live data
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  </PageLayout>
</template>
