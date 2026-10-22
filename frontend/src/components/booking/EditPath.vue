<script setup lang="ts">
  import { ref, computed, watch } from 'vue'
  import { storeToRefs } from 'pinia'
  import { ArrowRight, Loader2, MapPin, Trash2 } from '@lucide/vue'
  import type { EventClickInfo, EventInput } from '@fullcalendar/vue3'
  import { useBookingStore } from '../../stores/bookingStore'
  import { useNotification } from '../../composables/useNotification'
  import { bookingApi } from '../../services/bookingApi'
  import { businessHoursForTeachers } from '../../utils/availabilityTransform'
  import {
    buildSubjectTones,
    subjectTone,
    subjectToneSoftVar,
    subjectToneVar,
  } from '../../utils/subjectColor'
  import {
    editBlocks,
    foldToSingleEvent,
    sameTimes,
    upcomingForStudent,
  } from '../../utils/bookingEdit'
  import { bookingsInWeekOf } from '../../utils/roster'
  import Calendar from '../Calendar.vue'
  import { Button } from '@/components/ui/button'
  import { Card, CardContent } from '@/components/ui/card'
  import { Dialog, DialogContent, DialogDescription, DialogTitle } from '@/components/ui/dialog'
  import type { Booking } from '../../types'

  const store = useBookingStore()
  const { selectedStudent, confirmedBookings, availabilityCache, isAvailabilityLoaded, subjects } =
    storeToRefs(store)
  const { showSuccess, showError } = useNotification()

  const studentId = computed(() => selectedStudent.value?.id ?? null)
  const now = ref(Date.now())

  // The grid shows the current week, the same week the dashboard does.
  const weekBookings = computed(() =>
    bookingsInWeekOf(confirmedBookings.value, new Date(now.value))
  )

  const upcoming = computed(() =>
    studentId.value === null
      ? []
      : upcomingForStudent(weekBookings.value, studentId.value, now.value)
  )

  const dayFormat = new Intl.DateTimeFormat('en-US', {
    weekday: 'short',
    month: 'short',
    day: 'numeric',
  })
  const timeFormat = new Intl.DateTimeFormat('en-US', {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  })
  const formatSlot = (start: string | Date, end: string | Date): string =>
    `${dayFormat.format(new Date(start))} · ${timeFormat.format(new Date(start))}–${timeFormat.format(new Date(end))}`

  const teacherName = (b: Booking) => b.teacher_name ?? `Teacher #${b.teacher_id}`
  const subjectName = (b: Booking) => b.subject_name ?? `Subject #${b.subject_id}`
  const branchName = (b: Booking) => b.branch_name ?? `Branch #${b.branch_id}`

  const tones = computed(() => buildSubjectTones(subjects.value))

  // ---- the class being moved -----------------------------------------------
  const selected = ref<Booking | null>(null)
  const draft = ref<EventInput[]>([])

  // The calendar only adds and removes events by id; it never moves one whose
  // id it already has. A slot drawn elsewhere, or a rejected drag, would leave
  // the block where the grid last put it. A fresh id per change makes the grid
  // redraw the class exactly where the draft says it is.
  const revision = ref(0)
  const draftId = computed(() =>
    selected.value ? `booking-${selected.value.id}-${revision.value}` : ''
  )
  const setDraft = (base: EventInput) => {
    revision.value++
    draft.value = [{ ...base, id: draftId.value }]
  }

  const select = (b: Booking) => {
    now.value = Date.now()
    selected.value = b
    setDraft({ title: subjectName(b), start: b.start_time, end: b.end_time })
  }
  const clearSelection = () => {
    selected.value = null
    draft.value = []
  }

  watch(studentId, clearSelection)

  const draftEvent = computed(() => draft.value[0] ?? null)
  const isMoved = computed(
    () => !!selected.value && !!draftEvent.value && !sameTimes(draftEvent.value, selected.value)
  )

  const removing = ref<Booking | null>(null)

  const onDraftChange = (events: EventInput[]) => {
    const target = selected.value
    const current = draftEvent.value
    if (!target || !current) return
    const next = foldToSingleEvent(events, { ...current, id: draftId.value })
    // The grid's own delete on the class means "remove this class".
    if (!next) {
      setDraft(current)
      removing.value = target
      return
    }
    if (new Date(next.start as string | Date).getTime() <= Date.now()) {
      setDraft(current)
      showError(null, 'Pick a time that has not passed yet.')
      return
    }
    setDraft(next)
  }

  const blocks = computed(() =>
    studentId.value === null
      ? { ownClasses: [], teacherBusy: [] }
      : editBlocks(weekBookings.value, studentId.value, selected.value, now.value)
  )

  const ownClassEvents = computed<EventInput[]>(() =>
    blocks.value.ownClasses.map((b) => {
      const tone = subjectTone(tones.value, subjectName(b))
      return {
        id: `own-${b.id}`,
        title: `${subjectName(b)} · ${teacherName(b)}`,
        start: b.start_time,
        end: b.end_time,
        editable: false,
        color: subjectToneSoftVar(tone) ?? 'var(--bg-subtle)',
        borderColor: subjectToneVar(tone) ?? 'var(--border)',
        contrastColor: 'var(--text-primary)',
        className: 'booked-event cursor-pointer',
        extendedProps: { isBooked: true, bookingId: b.id },
      }
    })
  )

  const teacherBusyEvents = computed<EventInput[]>(() =>
    blocks.value.teacherBusy.map((b) => ({
      id: `busy-${b.id}`,
      title: 'Booked',
      start: b.start_time,
      end: b.end_time,
      editable: false,
      color: 'var(--bg-subtle)',
      contrastColor: 'var(--text-primary)',
      className: 'booked-event',
      extendedProps: { isBooked: true },
    }))
  )

  const additionalEvents = computed(() => [...ownClassEvents.value, ...teacherBusyEvents.value])

  // Nothing selected: every class is on a plain grid and can be clicked. Once
  // a class is picked, the grid shows its teacher's hours and only it moves.
  const businessHours = computed(() =>
    selected.value
      ? businessHoursForTeachers(availabilityCache.value, [selected.value.teacher_id])
      : []
  )
  const hasAvailability = computed(
    () => Array.isArray(businessHours.value) && businessHours.value.length > 0
  )

  const onEventClick = (info: EventClickInfo) => {
    const id = info.event.extendedProps?.bookingId as number | undefined
    if (id === undefined || !info.event.id.startsWith('own-')) return
    const booking = upcoming.value.find((b) => b.id === id)
    if (booking) select(booking)
  }

  // ---- save / remove --------------------------------------------------------
  const isSaving = ref(false)

  const saveTime = async () => {
    const target = selected.value
    const event = draftEvent.value
    if (!target || !event || !isMoved.value || isSaving.value) return
    isSaving.value = true
    try {
      const start = new Date(event.start as string | Date).toISOString()
      const end = new Date(event.end as string | Date).toISOString()
      await bookingApi.reschedule(target.id, start, end)
      showSuccess(`${subjectName(target)} moved to ${formatSlot(start, end)}`, 3000)
      clearSelection()
      await store.fetchConfirmedBookings()
    } catch (error) {
      showError(error, 'Failed to change the time')
    } finally {
      isSaving.value = false
    }
  }

  const isRemoving = ref(false)

  const confirmRemove = async () => {
    const target = removing.value
    if (!target || isRemoving.value) return
    isRemoving.value = true
    try {
      await bookingApi.cancel(target.id)
      showSuccess(
        `${subjectName(target)} on ${formatSlot(target.start_time, target.end_time)} removed`,
        3000
      )
      if (selected.value?.id === target.id) clearSelection()
      removing.value = null
      await store.fetchConfirmedBookings()
    } catch (error) {
      showError(error, 'Failed to remove the class')
    } finally {
      isRemoving.value = false
    }
  }
</script>

<template>
  <Card>
    <CardContent class="flex flex-col gap-3 py-4">
      <div class="h-[30rem] lg:h-[36rem]">
        <Calendar
          :key="studentId ?? 'none'"
          :model-value="draft"
          :additional-events="additionalEvents"
          :editable="selected !== null"
          :backdrop="selected ? 'hours' : 'none'"
          :business-hours="businessHours"
          constraint="businessHours"
          :show-header="false"
          class="h-full"
          @update:model-value="onDraftChange"
          @event-click="onEventClick"
        />
      </div>

      <p v-if="!selected && upcoming.length === 0" class="text-muted-foreground text-sm">
        {{ selectedStudent?.name }} has no upcoming classes this week.
      </p>

      <div
        v-if="selected"
        class="border-border bg-muted/50 flex flex-col gap-3 rounded-md border px-3 py-3 sm:flex-row sm:items-center sm:justify-between"
      >
        <div class="flex min-w-0 flex-col gap-1 text-sm">
          <span class="font-medium">
            {{ subjectName(selected) }} · {{ teacherName(selected) }}
            <span class="text-muted-foreground inline-flex items-center gap-1 font-normal">
              <MapPin class="size-3" />
              {{ branchName(selected) }}
            </span>
          </span>
          <span class="flex flex-wrap items-center gap-x-2 tabular-nums">
            <span :class="isMoved ? 'text-muted-foreground line-through' : ''">
              {{ formatSlot(selected.start_time, selected.end_time) }}
            </span>
            <template v-if="isMoved && draftEvent">
              <ArrowRight class="size-3.5" />
              <span class="font-medium">
                {{ formatSlot(draftEvent.start as string, draftEvent.end as string) }}
              </span>
            </template>
          </span>
          <span class="text-muted-foreground text-xs">
            <template v-if="isAvailabilityLoaded && !hasAvailability">
              {{ teacherName(selected) }} has no free hours, so this class can only be removed.
            </template>
            <template v-else>
              Drag the class, or click a white hour, to move it. Grey hours are outside
              {{ teacherName(selected) }}'s free time.
            </template>
          </span>
        </div>
        <div class="flex shrink-0 gap-2">
          <Button variant="outline" :disabled="isSaving" @click="removing = selected">
            <Trash2 class="size-4" />
            Remove
          </Button>
          <Button variant="ghost" :disabled="isSaving" @click="clearSelection">Cancel</Button>
          <Button :disabled="!isMoved || isSaving" :aria-busy="isSaving" @click="saveTime">
            <Loader2 v-if="isSaving" class="animate-spin" />
            Save time
          </Button>
        </div>
      </div>
    </CardContent>
  </Card>

  <Dialog :open="removing !== null" @update:open="(open) => !open && (removing = null)">
    <DialogContent>
      <div class="flex flex-col gap-1">
        <DialogTitle>Remove this class?</DialogTitle>
        <DialogDescription v-if="removing">
          {{ subjectName(removing) }} with {{ teacherName(removing) }} at
          {{ branchName(removing) }}, {{ formatSlot(removing.start_time, removing.end_time) }}. This
          cannot be undone.
        </DialogDescription>
      </div>
      <div class="flex justify-end gap-2">
        <Button variant="ghost" :disabled="isRemoving" @click="removing = null">Cancel</Button>
        <Button
          variant="destructive"
          :disabled="isRemoving"
          :aria-busy="isRemoving"
          @click="confirmRemove"
        >
          <Loader2 v-if="isRemoving" class="animate-spin" />
          Remove
        </Button>
      </div>
    </DialogContent>
  </Dialog>
</template>
