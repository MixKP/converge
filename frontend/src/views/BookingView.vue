<script setup lang="ts">
  import { ref, computed, onMounted, watch } from 'vue'
  import type { EventInput, BusinessHoursInput } from '@fullcalendar/core'
  import { useTeacherStore } from '../stores/teacherStore'
  import { availabilityApi } from '../services/availabilityApi'
  import PageLayout from '../components/PageLayout.vue'
  import Calendar from '../components/Calendar.vue'
  import CalendarDisabledOverlay from '../components/CalendarDisabledOverlay.vue'
  import SubmitButton from '../components/SubmitButton.vue'
  import type { WeeklySlot, Subject } from '../types'
  import { generateAvailabilityPayload } from '../utils/calendarHelpers'

  const teacherStore = useTeacherStore()
  const calendarRef = ref<InstanceType<typeof Calendar>>()

  const isLoading = ref(false)
  const events = ref<EventInput[]>([])
  const businessHours = ref<BusinessHoursInput>({})

  const subjects = ref<Subject[]>([
    { id: 1, name: 'Mathematics' },
    { id: 2, name: 'Physics' },
    { id: 3, name: 'Chemistry' },
    { id: 4, name: 'Biology' },
    { id: 5, name: 'English' },
    { id: 6, name: 'Thai Language' },
  ])
  const selectedSubjectId = ref<number | null>(null)

  const selectedTeacherId = computed({
    get: () => teacherStore.selectedTeacherId,
    set: (val) => teacherStore.setSelectedTeacherById(val),
  })

  const canSubmit = computed(
    () =>
      !!selectedSubjectId.value &&
      !!selectedTeacherId.value &&
      !isLoading.value &&
      events.value.length > 0
  )

  const canSelectTeacher = computed(() => !!selectedSubjectId.value)

  const generateBusinessHours = (availabilityList: WeeklySlot[]): BusinessHoursInput => {
    const result: BusinessHoursInput = []
    for (const avail of availabilityList) {
      result.push({
        daysOfWeek: [avail.day_of_week],
        startTime: avail.start,
        endTime: avail.end,
      })
    }
    return result
  }

  const fetchAvailability = async (_teacherId: number) => {
    const mockAvailability: WeeklySlot[] = [
      { day_of_week: 1, start: '09:00', end: '12:00' },
      { day_of_week: 1, start: '14:00', end: '17:00' },
      { day_of_week: 3, start: '10:00', end: '12:00' },
      { day_of_week: 5, start: '09:00', end: '12:00' },
      { day_of_week: 0, start: '10:00', end: '12:00' },
    ]

    businessHours.value = generateBusinessHours(mockAvailability)
    if (calendarRef.value) {
      calendarRef.value.setOption('businessHours', businessHours.value)
    }
  }

  const handleSubmit = async () => {
    if (!selectedTeacherId.value || isLoading.value) return

    isLoading.value = true
    try {
      const payload = generateAvailabilityPayload(events.value, selectedTeacherId.value)
      await availabilityApi.submitAvailability(payload)

      events.value = []
      teacherStore.setSelectedTeacherById(null)
    } finally {
      isLoading.value = false
    }
  }

  onMounted(() => {
    teacherStore.fetchTeachers()
  })

  watch(selectedSubjectId, () => {
    events.value = []
    teacherStore.setSelectedTeacherById(null)
  })

  watch(selectedTeacherId, (newId) => {
    events.value = []
    if (newId) {
      fetchAvailability(newId)
    } else {
      businessHours.value = {}
    }
  })
</script>

<template>
  <PageLayout title="Booking Course">
    <form class="flex flex-col gap-3 flex-1 min-h-0" @submit.prevent="handleSubmit">
      <div class="flex gap-3 shrink-0">
        <div class="bg-slate-50/50 p-3 rounded-lg border border-slate-100 flex-1">
          <label class="font-semibold text-slate-700 text-sm block mb-1">
            <span class="bg-blue-100 text-blue-700 px-2 py-0.5 rounded text-xs mr-1">1</span>
            Select Subject
          </label>
          <select
            v-model="selectedSubjectId"
            class="w-full bg-white border border-slate-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 outline-none transition-all text-slate-700 shadow-sm cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <option :value="null" disabled>-- Choose Subject --</option>
            <option v-for="subject in subjects" :key="subject.id" :value="subject.id">
              {{ subject.name }}
            </option>
          </select>
        </div>

        <div class="bg-slate-50/50 p-3 rounded-lg border border-slate-100 flex-1">
          <label class="font-semibold text-slate-700 text-sm block mb-1">
            <span class="bg-blue-100 text-blue-700 px-2 py-0.5 rounded text-xs mr-1">2</span>
            Select Teacher
          </label>
          <select
            v-model="selectedTeacherId"
            :disabled="!canSelectTeacher"
            class="w-full bg-white border border-slate-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 outline-none transition-all text-slate-700 shadow-sm cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <option :value="null" disabled>
              {{ canSelectTeacher ? '-- Choose Teacher --' : '-- Select subject first --' }}
            </option>
            <option v-for="teacher in teacherStore.teachers" :key="teacher.id" :value="teacher.id">
              {{ teacher.name }}
            </option>
          </select>
        </div>
      </div>

      <div class="flex-1 flex flex-col min-h-0">
        <div class="flex items-center justify-between mb-2 shrink-0">
          <label class="font-semibold text-slate-700 text-sm">
            <span class="bg-blue-100 text-blue-700 px-2 py-0.5 rounded text-xs mr-1">3</span>
            Select Time Slots
          </label>
          <span
            v-if="selectedTeacherId"
            class="text-xs font-medium text-blue-600 bg-blue-50 px-2 py-0.5 rounded border border-blue-100"
          >
            Click and drag to select
          </span>
        </div>

        <div
          class="relative border border-slate-200 rounded-lg overflow-hidden shadow-sm bg-white flex-1 min-h-0"
        >
          <div
            :class="{
              'opacity-40 grayscale-30% pointer-events-none transition-opacity duration-300':
                !selectedTeacherId,
            }"
            class="h-full"
          >
            <Calendar
              ref="calendarRef"
              v-model="events"
              :editable="!!selectedTeacherId"
              :business-hours="businessHours"
              constraint="businessHours"
              class="h-full"
            />
          </div>

          <CalendarDisabledOverlay
            v-if="!selectedTeacherId"
            :message="
              !selectedSubjectId ? 'Select subject and teacher first' : 'Select a teacher first'
            "
          />
        </div>
      </div>

      <div class="flex justify-end pt-2 border-t border-slate-100 shrink-0">
        <SubmitButton
          :is-disabled="!canSubmit"
          :is-loading="isLoading"
          loading-text="Submitting..."
          normal-text="Confirm Booking"
        />
      </div>
    </form>
  </PageLayout>
</template>
