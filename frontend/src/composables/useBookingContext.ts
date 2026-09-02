import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useBookingStore } from '../stores/bookingStore'

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
