import { ref, computed, watch } from 'vue'
import { useForm } from 'vee-validate'
import { useTeacherStore } from '../stores/teacherStore'
import { availabilityApi } from '../services/availabilityApi'
import type { EventInput } from '@fullcalendar/core'
import { generateAvailabilityPayload } from '../utils/calendarHelpers'
import { availabilityFormSchema } from '../schemas/forms'
import { availabilityPayloadSchema, type WeeklySlotInput } from '../schemas/calendar'
import { toTypedSchema } from '../schemas/veeValidate'
import { useMessages } from './useMessages'

export function useSubmitAvailability() {
  const teacherStore = useTeacherStore()
  const isLoading = ref(false)
  const events = ref<EventInput[]>([])
  const showConfirm = ref(false)
  const { successMessage, errorMessage, showSuccess, showError } = useMessages()

  const { meta, errors, setFieldValue, handleSubmit: veeHandleSubmit } = useForm({
    validationSchema: toTypedSchema(availabilityFormSchema),
    initialValues: {
      teacher_id: teacherStore.selectedTeacherId,
      weekly: [] as WeeklySlotInput[],
    },
  })

  const selectedTeacherId = computed({
    get: () => teacherStore.selectedTeacherId,
    set: (val) => teacherStore.setSelectedTeacherById(val === null ? null : Number(val)),
  })

  const selectedTeacher = computed(() =>
    teacherStore.teachers.find((t) => t.id === selectedTeacherId.value)
  )

  // `weekly` is a hidden form field derived from the calendar events, so schema
  // rules (min 1 slot, no same-day overlap) drive form validity.
  watch(
    events,
    (list) => {
      const teacherId = selectedTeacherId.value ?? 0
      const weekly = teacherId > 0 ? generateAvailabilityPayload(list, teacherId).weekly : []
      setFieldValue('weekly', weekly)
    },
    { deep: true }
  )

  const canSubmit = computed(() => meta.value.valid && !isLoading.value)

  const weeklyError = computed(() => errors.value.weekly)

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

  const handleSubmit = veeHandleSubmit(() => {
    showConfirm.value = true
  })

  const confirmSubmit = async () => {
    if (isLoading.value) return

    isLoading.value = true
    errorMessage.value = ''
    successMessage.value = ''

    try {
      const payload = availabilityPayloadSchema.parse(
        generateAvailabilityPayload(events.value, selectedTeacherId.value ?? 0)
      )
      await availabilityApi.submitAvailability(payload)
      events.value = []
      teacherStore.setSelectedTeacherById(null)
      showConfirm.value = false
      showSuccess('Availability submitted successfully!')
    } catch (error) {
      showError(error, 'Failed to submit availability. Please try again.')
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
    weeklyError,
    formattedSlots,
    handleSubmit,
    confirmSubmit,
    cancelConfirm,
    clearEvents,
  }
}
