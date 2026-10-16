<script setup lang="ts">
  import { computed, onMounted, onUnmounted, ref } from 'vue'
  import { DatabaseBackup, Loader2 } from '@lucide/vue'
  import PageLayout from '../components/PageLayout.vue'
  import ManagementNav from '../components/ManagementNav.vue'
  import { recoveryApi, type BackupStatus } from '../services/recoveryApi'
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
  let refreshTimer: ReturnType<typeof setInterval> | undefined

  const load = async () => {
    try {
      status.value = await recoveryApi.status()
      loadError.value = ''
    } catch (err) {
      loadError.value = err instanceof Error ? err.message : 'Failed to load backup status'
    } finally {
      isLoading.value = false
    }
  }

  onMounted(() => {
    void load()
    refreshTimer = setInterval(() => void load(), 10000)
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
  </PageLayout>
</template>
