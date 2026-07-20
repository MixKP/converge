import { storeToRefs } from 'pinia'
import { useCartStore } from '../stores/cartStore'

export function useCart() {
  const cartStore = useCartStore()
  const { cartItems, isConfirming } = storeToRefs(cartStore)

  return {
    cartItems,
    isConfirming,
    loadCart: cartStore.loadCart,
    addToCart: cartStore.addToCart,
    addSlotToCart: cartStore.addSlotToCart,
    removeCartItem: cartStore.removeCartItem,
    clearCart: cartStore.clearCart,
    confirmBookings: cartStore.confirmBookings,
  }
}
