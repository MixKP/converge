import { ref } from 'vue'
import { useBookingStore } from '../stores/bookingStore'
import { useNotification } from './useNotification'
import { toMinutes } from '../utils/dateValidation'
import { bookingRequestSchema } from '../schemas/booking'
import { getErrorMessage, isNetworkError } from '../utils/errorHandler'

export function useAISuggestions() {
  const bookingStore = useBookingStore()
  const { showSuccess, showError, clearMessages } = useNotification()

  const currentRequestId = ref<string | null>(null)

  const invalidateRequest = () => {
    currentRequestId.value = null
  }

  const getSuggestions = async (
    slots: Array<{ day_of_week: number; start: string; end: string }>
  ): Promise<boolean> => {
    const firstSlot = slots[0]
    const aiDuration = firstSlot ? toMinutes(firstSlot.end) - toMinutes(firstSlot.start) : undefined

    const request = bookingRequestSchema.safeParse({
      subject_id: bookingStore.selectedSubjectId,
      branch_id: bookingStore.selectedBranchId,
      preferred_slots: slots,
      duration_minutes: aiDuration,
      preferred_teacher_id: bookingStore.selectedTeacherId ?? undefined,
    })
    if (!request.success) {
      showError(
        new Error('Invalid request'),
        'Please select a subject, a branch, and at least one time slot.'
      )
      return false
    }

    const requestId = globalThis.crypto.randomUUID()
    currentRequestId.value = requestId

    bookingStore.isEvaluating = true
    clearMessages()

    try {
      const { bookingApi } = await import('../services/bookingApi')
      const response = await bookingApi.evaluate(request.data)

      if (currentRequestId.value !== requestId) {
        return false
      }

      bookingStore.suggestions = response
      bookingStore.showDetailedResults = true

      const msg = `${response.results.length} teacher${response.results.length > 1 ? 's' : ''} available. Click a time slot on the calendar or a suggestion below to book.`
      showSuccess(msg, 8000)
      return true
    } catch (error) {
      showError(
        error,
        isNetworkError(error)
          ? 'Network error. Check your connection and try again.'
          : getErrorMessage(
              error,
              'No teachers available for those time slots. Try adding more time options or different days.'
            )
      )
      return false
    } finally {
      bookingStore.isEvaluating = false
    }
  }

  return {
    getSuggestions,
    currentRequestId,
    invalidateRequest,
  }
}
