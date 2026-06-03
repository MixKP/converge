import { ref } from 'vue'
import { bookingApi } from '../services/bookingApi'

export interface CartItem {
  id: number
  teacher_id: number
  teacher_name: string
  branch_id: number
  branch_name: string
  subject_id: number
  subject_name: string
  start_time: string
  end_time: string
  client_name: string
  status: 'pending' | 'confirmed'
}

const cartItems = ref<CartItem[]>([])
const isLoading = ref(false)
const errorMessage = ref<string>('')
const successMessage = ref<string>('')

export function useBookingCart() {
  const fetchCartItems = () => {
    const stored = localStorage.getItem('bookingCart')
    if (stored) {
      try {
        cartItems.value = JSON.parse(stored)
      } catch {
        cartItems.value = []
      }
    }
  }

  const saveCart = () => {
    localStorage.setItem('bookingCart', JSON.stringify(cartItems.value))
  }

  const addToCart = (item: Omit<CartItem, 'id' | 'status'>) => {
    const newItem: CartItem = {
      ...item,
      id: Date.now() + Math.random(),
      status: 'pending',
    }
    cartItems.value.push(newItem)
    saveCart()
    successMessage.value = 'Added to cart'
    setTimeout(() => (successMessage.value = ''), 2000)
  }

  const removeItem = (id: number) => {
    cartItems.value = cartItems.value.filter((item) => item.id !== id)
    saveCart()
  }

  const clearCart = () => {
    cartItems.value = []
    saveCart()
  }

  const submitBookings = async () => {
    if (cartItems.value.length === 0) return

    isLoading.value = true
    errorMessage.value = ''
    successMessage.value = ''

    try {
      const promises = cartItems.value.map((item) =>
        bookingApi.confirm({
          teacher_id: item.teacher_id,
          branch_id: item.branch_id,
          subject_id: item.subject_id,
          start_time: item.start_time,
          end_time: item.end_time,
          client_name: item.client_name,
        })
      )

      const results = await Promise.allSettled(promises)

      const successfulItems: CartItem[] = []
      const failedCount = results.filter((r) => r.status === 'rejected').length

      results.forEach((result, index) => {
        if (result.status === 'fulfilled') {
          successfulItems.push(cartItems.value[index])
        }
      })

      if (failedCount > 0) {
        cartItems.value = cartItems.value.filter(
          (item) => !successfulItems.some((success) => success.id === item.id)
        )
        saveCart()
        errorMessage.value = `${failedCount} booking(s) failed. ${successfulItems.length} succeeded and were removed from cart.`
      } else {
        successMessage.value = `Successfully confirmed ${cartItems.value.length} booking(s)!`
        clearCart()
      }
    } catch (error) {
      errorMessage.value = error instanceof Error ? error.message : 'Failed to confirm bookings'
    } finally {
      isLoading.value = false
    }
  }

  return {
    cartItems,
    isLoading,
    errorMessage,
    successMessage,
    fetchCartItems,
    addToCart,
    removeItem,
    clearCart,
    submitBookings,
  }
}
