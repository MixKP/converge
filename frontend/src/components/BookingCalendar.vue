<script setup lang="ts">
  import { ref, nextTick } from 'vue'
  import { VueCal } from 'vue-cal'
  import 'vue-cal/style.css'
  import type { CalendarEvent } from '../types/calendar'
  import type { VueCalCellEvents } from 'vue-cal'

  interface Props {
    editable: boolean
    specialHours?: Record<
      string,
      Array<{ from: number; to: number; class?: string; allowEvents?: boolean }>
    >
  }

  const props = defineProps<Props>()
  const events = ref<CalendarEvent[]>([])

  const DAY_NAMES = ['sun', 'mon', 'tue', 'wed', 'thu', 'fri', 'sat'] as const

  // ─── Core: check if a time range overlaps any blocked zone ───────────────

  const isBlocked = (date: Date, startMin: number, endMin: number): boolean => {
    const dayName = DAY_NAMES[date.getDay()]
    const blocks = props.specialHours?.[dayName]
    if (!blocks) return false
    return blocks.some((b) => b.allowEvents === false && startMin < b.to && endMin > b.from)
  }

  const toMins = (d: Date) => d.getHours() * 60 + d.getMinutes()

  // ─── Create (dblclick / hold) ────────────────────────────────────────────

  const createEvent = (data: VueCalCellEvents) => {
    const start = new Date(data.cursor.date)
    start.setMinutes(Math.round(start.getMinutes() / 30) * 30, 0, 0)
    const end = new Date(start.getTime() + 60 * 60 * 1000)

    if (isBlocked(start, toMins(start), toMins(end))) return

    events.value.push({ title: '', start, end })
  }

  // ─── Create (drag-to-create) ─────────────────────────────────────────────

  const onEventCreate = (payload: any) => {
    const { event, resolve } = payload
    const start = new Date(event.start)
    const end = new Date(event.end)

    if (isBlocked(start, toMins(start), toMins(end))) return // don't resolve → VueCal discards

    events.value.push({ title: '', start, end })
    resolve()
  }

  // ─── Drag (move) ─────────────────────────────────────────────────────────

  const dragOrigin = ref<{ start: Date; end: Date; id: any } | null>(null)

  const onEventDragStart = (payload: any) => {
    const e = payload?.event ?? payload
    if (!e?.start) return
    dragOrigin.value = { start: new Date(e.start), end: new Date(e.end), id: e._?.id }
  }

  const onEventDrop = async (payload: any) => {
    if (!dragOrigin.value) return
    const e = payload?.event ?? payload
    if (!e?.start) return

    const start = new Date(e.start)
    const end = new Date(e.end)

    if (isBlocked(start, toMins(start), toMins(end))) {
      const { start: os, end: oe, id } = dragOrigin.value
      await nextTick()
      const idx = events.value.findIndex((ev) => (ev as any)._?.id === id)
      if (idx >= 0) {
        events.value[idx].start = new Date(os)
        events.value[idx].end = new Date(oe)
      }
    }

    dragOrigin.value = null
  }

  // ─── Resize ──────────────────────────────────────────────────────────────

  const resizeOrigin = ref<{ start: Date; end: Date; id: any } | null>(null)

  const onEventResizeStart = (payload: any) => {
    const e = payload?.event ?? payload
    if (!e?.start) return
    resizeOrigin.value = { start: new Date(e.start), end: new Date(e.end), id: e._?.id }
  }

  const onEventResize = async (payload: any) => {
    if (!resizeOrigin.value) return
    const e = payload?.event ?? payload
    if (!e?.start) return

    const start = new Date(e.start)
    const end = new Date(e.end)
    const startMin = toMins(start)
    const endMin = toMins(end)

    if (isBlocked(start, startMin, endMin)) {
      const { start: os, end: oe, id } = resizeOrigin.value
      const dayName = DAY_NAMES[start.getDay()]
      const blocks = props.specialHours?.[dayName] ?? []

      // Clamp end to the nearest blocked zone boundary
      const hitBlock = blocks
        .filter((b) => b.allowEvents === false && endMin > b.from && startMin < b.to)
        .sort((a, b) => a.from - b.from)[0]

      const clampedEnd = (() => {
        if (!hitBlock) return new Date(oe)
        const d = new Date(start)
        d.setHours(Math.floor(hitBlock.from / 60), hitBlock.from % 60, 0, 0)
        return d > start ? d : new Date(oe)
      })()

      await nextTick()
      const idx = events.value.findIndex((ev) => (ev as any)._?.id === id)
      if (idx >= 0) {
        events.value[idx].start = new Date(os)
        events.value[idx].end = clampedEnd
      }
    }

    resizeOrigin.value = null
  }

  // ─── Public API ──────────────────────────────────────────────────────────

  const clearEvents = () => {
    events.value = []
  }
  defineExpose({ events, clearEvents })
</script>

<template>
  <div class="border border-slate-200 rounded-xl overflow-hidden">
    <vue-cal
      :events="events"
      :special-hours="props.specialHours"
      :time-from="8 * 60"
      :time-to="19 * 60"
      :time-step="30"
      :today-button="false"
      :views-bar="false"
      :snap-to-interval="30"
      :event-create-min-drag="60"
      :editable-events="editable"
      :disable-views="['years', 'year', 'month']"
      @cell-dblclick="createEvent"
      @cell-hold="createEvent"
      @event-create="onEventCreate"
      @event-drag-start="onEventDragStart"
      @event-drop="onEventDrop"
      @event-resize-start="onEventResizeStart"
      @event-resize-end="onEventResize"
    />
  </div>
</template>

<style scoped>
  :deep(.vuecal__special-hours.blocked-time) {
    background-color: #f3f4f6;
    background-image: repeating-linear-gradient(
      45deg,
      transparent,
      transparent 10px,
      rgba(0, 0, 0, 0.03) 10px,
      rgba(0, 0, 0, 0.03) 20px
    );
    color: #9ca3af;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 0.8em;
  }

  :deep(.vuecal__special-hours.blocked-time::after) {
    content: 'Unavailable';
  }
</style>
