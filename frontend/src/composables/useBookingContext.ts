import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useBookingStore } from '../stores/bookingStore'

/* Student, subject and branch are the prerequisites of *both* booking paths, so
 * they are the only real gate. The blocker is derived from which one is still
 * empty rather than hardcoded — the old page always said "select a subject"
 * even when the subject was set and the branch was what was actually missing. */
export function useBookingContext() {
  const store = useBookingStore()
  const { selectedStudentId, selectedSubjectId, selectedBranchId } = storeToRefs(store)

  const contextBlocker = computed<string | null>(() => {
    if (selectedStudentId.value === null) return 'Choose the student this booking is for.'
    if (selectedSubjectId.value === null) return 'Choose a subject.'
    if (selectedBranchId.value === null) return 'Choose a branch.'
    return null
  })

  const contextComplete = computed(() => contextBlocker.value === null)

  return { contextBlocker, contextComplete }
}
