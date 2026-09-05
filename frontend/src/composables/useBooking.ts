import { storeToRefs } from 'pinia'
import { useBookingStore } from '../stores/bookingStore'

export function useBooking() {
  const bookingStore = useBookingStore()
  const refs = storeToRefs(bookingStore)

  return {
    subjects: refs.subjects,
    branches: refs.branches,
    activeBranches: refs.activeBranches,
    students: refs.students,
    filteredTeachers: refs.filteredTeachers,
    genderFilteredTeachers: refs.genderFilteredTeachers,
    selectedStudentId: refs.selectedStudentId,
    selectedSubjectId: refs.selectedSubjectId,
    selectedBranchId: refs.selectedBranchId,
    selectedTeacherId: refs.selectedTeacherId,
    isLoadingTeachers: refs.isLoadingTeachers,
    requiredGender: refs.requiredGender,

    calendarRef: refs.calendarRef,
    isEvaluating: refs.isEvaluating,
    events: refs.events,
    businessHours: refs.businessHours,
    isAvailabilityLoaded: refs.isAvailabilityLoaded,
    suggestions: refs.suggestions,
    showDetailedResults: refs.showDetailedResults,
    allEvents: refs.allEvents,

    fetchSubjects: bookingStore.fetchSubjects,
    fetchBranches: bookingStore.fetchBranches,
    fetchStudents: bookingStore.fetchStudents,
    resetBookingState: bookingStore.resetBookingState,
    fetchConfirmedBookings: bookingStore.fetchConfirmedBookings,
    initialize: bookingStore.initialize,
  }
}
