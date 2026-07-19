import { storeToRefs } from 'pinia'
import { useBookingStore } from '../stores/bookingStore'

export function useBooking() {
  const bookingStore = useBookingStore()
  const refs = storeToRefs(bookingStore)

  return {
    // Selection State
    subjects: refs.subjects,
    branches: refs.branches,
    filteredTeachers: refs.filteredTeachers,
    selectedSubjectId: refs.selectedSubjectId,
    selectedBranchId: refs.selectedBranchId,
    selectedTeacherId: refs.selectedTeacherId,
    isLoadingTeachers: refs.isLoadingTeachers,

    // Calendar & Availability State
    calendarRef: refs.calendarRef,
    isEvaluating: refs.isEvaluating,
    events: refs.events,
    businessHours: refs.businessHours,
    suggestions: refs.suggestions,
    showDetailedResults: refs.showDetailedResults,
    suggestionEvents: refs.suggestionEvents,
    filteredSuggestionEvents: refs.filteredSuggestionEvents,
    allEvents: refs.allEvents,

    // Actions
    fetchSubjects: bookingStore.fetchSubjects,
    fetchBranches: bookingStore.fetchBranches,
    resetBookingState: bookingStore.resetBookingState,
    fetchConfirmedBookings: bookingStore.fetchConfirmedBookings,
    initialize: bookingStore.initialize,
  }
}

export type { CartItem } from '../types/booking'
