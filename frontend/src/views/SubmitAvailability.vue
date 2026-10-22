<script setup lang="ts">
  import PageLayout from '../components/PageLayout.vue'
  import Calendar from '../components/Calendar.vue'
  import CalendarDisabledOverlay from '../components/CalendarDisabledOverlay.vue'
  import { computed } from 'vue'
  import { Loader2, TriangleAlert } from '@lucide/vue'
  import { Button } from '@/components/ui/button'
  import { Dialog, DialogContent, DialogDescription, DialogTitle } from '@/components/ui/dialog'
  import { Label } from '@/components/ui/label'
  import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
  } from '@/components/ui/select'
  import { useSubmitAvailability } from '../composables/useSubmitAvailability'

  const {
    teacherStore,
    isLoading,
    events,
    showConfirm,
    isCountingBookings,
    affectedBookingCount,
    selectedTeacherId,
    selectedTeacher,
    showErrors,
    formattedSlots,
    handleSubmit,
    confirmSubmit,
    cancelConfirm,
  } = useSubmitAvailability()

  // The shared Select speaks strings; the composable holds a numeric id.
  const teacherValue = computed(() =>
    selectedTeacherId.value === null ? '' : String(selectedTeacherId.value)
  )

  const onTeacherChange = (value: unknown) => {
    selectedTeacherId.value = value ? Number(value) : null
  }
</script>

<template>
  <PageLayout
    title="Submit Availability"
    :show-cart="false"
    back-to="/manage"
    back-label="Manage Teachers"
  >
    <div class="mx-auto flex w-full max-w-6xl flex-col gap-4 px-4 py-6 sm:px-6 lg:px-8">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-end sm:gap-2">
        <div class="flex flex-1 flex-col gap-2">
          <Label for="availability-teacher">Teacher</Label>
          <Select :model-value="teacherValue" @update:model-value="onTeacherChange">
            <SelectTrigger id="availability-teacher" class="w-full sm:max-w-sm">
              <SelectValue placeholder="Select a teacher" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem
                v-for="teacher in teacherStore.teachers"
                :key="teacher.id"
                :value="String(teacher.id)"
              >
                {{ teacher.name }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <Button :disabled="isLoading" :aria-busy="isLoading" @click="handleSubmit">
          <Loader2 v-if="isLoading" class="animate-spin" />
          Submit availability
          <span v-if="events.length" class="tabular-nums opacity-80">({{ events.length }})</span>
        </Button>
      </div>

      <p v-if="showErrors && !selectedTeacherId" class="text-destructive px-1 text-sm">
        Select a teacher first.
      </p>

      <!-- Above the calendar, not under it: this says what the page is about to
           do to data that already exists, so it has to be read before the
           drawing starts. It was a muted footnote below the calendar. -->
      <div
        v-if="selectedTeacherId"
        class="border-warning-border bg-warning-surface text-warning-text flex items-start gap-2 rounded-md border px-3 py-2 text-xs"
      >
        <TriangleAlert class="mt-px size-4 shrink-0" />
        <span>
          Drag on the calendar to add a slot. Submitting replaces
          <strong class="font-medium">{{ selectedTeacher?.name }}</strong
          >'s whole week, including any slot not drawn here.
        </span>
      </div>

      <!-- No Card wrapper: Calendar already paints its own card surface (bg,
           border, rounded-2xl, shadow). Nesting it in one gave two borders at
           two different radii, and Card's py-6/px-0 split inset the calendar
           24px top and bottom while leaving it flush left and right.
           Same height as the booking calendar so an hour is the same size in
           both places. -->
      <div class="h-[30rem] lg:h-[36rem]">
        <Calendar
          v-if="selectedTeacherId"
          v-model="events"
          :editable="true"
          :show-header="false"
          class="h-full"
        />
        <CalendarDisabledOverlay
          v-else
          message="Select a teacher first"
          hint="Then drag on the calendar to mark the hours they are free to teach."
        />
      </div>
    </div>

    <Dialog v-model:open="showConfirm">
      <DialogContent>
        <div class="flex flex-col gap-1">
          <DialogTitle>Confirm availability</DialogTitle>
          <DialogDescription>
            This replaces {{ selectedTeacher?.name }}'s entire weekly schedule and deletes every
            class students have booked with them from this week (Sunday) onward, including classes
            earlier this week. Classes before this week are kept.
          </DialogDescription>
        </div>

        <div
          class="border-warning-border bg-warning-surface text-warning-text flex items-start gap-2 rounded-md border px-3 py-2 text-xs"
          role="status"
        >
          <Loader2 v-if="isCountingBookings" class="mt-px size-4 shrink-0 animate-spin" />
          <TriangleAlert v-else class="mt-px size-4 shrink-0" />
          <span v-if="isCountingBookings">Checking booked classes…</span>
          <span v-else-if="affectedBookingCount === null">
            Couldn't load booked classes. All of {{ selectedTeacher?.name }}'s booked classes from
            this week onward will still be deleted.
          </span>
          <span v-else-if="affectedBookingCount === 0">
            {{ selectedTeacher?.name }} has no booked classes from this week onward.
          </span>
          <span v-else>
            <strong class="font-medium tabular-nums">{{ affectedBookingCount }}</strong>
            booked class{{ affectedBookingCount === 1 ? '' : 'es' }} from this week onward will be
            deleted.
          </span>
        </div>

        <div class="border-border flex flex-col gap-1 rounded-md border p-3">
          <p class="text-muted-foreground text-xs">
            {{ events.length }} slot{{ events.length === 1 ? '' : 's' }}
          </p>
          <ul class="flex flex-col gap-0.5">
            <li v-for="(slot, index) in formattedSlots" :key="index" class="text-sm tabular-nums">
              {{ slot.day }} {{ slot.start }}–{{ slot.end }}
            </li>
          </ul>
        </div>

        <div class="flex justify-end gap-2">
          <Button variant="ghost" @click="cancelConfirm">Cancel</Button>
          <Button
            :disabled="isLoading || isCountingBookings"
            :aria-busy="isLoading"
            @click="confirmSubmit"
          >
            <Loader2 v-if="isLoading" class="animate-spin" />
            Confirm
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  </PageLayout>
</template>
