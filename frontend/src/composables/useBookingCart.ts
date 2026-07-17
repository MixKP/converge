import { storeToRefs } from 'pinia'
import { useBookingCartStore } from '../stores/bookingCartStore'
import { useBookingCalendarStore } from '../stores/bookingCalendarStore'

export type { CartItem } from '../types/booking'

export function useBookingCart() {
  const cartStore = useBookingCartStore()
  const calendarStore = useBookingCalendarStore()

  const cartRefs = storeToRefs(cartStore)
  const calendarRefs = storeToRefs(calendarStore)

  return {
    cartItems: cartRefs.cartItems,
    isLoading: cartRefs.cartIsLoading,
    errorMessage: calendarRefs.errorMessage,
    successMessage: calendarRefs.successMessage,
    fetchCartItems: cartStore.fetchCartItems,
    addToCart: cartStore.addToCart,
    removeItem: cartStore.removeItem,
    clearCart: cartStore.clearCart,
    submitBookings: cartStore.submitBookings,
  }
}