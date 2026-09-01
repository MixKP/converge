<script setup lang="ts">
  import { ref, computed, watch } from 'vue'
  import { CalendarOff, Eye, Loader2, X } from '@lucide/vue'
  import { useBooking } from '../../composables/useBooking'
  import { useBookingContext } from '../../composables/useBookingContext'
  import {
    useBrowseEvents,
    type BrowseTeacher,
    type VisibleRange,
  } from '../../composables/useBrowseEvents'
  import { useCart } from '../../composables/useCart'
  import { useNotification } from '../../composables/useNotification'
  import { useNumberSelect, useEnumSelect, NONE } from '../../composables/useSelectProxy'
  import Calendar from '../Calendar.vue'
  import { Button } from '@/components/ui/button'
  import { Card, CardContent } from '@/components/ui/card'
  import { Label } from '@/components/ui/label'
  import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
  } from '@/components/ui/select'
  import type { EventClickArg } from '@fullcalendar/core'

  const {
    events,
    businessHours,
    allEvents,
    genderFilteredTeachers,
    isLoadingTeachers,
    selectedSubjectId,
    selectedBranchId,
    selectedTeacherId,
    requiredGender,
  } = useBooking()

  const { contextBlocker, contextComplete } = useBookingContext()

  // Browse blocks are built per visible week so they can be reconciled against
  // dated bookings, so the calendar has to say which week it is showing.
  const visibleRange = ref<VisibleRange | null>(null)
  const { browseEvents } = useBrowseEvents(() => visibleRange.value)
  const { addSlotToCart } = useCart()
  const { showSuccess } = useNotification()

  const teacherValue = useNumberSelect(selectedTeacherId)
  const genderValue = useEnumSelect(requiredGender)

  const isAddingToCart = ref(false)
  const pendingTeachers = ref<BrowseTeacher[] | null>(null)

  /* Three honest states. The old page rendered a live-looking calendar whenever
   * a subject was set, but only made it editable once a branch AND a teacher
   * were chosen — so dragging in between silently did nothing. */
  const calendarState = computed<'locked' | 'browse' | 'editable'>(() => {
    if (!contextComplete.value) return 'locked'
    return selectedTeacherId.value === null ? 'browse' : 'editable'
  })

  const selectedTeacher = computed(
    () => genderFilteredTeachers.value.find((t) => t.id === selectedTeacherId.value) ?? null
  )

  const hasAvailability = computed(
    () => Array.isArray(businessHours.value) && businessHours.value.length > 0
  )

  const canAddToBooking = computed(
    () => calendarState.value === 'editable' && events.value.length > 0 && !isAddingToCart.value
  )

  watch(calendarState, (state) => {
    if (state !== 'browse') pendingTeachers.value = null
  })

  const pickTeacher = (teacher: BrowseTeacher) => {
    selectedTeacherId.value = teacher.id
    pendingTeachers.value = null
    showSuccess(`Now booking with ${teacher.name} — drag the calendar to select times`, 4000)
  }

  /* Clicking a browse block promotes you into the editable state. The store's
   * watcher on selectedTeacherId narrows businessHours to that teacher, so the
   * calendar swaps from union shading to their hours with no extra work here. */
  const handleEventClick = (info: EventClickArg) => {
    const props = info.event.extendedProps
    if (!props?.isBrowse) return

    const teachers = props.teachers as BrowseTeacher[]
    if (teachers.length === 1) {
      pickTeacher(teachers[0])
      return
    }
    pendingTeachers.value = teachers
  }

  const addToBooking = () => {
    const teacher = selectedTeacher.value
    if (!teacher || !canAddToBooking.value) return

    isAddingToCart.value = true
    try {
      const slots = [...events.value]
      slots.forEach((slot) => {
        addSlotToCart(
          teacher.id,
          teacher.name,
          slot.start as string,
          slot.end as string,
          selectedSubjectId.value ?? undefined,
          selectedBranchId.value ?? undefined
        )
      })
      events.value = []
      showSuccess(`Added ${slots.length} slot${slots.length === 1 ? '' : 's'} to the cart`, 3000)
    } finally {
      isAddingToCart.value = false
    }
  }
</script>

<template>
  <Card>
    <CardContent class="flex flex-col gap-4 py-4">
      <div class="grid gap-4 sm:grid-cols-2">
        <div class="flex flex-col gap-2">
          <Label for="v3-gender">
            Gender preference
            <span class="text-muted-foreground font-normal">(optional)</span>
          </Label>
          <Select v-model="genderValue">
            <SelectTrigger id="v3-gender" class="w-full">
              <SelectValue placeholder="Any" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem :value="NONE">Any</SelectItem>
              <SelectItem value="male">Male</SelectItem>
              <SelectItem value="female">Female</SelectItem>
              <SelectItem value="lgbtq+">LGBTQ+</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div class="flex flex-col gap-2">
          <Label for="v3-teacher">Teacher</Label>
          <Select v-model="teacherValue" :disabled="isLoadingTeachers || !contextComplete">
            <SelectTrigger id="v3-teacher" class="w-full">
              <SelectValue
                :placeholder="isLoadingTeachers ? 'Loading teachers…' : 'All teachers'"
              />
            </SelectTrigger>
            <SelectContent>
              <SelectItem :value="NONE">All teachers</SelectItem>
              <SelectItem
                v-for="teacher in genderFilteredTeachers"
                :key="teacher.id"
                :value="String(teacher.id)"
              >
                {{ teacher.name }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      <!-- Locked: the calendar is not mounted at all, so there is nothing that
           looks draggable but isn't. -->
      <div
        v-if="calendarState === 'locked'"
        class="border-border flex min-h-[20rem] flex-col items-center justify-center gap-3 rounded-lg border-2 border-dashed"
      >
        <CalendarOff class="text-muted-foreground size-7" />
        <p class="text-sm font-medium">{{ contextBlocker }}</p>
        <p class="text-muted-foreground text-xs">
          Availability appears once the details above are set.
        </p>
      </div>

      <div v-else class="flex flex-col gap-3">
        <div
          v-if="calendarState === 'browse'"
          class="border-border bg-muted/40 text-muted-foreground flex items-center gap-2 rounded-md border px-3 py-2 text-xs"
        >
          <Eye class="size-4 shrink-0" />
          <span>
            Read-only — showing when every matching teacher is free. Pick a teacher above, or click
            a block below, to start selecting times.
          </span>
        </div>

        <div class="relative">
          <!-- Tall enough on desktop to show 08:00–19:00 without scrolling; the
               grid used to open scrolled past 9am, hiding the block labels. -->
          <div class="border-border h-[30rem] overflow-hidden rounded-lg border lg:h-[44rem]">
            <Calendar
              :model-value="allEvents"
              :additional-events="calendarState === 'browse' ? browseEvents : []"
              :editable="calendarState === 'editable'"
              :show-header="false"
              :business-hours="businessHours"
              constraint="businessHours"
              @update:model-value="events = $event"
              @event-click="handleEventClick"
              @dates-set="visibleRange = $event"
            />
          </div>

          <!-- Identical weekly windows collapse into one block, so a click on a
               shared range has to disambiguate before selecting a teacher. -->
          <div
            v-if="pendingTeachers"
            class="bg-background/80 absolute inset-0 z-20 flex items-center justify-center rounded-lg p-4"
          >
            <Card class="w-full max-w-xs">
              <CardContent class="flex flex-col gap-2 py-4">
                <div class="flex items-center justify-between">
                  <p class="text-sm font-medium">Which teacher?</p>
                  <Button
                    variant="ghost"
                    size="icon"
                    aria-label="Cancel teacher selection"
                    @click="pendingTeachers = null"
                  >
                    <X class="size-4" />
                  </Button>
                </div>
                <Button
                  v-for="teacher in pendingTeachers"
                  :key="teacher.id"
                  variant="outline"
                  class="justify-start"
                  @click="pickTeacher(teacher)"
                >
                  {{ teacher.name }}
                </Button>
              </CardContent>
            </Card>
          </div>
        </div>

        <div
          v-if="calendarState === 'editable' && !hasAvailability"
          class="border-destructive/30 bg-destructive/5 text-destructive rounded-md border px-3 py-2 text-sm"
        >
          {{ selectedTeacher?.name ?? 'This teacher' }} has no availability submitted yet, so there
          are no bookable hours. Pick a different teacher.
        </div>

        <div class="flex items-center justify-between gap-3">
          <p class="text-muted-foreground text-sm">
            <template v-if="calendarState === 'editable'">
              {{ events.length }} slot{{ events.length === 1 ? '' : 's' }} selected
            </template>
            <template v-else>Select a teacher to start booking</template>
          </p>
          <Button :disabled="!canAddToBooking" @click="addToBooking">
            <Loader2 v-if="isAddingToCart" class="size-4 animate-spin" />
            Add to booking
          </Button>
        </div>
      </div>
    </CardContent>
  </Card>
</template>
