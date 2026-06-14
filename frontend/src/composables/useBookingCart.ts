import { storeToRefs } from 'pinia'
import { useBookingStore } from '../stores/bookingStore'

export type { CartItem } from '../stores/bookingStore'

/**
 * Cart composable wrapper.
 * Delegates to useBookingStore for backwards compatibility.
 * All cart state is managed in useBookingStore (Pinia).
 */
export function useBookingCart() {
  const store = useBookingStore()
  const refs = storeToRefs(store)

  return {
    cartItems: refs.cartItems,
    isLoading: refs.cartIsLoading,
    errorMessage: refs.errorMessage,
    successMessage: refs.successMessage,
    fetchCartItems: store.fetchCartItems,
    addToCart: store.addToCart,
    removeItem: store.removeItem,
    clearCart: store.clearCart,
    submitBookings: store.submitBookings,
  }
}
