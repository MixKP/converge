<script setup lang="ts">
  import { ref, computed, onMounted } from 'vue'
  import { useTeacherStore } from '../stores/teacherStore'
  import { availabilityApi } from '../services/availabilityApi'
  import Calendar from '../components/Calendar.vue'
  import type { CalendarEvent } from '../types/calendar'

  const teacherStore = useTeacherStore()
  const calendarRef = ref<InstanceType<typeof Calendar>>()

  const isLoading = ref(false)

  const selectedTeacherId = computed({
    get: () => teacherStore.selectedTeacherId,
    set: (val) => teacherStore.setSelectedTeacherById(val),
  })

  const canSubmit = computed(
    () =>
      !!selectedTeacherId.value &&
      !isLoading.value &&
      calendarRef.value?.events &&
      calendarRef.value.events.length > 0
  )

  onMounted(() => {
    teacherStore.fetchTeachers()
  })

  const generatePayload = (eventsList: CalendarEvent[], teacherId: number) => {
    const weeklyMap = new Map<string, { day_of_week: number; start: string; end: string }>()

    for (const event of eventsList) {
      const dayOfWeek = event.start.getDay()
      const start = event.start.toTimeString().slice(0, 5)
      const end = event.end.toTimeString().slice(0, 5)
      const key = `${dayOfWeek}-${start}-${end}`
      if (!weeklyMap.has(key)) {
        weeklyMap.set(key, { day_of_week: dayOfWeek, start, end })
      }
    }

    return { teacher_id: teacherId, weekly: Array.from(weeklyMap.values()) }
  }

  const handleSubmit = async () => {
    if (!selectedTeacherId.value || !calendarRef.value?.events || isLoading.value) return

    isLoading.value = true

    try {
      const payload = generatePayload(calendarRef.value.events, selectedTeacherId.value)
      await availabilityApi.submitAvailability(payload)

      calendarRef.value.clearEvents()
      teacherStore.setSelectedTeacherById(null)
    } finally {
      isLoading.value = false
    }
  }
</script>

<template>
  <div class="bg-slate-50 flex items-center justify-center p-4 font-sans min-h-screen">
    <div class="w-full max-w-5xl bg-white rounded-2xl shadow-lg border border-slate-200 p-5">
      <h2 class="text-xl font-bold mb-4 text-slate-800">Submit Availability</h2>

      <form class="flex flex-col gap-4" @submit.prevent="handleSubmit">
        <div>
          <label class="font-semibold text-slate-700 text-sm block mb-1">Teacher</label>
          <select
            v-model="selectedTeacherId"
            class="w-full md:w-1/2 border border-slate-300 rounded-lg px-3 py-2 text-sm"
          >
            <option :value="null" disabled>-- Choose Your Name --</option>
            <option v-for="teacher in teacherStore.teachers" :key="teacher.id" :value="teacher.id">
              {{ teacher.name }}
            </option>
          </select>
        </div>

        <div>
          <label class="font-semibold text-slate-700 text-sm block mb-1">Time Slots</label>
          <Calendar ref="calendarRef" :editable="!!selectedTeacherId" />
        </div>

        <button
          type="submit"
          :disabled="!canSubmit"
          class="ml-auto bg-slate-800 hover:bg-slate-900 text-white font-semibold py-2 px-6 rounded-lg text-sm disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {{ isLoading ? 'Submitting...' : 'Submit' }}
        </button>
      </form>
    </div>
  </div>
</template>
