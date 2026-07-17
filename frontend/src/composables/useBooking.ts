import { storeToRefs } from 'pinia'
import { useBookingSelectionStore } from '../stores/bookingSelectionStore'
import { useBookingCalendarStore } from '../stores/bookingCalendarStore'
import { useBookingCartStore } from '../stores/bookingCartStore'

export function useBooking() {
  const selectionStore = useBookingSelectionStore()
  const calendarStore = useBookingCalendarStore()
  const cartStore = useBookingCartStore()

  const selectionRefs = storeToRefs(selectionStore)
  const calendarRefs = storeToRefs(calendarStore)
  const cartRefs = storeToRefs(cartStore)

  return {
    // UI State (from calendar store)
    calendarRef: calendarRefs.calendarRef,
    activeTab: calendarRefs.activeTab,
    isEvaluating: calendarRefs.isEvaluating,
    isLoadingTeachers: selectionRefs.isLoadingTeachers,
    errorMessage: calendarRefs.errorMessage,
    successMessage: calendarRefs.successMessage,

    // Selection State (from selection store)
    events: calendarRefs.events,
    businessHours: calendarRefs.businessHours,
    subjects: selectionRefs.subjects,
    branches: selectionRefs.branches,
    filteredTeachers: selectionRefs.filteredTeachers,
    selectedSubjectId: selectionRefs.selectedSubjectId,
    selectedBranchId: selectionRefs.selectedBranchId,
    selectedTeacherId: selectionRefs.selectedTeacherId,
    suggestions: calendarRefs.suggestions,
    showDetailedResults: calendarRefs.showDetailedResults,
    suggestionEvents: calendarRefs.suggestionEvents,
    allEvents: calendarRefs.allEvents,

    // Cart (from cart store)
    cartItems: cartRefs.cartItems,
    cartIsLoading: cartRefs.cartIsLoading,

    // Actions (from calendar store)
    resetBookingState: calendarStore.resetBookingState,
    addEvent: calendarStore.addEvent,
    showSuccess: calendarStore.showSuccess,
    showError: calendarStore.showError,
    initWatchers: calendarStore.initialize,

    // Actions (from selection store)
    fetchSubjects: selectionStore.fetchSubjects,
    fetchBranches: selectionStore.fetchBranches,

    // Actions (from cart store)
    addToCartDirectly: cartStore.addToCartDirectly,
    addToCart: cartStore.addToCart,
    removeItem: cartStore.removeItem,
    clearCart: cartStore.clearCart,
    submitBookings: cartStore.submitBookings,
    fetchCartItems: cartStore.fetchCartItems,
  }
}

export type { CartItem } from '../types/booking'