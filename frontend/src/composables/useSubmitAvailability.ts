import { ref, computed, watch } from 'vue'
import { useTeacherStore } from '../stores/teacherStore'
import { availabilityApi } from '../services/availabilityApi'
import type { EventInput } from '@fullcalendar/core'
import { generateAvailabilityPayload } from '../utils/calendarHelpers'

export function useSubmitAvailability() {
  const teacherStore = useTeacherStore()
  const isLoading = ref(false)
  const errorMessage = ref('')
  const successMessage = ref('')
  const events = ref<EventInput[]>([])
  const showConfirm = ref(false)

  const selectedTeacherId = computed({
    get: () => teacherStore.selectedTeacherId,
    set: (val) => teacherStore.setSelectedTeacherById(val === null ? null : Number(val)),
  })

  const selectedTeacher = computed(() =>
    teacherStore.teachers.find((t) => t.id === selectedTeacherId.value)
  )

  const canSubmit = computed(
    () => !!selectedTeacherId.value && !isLoading.value && events.value.length > 0
  )

  const formattedSlots = computed(() => {
    return events.value
      .map((event) => {
        const start = new Date(event.start as string)
        const end = new Date(event.end as string)
        const dayName = start.toLocaleDateString('en-US', { weekday: 'long' })
        const startTime = start.toLocaleTimeString('en-US', {
          hour: '2-digit',
          minute: '2-digit',
          hour12: false,
        })
        const endTime = end.toLocaleTimeString('en-US', {
          hour: '2-digit',
          minute: '2-digit',
          hour12: false,
        })
        return { day: dayName, start: startTime, end: endTime }
      })
      .sort((a, b) => {
        const days = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']
        return days.indexOf(a.day) - days.indexOf(b.day) || a.start.localeCompare(b.start)
      })
  })

  watch(selectedTeacherId, () => {
    events.value = []
    errorMessage.value = ''
  })

  const handleSubmit = () => {
    if (!canSubmit.value) return
    showConfirm.value = true
  }

  const confirmSubmit = async () => {
    if (isLoading.value || !canSubmit.value) return

    isLoading.value = true
    errorMessage.value = ''
    successMessage.value = ''

    try {
      const payload = generateAvailabilityPayload(events.value, selectedTeacherId.value!)
      await availabilityApi.submitAvailability(payload)
      events.value = []
      teacherStore.setSelectedTeacherById(null)
      showConfirm.value = false
      successMessage.value = 'Availability submitted successfully!'
      setTimeout(() => (successMessage.value = ''), 3000)
    } catch (error) {
      errorMessage.value =
        error instanceof Error ? error.message : 'Failed to submit availability. Please try again.'
    } finally {
      isLoading.value = false
    }
  }

  const cancelConfirm = () => {
    showConfirm.value = false
  }

  const clearEvents = () => {
    events.value = []
  }

  return {
    teacherStore,
    isLoading,
    errorMessage,
    successMessage,
    events,
    showConfirm,
    selectedTeacherId,
    selectedTeacher,
    canSubmit,
    formattedSlots,
    handleSubmit,
    confirmSubmit,
    cancelConfirm,
    clearEvents,
  }
}
