<script setup lang="ts">
  import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
  import FullCalendar from '@fullcalendar/vue3'
  import timeGridPlugin from '@fullcalendar/timegrid'
  import interactionPlugin from '@fullcalendar/interaction'
  import type {
    EventInput,
    DateSelectArg,
    EventClickArg,
    EventDropArg,
    BusinessHoursInput,
    CalendarOptions,
  } from '@fullcalendar/core'
  interface Props {
    editable?: boolean
    businessHours?: BusinessHoursInput
    constraint?: string | 'businessHours'
    modelValue?: EventInput[]
  }
  const props = withDefaults(defineProps<Props>(), {
    editable: false,
    businessHours: undefined,
    constraint: undefined,
    modelValue: () => [],
  })
  const emit = defineEmits<{
    'update:modelValue': [value: EventInput[]]
  }>()
  const calendarRef = ref<InstanceType<typeof FullCalendar> | null>(null)
  const screenWidth = ref(0)
  const updateScreenWidth = () => {
    screenWidth.value = globalThis.innerWidth
  }
  onMounted(() => {
    updateScreenWidth()
    globalThis.addEventListener('resize', updateScreenWidth)
  })
  onUnmounted(() => {
    globalThis.removeEventListener('resize', updateScreenWidth)
  })
  const dayHeaderFormat = computed(() => {
    if (screenWidth.value < 640) {
      return { weekday: 'narrow' as const }
    } else if (screenWidth.value < 768) {
      return { weekday: 'short' as const }
    }
    return { weekday: 'long' as const }
  })
  const showDeleteDialog = ref(false)
  const eventToDelete = ref<string | null>(null)
  const cancelBtnRef = ref<HTMLButtonElement | null>(null)
  const previouslyFocused = ref<HTMLElement | null>(null)
  const handleEscapeKey = () => {
    closeDeleteDialog()
  }
  const generateEventId = (): string => {
    return `event-${Date.now()}-${Math.random().toString(36).slice(2, 11)}`
  }
  const getCurrentEvents = (): EventInput[] => {
    const api = calendarRef.value?.getApi()
    if (!api) return []
    return api
      .getEvents()
      .filter((e) => e.start != null && e.end != null)
      .map((e) => ({
        id: e.id,
        start: e.start!,
        end: e.end!,
        title: e.title,
      }))
  }
  const handleDateSelect = (selectInfo: DateSelectArg) => {
    if (!props.editable) return
    const calendar = selectInfo.view.calendar
    const start = selectInfo.start
    const end = selectInfo.end
    calendar.unselect()
    const newEvent = { id: generateEventId(), start, end, title: '' }
    emit('update:modelValue', [...(props.modelValue || []), newEvent])
  }
  const handleEventClick = (clickInfo: EventClickArg) => {
    const id = clickInfo.event.id
    if (!id) return
    eventToDelete.value = id
    showDeleteDialog.value = true
  }
  const confirmDelete = () => {
    if (eventToDelete.value) {
      const filteredEvents = (props.modelValue || []).filter((e) => e.id !== eventToDelete.value)
      emit('update:modelValue', filteredEvents)
    }
    closeDeleteDialog()
  }
  const closeDeleteDialog = () => {
    showDeleteDialog.value = false
    eventToDelete.value = null
  }
  const handleEventChange = () => {
    emit('update:modelValue', getCurrentEvents())
  }
  const handleEventDrop = (_info: EventDropArg) => {
    handleEventChange()
  }
  const handleEventResize = () => {
    handleEventChange()
  }
  const CALENDAR_DEFAULT_OPTIONS = {
    initialView: 'timeGridWeek',
    headerToolbar: false,
    height: '100%',
    weekends: true,
    allDaySlot: false,
    selectMirror: true,
    dayMaxEvents: true,
    nowIndicator: true,
    slotMinTime: '08:00',
    slotMaxTime: '19:00',
    slotDuration: '00:30:00',
    snapDuration: '01:00:00',
  } as const
  watch(
    () => props.modelValue,
    (newEvents, oldEvents) => {
      const api = calendarRef.value?.getApi()
      if (!api) return
      const oldIds = new Set(
        (oldEvents ?? []).map((e) => e.id).filter((id): id is string => id != null)
      )
      const newIds = new Set(
        (newEvents ?? []).map((e) => e.id).filter((id): id is string => id != null)
      )
      for (const id of oldIds) {
        if (!newIds.has(id)) {
          api.getEventById(id)?.remove()
        }
      }
      for (const event of newEvents ?? []) {
        if (!event.id) continue
        const existing = api.getEventById(event.id)
        if (!existing) {
          api.addEvent(event)
        }
      }
    },
    { immediate: true }
  )
  const calendarOptions = computed(() => ({
    plugins: [timeGridPlugin, interactionPlugin],
    ...CALENDAR_DEFAULT_OPTIONS,
    editable: props.editable,
    selectable: props.editable,
    businessHours: props.businessHours,
    eventConstraint: props.constraint,
    selectConstraint: props.constraint,
    select: handleDateSelect,
    eventClick: handleEventClick,
    eventDrop: handleEventDrop,
    eventResize: handleEventResize,
    dayHeaderFormat: dayHeaderFormat.value,
  }))
  watch(showDeleteDialog, (isOpen) => {
    if (isOpen) {
      previouslyFocused.value = document.activeElement as HTMLElement
      cancelBtnRef.value?.focus()
    } else {
      previouslyFocused.value?.focus()
      previouslyFocused.value = null
    }
  })
  const setOption = <K extends keyof CalendarOptions>(key: K, value: CalendarOptions[K]) => {
    const api = calendarRef.value?.getApi()
    if (api) {
      api.setOption(key, value)
    }
  }
  defineExpose({ setOption })
</script>
<template>
  <div
    class="border border-(--border-subtle) rounded-sm overflow-hidden h-full bg-(--paper-white) shadow-sm"
  >
    <FullCalendar ref="calendarRef" :options="calendarOptions" />
    <Transition
      enter-active-class="transition-opacity duration-200"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition-opacity duration-150"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="showDeleteDialog"
        class="fixed inset-0 z-50 flex items-center justify-center bg-(--ink-primary)/40 backdrop-blur-sm"
        role="dialog"
        aria-modal="true"
        aria-labelledby="deleteDialogTitle"
        @keydown.esc="handleEscapeKey"
      >
        <div
          class="transition-all duration-300 ease-out scale-in bg-(--paper-white) rounded-sm shadow-2xl border border-(--border-strong) p-6 w-full max-w-sm mx-4"
        >
          <div class="flex items-start gap-4 mb-5">
            <div
              class="shrink-0 w-12 h-12 rounded-sm bg-red-50 flex items-center justify-center border border-red-100"
            >
              <svg
                class="w-6 h-6 text-red-600"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                />
              </svg>
            </div>
            <div class="flex-1">
              <h3
                id="deleteDialogTitle"
                class="text-lg font-semibold text-(--ink-primary) mb-1.5 tracking-tight"
              >
                Delete Time Slot
              </h3>
              <p class="text-sm text-(--text-secondary) leading-relaxed">
                Are you sure you want to delete this time slot? This action cannot be undone.
              </p>
            </div>
          </div>
          <div class="flex gap-3 justify-end">
            <button
              ref="cancelBtnRef"
              type="button"
              class="px-4 py-2 text-sm font-medium text-(--ink-primary) bg-white border border-(--border-strong) rounded-sm hover:bg-(--paper-cream) focus:ring-2 focus:ring-(--border-strong) transition-all"
              @click="closeDeleteDialog"
            >
              Cancel
            </button>
            <button
              type="button"
              class="px-4 py-2 text-sm font-semibold text-white bg-red-600 rounded-sm hover:bg-red-700 focus:ring-2 focus:ring-red-100 transition-all shadow-sm shadow-red-600/15 tracking-wide"
              @click="confirmDelete"
            >
              Delete
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>
