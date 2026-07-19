import { ref } from 'vue'
import { useBookingStore } from '../stores/bookingStore'
import { useNotification } from './useNotification'
import { toMinutes } from '../utils/dateValidation'
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
    // Validate selection
    if (!bookingStore.selectedSubjectId || !bookingStore.selectedBranchId || slots.length === 0) {
      const missing = []
      if (!bookingStore.selectedSubjectId) missing.push('a subject')
      if (!bookingStore.selectedBranchId) missing.push('a branch')
      if (slots.length === 0) missing.push('at least one time slot')
      showError(
        new Error('Missing selection'),
        `To find teachers, please select ${missing.join(' and ')}`
      )
      return false
    }

    // Validate all slots have valid time format and same duration
    const durations = new Set<number>()
    for (const slot of slots) {
      const startMin = toMinutes(slot.start)
      const endMin = toMinutes(slot.end)

      if (isNaN(startMin) || isNaN(endMin)) {
        showError(
          new Error('Invalid time'),
          'Invalid time format. Please use HH:MM format (e.g., 09:30).'
        )
        return false
      }
      if (startMin >= endMin) {
        showError(new Error('Invalid range'), 'Start time must be before end time.')
        return false
      }
      durations.add(endMin - startMin)
    }

    if (durations.size !== 1) {
      showError(new Error('Mixed duration'), 'All AI time slots must use the same duration.')
      return false
    }
    const aiDuration = [...durations][0]

    // Generate unique request ID to guard against stale responses
    const requestId = Math.random().toString(36).substring(7)
    currentRequestId.value = requestId

    bookingStore.isEvaluating = true
    clearMessages()

    try {
      const { bookingApi } = await import('../services/bookingApi')
      const response = await bookingApi.evaluate({
        subject_id: bookingStore.selectedSubjectId!,
        branch_id: bookingStore.selectedBranchId!,
        preferred_slots: slots,
        duration_minutes: aiDuration,
        preferred_teacher_id: bookingStore.selectedTeacherId ?? undefined,
      })

      // Only apply results if this is still the current request
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
