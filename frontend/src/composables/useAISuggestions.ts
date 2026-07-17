import { ref } from 'vue'
import { useBookingCalendarStore } from '../stores/bookingCalendarStore'
import { useBookingSelectionStore } from '../stores/bookingSelectionStore'
import { toMinutes } from '../utils/dateValidation'
import { getErrorMessage, isNetworkError } from '../utils/errorHandler'

export function useAISuggestions() {
  const calendarStore = useBookingCalendarStore()
  const selectionStore = useBookingSelectionStore()

  const currentRequestId = ref<string | null>(null)

  const invalidateRequest = () => {
    currentRequestId.value = null
  }

  const getSuggestions = async (
    slots: Array<{ day_of_week: number; start: string; end: string }>
  ): Promise<boolean> => {
    // Validate selection
    if (
      !selectionStore.selectedSubjectId ||
      !selectionStore.selectedBranchId ||
      slots.length === 0
    ) {
      const missing = []
      if (!selectionStore.selectedSubjectId) missing.push('a subject')
      if (!selectionStore.selectedBranchId) missing.push('a branch')
      if (slots.length === 0) missing.push('at least one time slot')
      calendarStore.showError(
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
        calendarStore.showError(
          new Error('Invalid time'),
          'Invalid time format. Please use HH:MM format (e.g., 09:30).'
        )
        return false
      }
      if (startMin >= endMin) {
        calendarStore.showError(new Error('Invalid range'), 'Start time must be before end time.')
        return false
      }
      durations.add(endMin - startMin)
    }

    if (durations.size !== 1) {
      calendarStore.showError(
        new Error('Mixed duration'),
        'All AI time slots must use the same duration.'
      )
      return false
    }
    const aiDuration = [...durations][0]

    // Generate unique request ID to guard against stale responses
    const requestId = Math.random().toString(36).substring(7)
    currentRequestId.value = requestId

    calendarStore.isEvaluating = true
    calendarStore.errorMessage = ''
    calendarStore.successMessage = ''

    try {
      const { bookingApi } = await import('../services/bookingApi')
      const response = await bookingApi.evaluate({
        subject_id: selectionStore.selectedSubjectId!,
        branch_id: selectionStore.selectedBranchId!,
        preferred_slots: slots,
        duration_minutes: aiDuration,
        preferred_teacher_id: selectionStore.selectedTeacherId ?? undefined,
      })

      // Only apply results if this is still the current request
      if (currentRequestId.value !== requestId) {
        return false
      }

      calendarStore.suggestions = response
      calendarStore.showDetailedResults = true
      calendarStore.activeTab = 'ai'

      const msg = `${response.results.length} teacher${response.results.length > 1 ? 's' : ''} available. Click a time slot on the calendar or a suggestion below to book.`
      calendarStore.showSuccess(msg, 8000)
      return true
    } catch (error) {
      calendarStore.showError(
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
      calendarStore.isEvaluating = false
    }
  }

  return {
    getSuggestions,
    currentRequestId,
    invalidateRequest,
  }
}
