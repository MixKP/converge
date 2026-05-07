<script setup lang="ts">
  import { ref, reactive } from 'vue'
  import { VueCal, addDatePrototypes } from 'vue-cal'
  import 'vue-cal/style'

  addDatePrototypes()

  const selectedTeacher = ref()

  const teachers = ref([
    { name: 'Teacher A' },
    { name: 'Teacher B' },
    { name: 'Teacher C' },
    { name: 'Teacher D' },
  ])

  const events = reactive([
    {
      start: new Date(new Date().setHours(9, 0, 0, 0)),
      end: new Date(new Date().setHours(11, 30, 0, 0)),
      title: 'Teacher A',
    },
    {
      start: new Date(new Date().addDays(1).setHours(14, 0, 0, 0)),
      end: new Date(new Date().addDays(1).setHours(16, 0, 0, 0)),
      title: 'Teacher B',
    },
    {
      start: new Date(new Date().addDays(2).setHours(10, 0, 0, 0)),
      end: new Date(new Date().addDays(2).setHours(12, 0, 0, 0)),
      title: 'Teacher C',
    },
  ])

  const handleSubmit = () => {}
</script>

<template>
  <div class="no-scroll-page bg-slate-50 flex items-center justify-center p-4 font-sans">
    <div
      class="w-full max-w-5xl bg-white rounded-2xl shadow-lg border border-slate-200 p-5 flex flex-col max-h-full"
    >
      <h2 class="text-xl font-bold mb-3 text-slate-800 border-b pb-2">Submit Availability</h2>

      <form @submit.prevent="handleSubmit" class="flex flex-col gap-3 flex-1 min-h-0">
        <div class="flex flex-col gap-1">
          <label class="font-semibold text-slate-700 text-sm">1. Teacher</label>
          <select
            v-model="selectedTeacher"
            class="w-full md:w-1/2 border border-slate-300 text-slate-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-200 transition-colors cursor-pointer bg-white"
          >
            <option :value="undefined" disabled>-- Choose Your Name --</option>
            <option v-for="teacher in teachers" :key="teacher.name" :value="teacher">
              {{ teacher.name }}
            </option>
          </select>
        </div>

        <div class="flex flex-col gap-1">
          <label class="font-semibold text-slate-700 text-sm"> 2. Select Time Slot </label>
          <div class="border border-slate-200 rounded-xl overflow-hidden shadow-sm">
            <vue-cal
              :events="events"
              :time-from="8 * 60"
              :time-to="19 * 60"
              :time-step="30"
              :today-button="false"
              :views-bar="false"
              editable-events
            />
          </div>
        </div>

        <button
          type="submit"
          class="w-full md:w-auto self-end bg-slate-800 hover:bg-slate-900 text-white font-semibold py-2 px-6 rounded-lg shadow-sm transition-all active:scale-95 text-sm"
        >
          Submit Data
        </button>
      </form>
    </div>
  </div>
</template>
