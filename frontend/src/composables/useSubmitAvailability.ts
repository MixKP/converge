import { ref, computed, watch } from 'vue'
import { useForm } from 'vee-validate'
import { useTeacherStore } from '../stores/teacherStore'
import { availabilityApi } from '../services/availabilityApi'
import { bookingApi } from '../services/bookingApi'
import type { EventInput } from '@fullcalendar/vue3'
import { generateAvailabilityPayload, startOfWeek } from '../utils/calendarHelpers'
import { availabilityFormSchema } from '../schemas/forms'
import { availabilityPayloadSchema, type WeeklySlotInput } from '../schemas/calendar'
import { toTypedSchema } from '../schemas/veeValidate'
import { useNotification } from './useNotification'

export function useSubmitAvailability() {
  const teacherStore = useTeacherStore()
  const isLoading = ref(false)
  const events = ref<EventInput[]>([])
  const showConfirm = ref(false)
  const isCountingBookings = ref(false)
  const affectedBookingCount = ref<number | null>(null)
  const { successMessage, errorMessage, showSuccess, showError } = useNotification()

  const {
    submitCount,
    setFieldValue,
    resetForm,
    handleSubmit: veeHandleSubmit,
  } = useForm({
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

  watch(
    events,
    (list) => {
      const teacherId = selectedTeacherId.value ?? 0
      const weekly = teacherId > 0 ? generateAvailabilityPayload(list, teacherId).weekly : []
      setFieldValue('weekly', weekly)
    },
    { deep: true }
  )

  const showErrors = computed(() => submitCount.value > 0)

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

  watch(selectedTeacherId, (id) => {
    setFieldValue('teacher_id', id)
    events.value = []
    errorMessage.value = ''
  })

  const loadAffectedBookingCount = async (teacherId: number) => {
    isCountingBookings.value = true
    affectedBookingCount.value = null
    try {
      // tradeoff: week start uses the browser's timezone while the server uses Asia/Bangkok -> count server-side via a preview endpoint
      const weekStart = startOfWeek(new Date())
      const bookings = await bookingApi.list()
      affectedBookingCount.value = bookings.filter(
        (b) => b.teacher_id === teacherId && new Date(b.start_time).getTime() >= weekStart.getTime()
      ).length
    } catch (error) {
      console.error('Failed to count affected bookings:', error)
    } finally {
      isCountingBookings.value = false
    }
  }

  const handleSubmit = veeHandleSubmit((values) => {
    showConfirm.value = true
    loadAffectedBookingCount(values.teacher_id)
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
      const { deleted_bookings: deleted } = await availabilityApi.submitAvailability(payload)
      events.value = []
      teacherStore.setSelectedTeacherById(null)
      resetForm({ values: { teacher_id: null, weekly: [] } })
      showConfirm.value = false
      showSuccess(
        deleted > 0
          ? `Availability submitted. ${deleted} booked class${deleted === 1 ? '' : 'es'} removed.`
          : 'Availability submitted successfully!',
        3000
      )
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
    isCountingBookings,
    affectedBookingCount,
    selectedTeacherId,
    selectedTeacher,
    showErrors,
    formattedSlots,
    handleSubmit,
    confirmSubmit,
    cancelConfirm,
    clearEvents,
  }
}
