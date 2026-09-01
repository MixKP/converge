<script setup lang="ts">
  import { ref, computed } from 'vue'
  import { Loader2, Plus, Sparkles, X } from '@lucide/vue'
  import { useBooking } from '../../composables/useBooking'
  import { useBookingContext } from '../../composables/useBookingContext'
  import { useAISuggestions } from '../../composables/useAISuggestions'
  import { useCart } from '../../composables/useCart'
  import { useNumberSelect, useEnumSelect, NONE } from '../../composables/useSelectProxy'
  import { weeklySlotSchema } from '../../schemas/calendar'
  import { toMinutes } from '../../utils/dateValidation'
  import BookingResults from '../BookingResults.vue'
  import { Button } from '@/components/ui/button'
  import { Card, CardContent } from '@/components/ui/card'
  import { Label } from '@/components/ui/label'
  import { Separator } from '@/components/ui/separator'
  import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
  } from '@/components/ui/select'
  import type { WeeklySlot } from '../../types'

  const {
    genderFilteredTeachers,
    selectedSubjectId,
    selectedBranchId,
    selectedTeacherId,
    requiredGender,
    suggestions,
    showDetailedResults,
    isEvaluating,
  } = useBooking()

  const { contextBlocker, contextComplete } = useBookingContext()
  const { getSuggestions } = useAISuggestions()
  const { cartItems, addSlotToCart } = useCart()

  const genderValue = useEnumSelect(requiredGender)
  const teacherValue = useNumberSelect(selectedTeacherId)

  const DAY_NAMES = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']

  const TIME_OPTIONS: string[] = []
  for (let hour = 8; hour <= 18; hour++) {
    const hourStr = String(hour).padStart(2, '0')
    TIME_OPTIONS.push(`${hourStr}:00`, `${hourStr}:30`)
  }
  TIME_OPTIONS.push('19:00')

  const timeSlots = ref<WeeklySlot[]>([])
  const draftDay = ref('1')
  const draftStart = ref('09:00')
  const draftEnd = ref('10:00')
  const draftError = ref('')

  const addTimeSlot = () => {
    const parsed = weeklySlotSchema.safeParse({
      day_of_week: Number(draftDay.value),
      start: draftStart.value,
      end: draftEnd.value,
    })
    if (!parsed.success) {
      draftError.value = parsed.error.issues[0]?.message || 'Invalid time slot'
      return
    }
    draftError.value = ''
    timeSlots.value = [...timeSlots.value, parsed.data]
    draftDay.value = String((Number(draftDay.value) + 1) % 7)
  }

  const removeTimeSlot = (index: number) => {
    timeSlots.value = timeSlots.value.filter((_, i) => i !== index)
  }

  /* The backend contract makes gender mandatory here even though it is only a
   * filter on the manual path — bookingRequestSchema requires required_gender,
   * and omitting it fails validation before the request is ever sent. */
  const submitBlocker = computed<string | null>(() => {
    if (!contextComplete.value) return contextBlocker.value
    if (requiredGender.value === null) return 'Choose a gender preference.'
    if (timeSlots.value.length === 0) return 'Add at least one preferred time window.'
    /* useAISuggestions derives duration_minutes from the first slot alone, so
     * mixed lengths would silently apply the first window's duration to all of
     * them. The old vee-validate schema blocked this; it is enforced here now
     * that the schema is gone. */
    const durations = new Set(timeSlots.value.map((s) => toMinutes(s.end) - toMinutes(s.start)))
    if (durations.size > 1) return 'All time windows must be the same length.'
    return null
  })

  const canSubmit = computed(() => submitBlocker.value === null && !isEvaluating.value)

  const handleSubmit = () => {
    if (!canSubmit.value) return
    getSuggestions(timeSlots.value)
  }

  const handleConfirmBooking = (
    teacherId: number,
    teacherName: string,
    startTime: string,
    endTime: string
  ) => {
    addSlotToCart(
      teacherId,
      teacherName,
      startTime,
      endTime,
      selectedSubjectId.value ?? undefined,
      selectedBranchId.value ?? undefined
    )
  }
</script>

<template>
  <Card>
    <CardContent class="flex flex-col gap-5 py-4">
      <div class="grid gap-4 sm:grid-cols-2">
        <div class="flex flex-col gap-2">
          <Label for="v3-smart-gender">Gender preference</Label>
          <Select v-model="genderValue" :disabled="!contextComplete">
            <SelectTrigger id="v3-smart-gender" class="w-full">
              <SelectValue placeholder="Select a preference" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="male">Male</SelectItem>
              <SelectItem value="female">Female</SelectItem>
              <SelectItem value="lgbtq+">LGBTQ+</SelectItem>
            </SelectContent>
          </Select>
          <p class="text-muted-foreground text-xs">Required by the matching engine.</p>
        </div>

        <div class="flex flex-col gap-2">
          <Label for="v3-smart-teacher">
            Preferred teacher
            <span class="text-muted-foreground font-normal">(optional)</span>
          </Label>
          <Select v-model="teacherValue" :disabled="!contextComplete">
            <SelectTrigger id="v3-smart-teacher" class="w-full">
              <SelectValue placeholder="Any teacher" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem :value="NONE">Any teacher</SelectItem>
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

      <Separator />

      <div class="flex flex-col gap-3">
        <Label>Preferred time windows</Label>
        <div class="flex flex-wrap items-center gap-2">
          <Select v-model="draftDay" :disabled="!contextComplete">
            <SelectTrigger class="w-36" aria-label="Day of week">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="(name, index) in DAY_NAMES" :key="name" :value="String(index)">
                {{ name }}
              </SelectItem>
            </SelectContent>
          </Select>

          <Select v-model="draftStart" :disabled="!contextComplete">
            <SelectTrigger class="w-28" aria-label="Start time">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="time in TIME_OPTIONS" :key="`s-${time}`" :value="time">
                {{ time }}
              </SelectItem>
            </SelectContent>
          </Select>

          <span class="text-muted-foreground text-sm">to</span>

          <Select v-model="draftEnd" :disabled="!contextComplete">
            <SelectTrigger class="w-28" aria-label="End time">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="time in TIME_OPTIONS" :key="`e-${time}`" :value="time">
                {{ time }}
              </SelectItem>
            </SelectContent>
          </Select>

          <Button
            variant="outline"
            size="icon"
            :disabled="!contextComplete"
            aria-label="Add time window"
            @click="addTimeSlot"
          >
            <Plus class="size-4" />
          </Button>
        </div>

        <p v-if="draftError" class="text-destructive text-xs" role="alert">{{ draftError }}</p>

        <div v-if="timeSlots.length" class="flex flex-col gap-1.5">
          <div
            v-for="(slot, index) in timeSlots"
            :key="`${slot.day_of_week}-${slot.start}-${slot.end}-${index}`"
            class="border-border flex items-center justify-between rounded-md border px-3 py-1.5 text-sm"
          >
            <span>{{ DAY_NAMES[slot.day_of_week] }} {{ slot.start }} – {{ slot.end }}</span>
            <Button
              variant="ghost"
              size="icon"
              :aria-label="`Remove ${DAY_NAMES[slot.day_of_week]} ${slot.start} to ${slot.end}`"
              @click="removeTimeSlot(index)"
            >
              <X class="size-4" />
            </Button>
          </div>
        </div>
      </div>

      <div class="flex items-center justify-between gap-3">
        <p v-if="submitBlocker" class="text-muted-foreground text-sm">{{ submitBlocker }}</p>
        <span v-else></span>
        <Button :disabled="!canSubmit" @click="handleSubmit">
          <Loader2 v-if="isEvaluating" class="size-4 animate-spin" />
          <Sparkles v-else class="size-4" />
          Find teachers
        </Button>
      </div>

      <!-- Results render in place rather than taking the viewport over, which is
           what the old modal did and why suggestions had nowhere to live. -->
      <template v-if="showDetailedResults || isEvaluating">
        <Separator />
        <BookingResults
          :suggestions="suggestions"
          :show-detailed-results="showDetailedResults"
          :is-evaluating="isEvaluating"
          :cart-items="cartItems"
          @confirm-booking="handleConfirmBooking"
          @reset="showDetailedResults = false"
        />
      </template>
    </CardContent>
  </Card>
</template>
