import { storeToRefs } from 'pinia'
import { useBookingStore } from '../stores/bookingStore'
import type { CartItem } from '../stores/bookingStore'

/**
 * Booking composable wrapper.
 * All state is managed in useBookingStore (Pinia).
 * This composable provides backwards compatibility with reactive refs.
 */
export function useBooking() {
  const store = useBookingStore()

  // Convert store state to refs for reactivity in templates
  const refs = storeToRefs(store)

  return {
    // UI State (as refs)
    calendarRef: refs.calendarRef,
    activeTab: refs.activeTab,
    isEvaluating: refs.isEvaluating,
    isLoadingTeachers: refs.isLoadingTeachers,
    errorMessage: refs.errorMessage,
    successMessage: refs.successMessage,

    // Selection State (as refs)
    events: refs.events,
    businessHours: refs.businessHours,
    subjects: refs.subjects,
    branches: refs.branches,
    filteredTeachers: refs.filteredTeachers,
    selectedSubjectId: refs.selectedSubjectId,
    selectedBranchId: refs.selectedBranchId,
    selectedTeacherId: refs.selectedTeacherId,
    suggestions: refs.suggestions,
    showDetailedResults: refs.showDetailedResults,
    suggestionEvents: refs.suggestionEvents,
    allEvents: refs.allEvents,

    // Cart (as refs)
    cartItems: refs.cartItems,
    cartIsLoading: refs.cartIsLoading,

    // Actions (direct functions)
    resetBookingState: store.resetBookingState,
    addEvent: store.addEvent,
    addToCartDirectly: store.addToCartDirectly,
    addToCart: store.addToCart,
    removeItem: store.removeItem,
    clearCart: store.clearCart,
    submitBookings: store.submitBookings,
    fetchCartItems: store.fetchCartItems,
    initWatchers: store.initialize,
    showSuccess: store.showSuccess,
    showError: store.showError,
  }
}

export type { CartItem }
