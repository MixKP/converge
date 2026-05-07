<script setup lang="ts">
  import { ref, computed } from 'vue'
  import { VueCal, addDatePrototypes } from 'vue-cal'
  import 'vue-cal/style.css'
  import { useTeacherStore } from '../stores/teacherStore'
  import type { CalendarEvent, CleanedEvent } from '../types/calendar'

  addDatePrototypes()

  const teacherStore = useTeacherStore()
  const selectedTeacherName = ref<string>()
  const events = ref<CalendarEvent[]>([])

  const canSubmit = computed(() => selectedTeacherName.value && events.value.length > 0)

  const createEvent = (params: { event: CalendarEvent; resolve: (e?: CalendarEvent) => void }) => {
    const { event, resolve } = params
    const exists = events.value.some(
      (e) => e.start.getTime() === event.start.getTime() && e.end.getTime() === event.end.getTime()
    )
    if (!exists) events.value.push(event)
    resolve()
  }

  const handleSubmit = () => {
    if (!selectedTeacherName.value) return

    const cleanedEvents: CleanedEvent[] = events.value.map((event) => ({
      Teacher: selectedTeacherName.value!,
      Date: event.start.toISOString().split('T')[0],
      start: event.start.toLocaleTimeString('en-US', {
        hour12: false,
        hour: '2-digit',
        minute: '2-digit',
      }),
      end: event.end.toLocaleTimeString('en-US', {
        hour12: false,
        hour: '2-digit',
        minute: '2-digit',
      }),
    }))

    events.value = []
    selectedTeacherName.value = undefined
    console.log(JSON.stringify(cleanedEvents, null, 2))
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
            v-model="selectedTeacherName"
            class="w-full md:w-1/2 border border-slate-300 rounded-lg px-3 py-2 text-sm"
          >
            <option :value="undefined" disabled>-- Choose Your Name --</option>
            <option v-for="name in teacherStore.teachers" :key="name" :value="name">
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
              :editable-events="
                !!selectedTeacherName && { drag: false, resize: false, delete: false }
              "
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
