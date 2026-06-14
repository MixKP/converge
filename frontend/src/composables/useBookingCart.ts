import { ref } from 'vue'
import { bookingApi } from '../services/bookingApi'
import { getValidatedArray, setItem } from '../utils/storage'
import { getErrorMessage } from '../utils/errorHandler'
import { useMessages } from './useMessages'

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

// Module-level cart ID counter for unique ID generation
let cartIdCounter = 0

export function useBookingCart() {
  // State is now per-instance, not shared across components
  const cartItems = ref<CartItem[]>([])
  const isLoading = ref(false)
  const { successMessage, errorMessage, showSuccess } = useMessages()
  // Type guard for CartItem validation
  const isCartItem = (value: unknown): value is CartItem => {
    return (
      value !== null &&
      typeof value === 'object' &&
      'teacher_id' in value &&
      typeof value.teacher_id === 'number' &&
      'teacher_name' in value &&
      typeof value.teacher_name === 'string' &&
      'branch_id' in value &&
      typeof value.branch_id === 'number' &&
      'branch_name' in value &&
      typeof value.branch_name === 'string' &&
      'subject_id' in value &&
      typeof value.subject_id === 'number' &&
      'subject_name' in value &&
      typeof value.subject_name === 'string' &&
      'start_time' in value &&
      typeof value.start_time === 'string' &&
      'end_time' in value &&
      typeof value.end_time === 'string' &&
      'client_name' in value &&
      typeof value.client_name === 'string' &&
      'status' in value &&
      (value.status === 'pending' || value.status === 'confirmed')
    )
  }

  const fetchCartItems = () => {
    cartItems.value = getValidatedArray('bookingCart', isCartItem)
  }

  const saveCart = () => {
    setItem('bookingCart', cartItems.value)
  }

  const addToCart = (item: Omit<CartItem, 'id' | 'status'>) => {
    // Validate input item
    if (!item || typeof item !== 'object') {
      console.error('Invalid cart item: not an object')
      return
    }

    const requiredFields: (keyof Omit<CartItem, 'id' | 'status'>)[] = [
      'teacher_id',
      'teacher_name',
      'branch_id',
      'branch_name',
      'subject_id',
      'subject_name',
      'start_time',
      'end_time',
      'client_name',
    ]
    for (const field of requiredFields) {
      if (!(field in item) || item[field] === null || item[field] === undefined) {
        console.error(`Invalid cart item: missing or invalid field "${field}"`)
        return
      }
    }

    // Use monotonic counter for unique IDs instead of timestamp + random
    const newItem: CartItem = {
      ...item,
      id: ++cartIdCounter,
      status: 'pending',
    }
    cartItems.value.push(newItem)
    saveCart()
    showSuccess('Added to cart', 2000)
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
    // Prevent multiple concurrent submissions
    if (isLoading.value) return

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
        // Remove only successful items from cart (they've been confirmed)
        // Keep failed items so user can retry or remove them manually
        const successfulIds = new Set(successfulItems.map((item) => item.id))
        cartItems.value = cartItems.value.filter((item) => !successfulIds.has(item.id))
        saveCart()
        errorMessage.value = `${failedCount} booking(s) failed and remain in cart. ${successfulItems.length} succeeded and were confirmed.`
      } else {
        successMessage.value = `Successfully confirmed ${cartItems.value.length} booking(s)!`
        clearCart()
      }
    } catch (error) {
      errorMessage.value = getErrorMessage(error, 'Failed to confirm bookings')
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
