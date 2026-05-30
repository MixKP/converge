<script setup lang="ts">
  import { ref, computed, onMounted, watch } from 'vue'
  import type { EventInput, BusinessHoursInput } from '@fullcalendar/core'
  import { useTeacherStore } from '../stores/teacherStore'
  import { availabilityApi } from '../services/availabilityApi'
  import { subjectApi } from '../services/subjectApi'
  import { teacherApi } from '../services/teacherApi'
  import type { Teacher } from '../types'
  import PageLayout from '../components/PageLayout.vue'
  import Calendar from '../components/Calendar.vue'
  import CalendarDisabledOverlay from '../components/CalendarDisabledOverlay.vue'
  import SubmitButton from '../components/SubmitButton.vue'
  import type { WeeklySlot, Subject } from '../types'
  import { generateAvailabilityPayload } from '../utils/calendarHelpers'
  import { transformBackendAvailability } from '../utils/availabilityTransform'

  const teacherStore = useTeacherStore()
  const calendarRef = ref<InstanceType<typeof Calendar>>()
  const isLoading = ref(false)
  const isLoadingAvailability = ref(false)
  const isLoadingTeachers = ref(false)
  const events = ref<EventInput[]>([])
  const businessHours = ref<BusinessHoursInput>([])
  const availabilityCache = ref<Map<number, WeeklySlot[]>>(new Map())
  const subjects = ref<Subject[]>([])
  const filteredTeachers = ref<Teacher[]>([])
  const selectedSubjectId = ref<number | null>(null)

  const fetchAvailability = async () => {
    try {
      isLoadingAvailability.value = true
      const data = await availabilityApi.getAll()
      availabilityCache.value = transformBackendAvailability(data)
    } finally {
      isLoadingAvailability.value = false
    }
  }
  onMounted(async () => {
    subjects.value = await subjectApi.getAll()
    fetchAvailability()
  })
  const fetchTeachersBySubject = async (subjectId: number) => {
    try {
      isLoadingTeachers.value = true
      filteredTeachers.value = await teacherApi.getBySubject(subjectId)
    } finally {
      isLoadingTeachers.value = false
    }
  }
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
  const updateBusinessHours = (teacherId: number) => {
    const availability = availabilityCache.value.get(teacherId)
    if (!availability) {
      businessHours.value = []
      return
    }
    businessHours.value = availability.map((avail) => ({
      daysOfWeek: [avail.day_of_week + 1],
      startTime: avail.start,
      endTime: avail.end,
    }))
    calendarRef.value?.setOption('businessHours', businessHours.value)
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
  watch(selectedSubjectId, (newSubjectId) => {
    events.value = []
    teacherStore.setSelectedTeacherById(null)
    if (newSubjectId) {
      fetchTeachersBySubject(newSubjectId)
    } else {
      filteredTeachers.value = []
    }
  })
  watch(selectedTeacherId, (newId) => {
    events.value = []
    if (newId) {
      updateBusinessHours(newId)
    } else {
      businessHours.value = []
    }
  })
</script>
<template>
  <PageLayout title="Booking Course">
    <form class="flex flex-col gap-3 sm:gap-4 flex-1 min-h-0" @submit.prevent="handleSubmit">
      <div class="flex flex-col sm:flex-row gap-3 sm:gap-5 shrink-0 stagger-in">
        <div
          class="bg-(--paper-white) p-4 sm:p-5 rounded-sm border border-(--border-subtle) flex-1 shadow-card"
        >
          <label
            class="font-semibold text-(--ink-primary) text-sm block mb-2 sm:mb-3 tracking-tight"
          >
            <span
              class="font-mono text-xs border-2 border-(--ink-primary) bg-(--paper-cream) px-2.5 sm:px-3 py-1 rounded-sm tracking-widest mr-2 sm:mr-3 shadow-badge inline-block"
              >01</span
            >
            Select Subject
          </label>
          <select
            v-model="selectedSubjectId"
            class="w-full bg-white border border-(--border-strong) rounded-sm px-3 sm:px-4 py-2.5 sm:py-3 text-sm focus:ring-2 focus:ring-(--accent-terracotta)/10 focus:border-(--accent-terracotta) outline-none transition-all text-(--ink-primary) cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed hover:border-(--text-secondary)"
          >
            <option :value="null" disabled>-- Choose Subject --</option>
            <option v-for="subject in subjects" :key="subject.id" :value="subject.id">
              {{ subject.name }}
            </option>
          </select>
        </div>
        <div
          class="bg-(--paper-white) p-4 sm:p-5 rounded-sm border border-(--border-subtle) flex-1 shadow-card"
        >
          <label
            class="font-semibold text-(--ink-primary) text-sm block mb-2 sm:mb-3 tracking-tight"
          >
            <span
              class="font-mono text-xs border-2 border-(--ink-primary) bg-(--paper-cream) px-2.5 sm:px-3 py-1 rounded-sm tracking-widest mr-2 sm:mr-3 shadow-badge inline-block"
              >02</span
            >
            Select Teacher
          </label>
          <select
            v-model="selectedTeacherId"
            :disabled="!canSelectTeacher || isLoadingTeachers"
            class="w-full bg-white border border-(--border-strong) rounded-sm px-3 sm:px-4 py-2.5 sm:py-3 text-sm focus:ring-2 focus:ring-(--accent-terracotta)/10 focus:border-(--accent-terracotta) outline-none transition-all text-(--ink-primary) cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed hover:border-(--text-secondary)"
          >
            <option :value="null" disabled>
              {{
                isLoadingTeachers
                  ? '-- Loading...'
                  : canSelectTeacher
                    ? '-- Choose Teacher --'
                    : '-- Select subject first --'
              }}
            </option>
            <option v-for="teacher in filteredTeachers" :key="teacher.id" :value="teacher.id">
              {{ teacher.name }}
            </option>
            <option
              v-if="canSelectTeacher && !isLoadingTeachers && filteredTeachers.length === 0"
              disabled
            >
              -- No teachers available for this subject --
            </option>
          </select>
        </div>
      </div>
      <div class="flex-1 flex flex-col min-h-0 stagger-in">
        <div class="flex items-center justify-between mb-3 sm:mb-4 shrink-0">
          <label class="font-semibold text-(--ink-primary) text-sm tracking-tight">
            <span
              class="font-mono text-xs border-2 border-(--ink-primary) bg-(--paper-cream) px-2.5 sm:px-3 py-1 rounded-sm tracking-widest mr-2 sm:mr-3 shadow-badge inline-block"
              >03</span
            >
            Select Time Slots
          </label>
          <span
            v-if="selectedTeacherId"
            class="text-xs font-medium text-(--accent-terracotta) bg-(--accent-terracotta-soft) px-3 sm:px-4 py-1 sm:py-1.5 rounded-sm border border-(--accent-terracotta)/25 tracking-wide hidden sm:inline-block"
          >
            Click and drag to select
          </span>
        </div>
        <div
          class="relative border border-(--border-subtle) rounded-sm overflow-hidden shadow-card bg-(--paper-white) flex-1 min-h-0"
        >
          <div
            :class="{
              'opacity-45 grayscale-30 pointer-events-none transition-opacity duration-300':
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
      <div
        class="flex justify-end pt-4 sm:pt-5 border-t border-(--border-subtle) shrink-0 stagger-in"
      >
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
