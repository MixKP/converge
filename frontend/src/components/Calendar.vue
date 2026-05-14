<script setup lang="ts">
  import { ref } from 'vue'
  import { VueCal, addDatePrototypes } from 'vue-cal'
  import 'vue-cal/style.css'
  import type { CalendarEvent } from '../types/calendar'
  import type { VueCalCellEvents } from 'vue-cal'

  addDatePrototypes()

  defineProps<{ editable: boolean }>()

  const events = ref<CalendarEvent[]>([])

  const createEvent = (data: VueCalCellEvents) => {
    const snapMinutes = 30
    const start = new Date(data.cursor.date)

    const minutes = start.getMinutes()
    const snappedMinutes = Math.round(minutes / snapMinutes) * snapMinutes
    start.setMinutes(snappedMinutes, 0, 0)

    const end = new Date(start.getTime() + 60 * 60 * 1000)

    events.value.push({
      title: '',
      start,
      end,
    })
  }

  const clearEvents = () => {
    events.value = []
  }

  defineExpose({ events, clearEvents })
</script>

<template>
  <div class="border border-slate-200 rounded-xl overflow-hidden">
    <vue-cal
      :events="events"
      :time-from="8 * 60"
      :time-to="19 * 60"
      :time-step="30"
      :today-button="false"
      :views-bar="false"
      :snap-to-interval="30"
      :event-create-min-drag="60"
      :editable-events="editable && { drag: true, resize: true, delete: true }"
      :disable-views="['years', 'year', 'month']"
      @cell-dblclick="createEvent"
      @cell-hold="createEvent"
    />
  </div>
</template>
