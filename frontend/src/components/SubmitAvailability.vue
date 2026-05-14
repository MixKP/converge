<script setup lang="ts">
  import { ref, computed, onMounted } from 'vue'
  import { VueCal, addDatePrototypes } from 'vue-cal'
  import 'vue-cal/style.css'
  import { useTeacherStore } from '../stores/teacherStore'
  import type { CalendarEvent, EventCreateParams } from '../types/calendar'

  addDatePrototypes()

  const teacherStore = useTeacherStore()

  onMounted(() => {
    teacherStore.fetchTeachers()
  })

  const selectedTeacherName = computed({
    get: () => teacherStore.selectedTeacher,
    set: (val) => teacherStore.setSelectedTeacher(val ?? ''),
  })

  const events = ref<CalendarEvent[]>([])
  const notification = ref<{ type: 'success' | 'error'; message: string } | null>(null)

  const canSubmit = computed(() => !!selectedTeacherName.value && events.value.length > 0)

  // const formatDate = (date: Date) => {
  //   const d = new Date(date)
  //   const day = String(d.getDate()).padStart(2, '0')
  //   const month = String(d.getMonth() + 1).padStart(2, '0')
  //   const year = String(d.getFullYear()).slice(-2)
  //   return `${day}/${month}/${year}`
  // }

  // const generatePayload = (eventsList: CalendarEvent[], teacher: string) => {
  //   return eventsList.map((event) => ({
  //     teacher,
  //     date: formatDate(event.start),
  //     start: event.start.toTimeString().slice(0, 5),
  //     end: event.end.toTimeString().slice(0, 5),
  //   }))
  // }

  const generatePayload = (eventsList: CalendarEvent[], teacher: string) => {
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

    return {
      teacher: teacher,
      weekly: Array.from(weeklyMap.values()),
    }
  }

  const saveEventsToStore = (eventsList: CalendarEvent[], teacher: string) => {
    for (const event of eventsList) {
      const result = teacherStore.addBooking({
        teacher,
        startTime: event.start,
        endTime: event.end,
      })

      if (!result.success) {
        return { success: false, message: result.message }
      }
    }

    return { success: true }
  }

  const resetForm = () => {
    events.value = []
    teacherStore.setSelectedTeacher('')
  }

  const isTimeOverlap = (newEvent: CalendarEvent, existingEvents: CalendarEvent[]) => {
    return existingEvents.some((existing) => {
      return (
        newEvent.start.getTime() < existing.end.getTime() &&
        newEvent.end.getTime() > existing.start.getTime()
      )
    })
  }

  const createEvent = ({ event, resolve }: EventCreateParams) => {
    const hasConflict = isTimeOverlap(event, events.value)

    if (hasConflict) {
      notification.value = {
        type: 'error',
        message: 'Time slot overlaps with an existing booking. Please choose a different time.',
      }
      return
    }

    notification.value = null
    resolve(event)
  }

  const handleSubmit = () => {
    const teacher = selectedTeacherName.value
    if (!teacher) return

    const payload = generatePayload(events.value, teacher)
    console.log('Submit payload:', JSON.stringify(payload, null, 2))

    const result = saveEventsToStore(events.value, teacher)

    if (result.success) {
      resetForm()
      notification.value = { type: 'success', message: 'Availability submitted successfully!' }
    } else {
      notification.value = {
        type: 'error',
        message: result.message || 'An error occurred while saving.',
      }
    }
  }
</script>

<template>
  <div class="bg-slate-50 flex items-center justify-center p-4 font-sans min-h-screen">
    <div class="w-full max-w-5xl bg-white rounded-2xl shadow-lg border border-slate-200 p-5">
      <h2 class="text-xl font-bold mb-4 text-slate-800">Submit Availability</h2>

      <div
        v-if="notification"
        :class="[
          'mb-4 border px-4 py-3 rounded-lg text-sm',
          notification.type === 'success'
            ? 'bg-green-50 border-green-200 text-green-700'
            : 'bg-red-50 border-red-200 text-red-700',
        ]"
      >
        {{ notification.message }}
      </div>

      <form class="flex flex-col gap-4" @submit.prevent="handleSubmit">
        <div>
          <label class="font-semibold text-slate-700 text-sm block mb-1">Teacher</label>
          <select
            v-model="selectedTeacherName"
            class="w-full md:w-1/2 border border-slate-300 rounded-lg px-3 py-2 text-sm"
          >
            <option v-for="name in teacherStore.teacherNames" :key="name" :value="name">
              {{ name }}
            </option>
          </select>
        </div>

        <div>
          <label class="font-semibold text-slate-700 text-sm block mb-1">Time Slots</label>
          <div class="border border-slate-200 rounded-xl overflow-hidden">
            <vue-cal
              :events="events"
              :time-from="8 * 60"
              :time-to="19 * 60"
              :time-step="30"
              :today-button="false"
              :views-bar="false"
              :snap-to-interval="15"
              :event-create-min-drag="15"
              :editable-events="!!selectedTeacherName && { drag: true, resize: true, delete: true }"
              :disable-views="['years', 'year', 'month']"
              @event-create="createEvent"
            />
          </div>
        </div>

        <button
          type="submit"
          :disabled="!canSubmit"
          class="ml-auto bg-slate-800 hover:bg-slate-900 text-white font-semibold py-2 px-6 rounded-lg text-sm disabled:opacity-50 disabled:cursor-not-allowed"
        >
          Submit
        </button>
      </form>
    </div>
  </div>
</template>
