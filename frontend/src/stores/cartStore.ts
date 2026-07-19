import { defineStore } from 'pinia'
import { ref } from 'vue'
import { bookingApi } from '../services/bookingApi'
import { getValidatedArray, setItem } from '../utils/storage'
import { getErrorMessage } from '../utils/errorHandler'
import { validateDateRange, hasOverlapWithCart } from '../utils/dateValidation'
import { useNotification } from '../composables/useNotification'
import type { CartItem } from '../types/booking'

import { useBookingStore } from './bookingStore'

export const useCartStore = defineStore('cart', () => {
  const { errorMessage, showSuccess, showError, clearMessages } = useNotification()

  // Cart State
  const cartItems = ref<CartItem[]>([])
  const isConfirming = ref(false)

  // Internal state
  let cartIdCounter = 0
  let isAddingToCart = false

  // Type guard
  const isCartItem = (value: unknown): value is CartItem => {
    return (
      value !== null &&
      typeof value === 'object' &&
      'id' in value &&
      typeof value.id === 'number' &&
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

  // Persistence
  const loadCart = () => {
    const loaded = getValidatedArray('bookingCart', isCartItem)
    cartItems.value = loaded
    // Sync counter to avoid ID collisions with persisted items
    if (loaded.length > 0) {
      const maxId = Math.max(...loaded.map((item) => item.id))
      cartIdCounter = Math.max(cartIdCounter, maxId)
    }
  }

  const saveCart = () => {
    setItem('bookingCart', cartItems.value)
  }

  // Cart CRUD
  const addToCart = (item: Omit<CartItem, 'id' | 'status'>) => {
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

    const newItem: CartItem = {
      ...item,
      id: ++cartIdCounter,
      status: 'pending',
    }
    cartItems.value.push(newItem)
    saveCart()

    showSuccess('Added to cart', 2000)
  }

  const removeCartItem = (id: number) => {
    cartItems.value = cartItems.value.filter((item) => item.id !== id)
    saveCart()
  }

  const clearCart = () => {
    cartItems.value = []
    saveCart()
  }

  const addSlotToCart = (
    teacherId: number,
    teacherName: string,
    startTime: string,
    endTime: string,
    subjectId?: number,
    branchId?: number
  ): void => {
    if (isAddingToCart) return

    const booking = useBookingStore()
    const effectiveSubjectId = subjectId ?? booking.selectedSubjectId
    const effectiveBranchId = branchId ?? booking.selectedBranchId
    if (!effectiveBranchId || !effectiveSubjectId) {
      showError(
        new Error('Branch and subject must be selected'),
        'Please select branch and subject before adding to cart'
      )
      return
    }

    isAddingToCart = true

    const newStartDate = new Date(startTime)
    const newEndDate = new Date(endTime)
    const validation = validateDateRange(newStartDate, newEndDate)
    if (!validation.isValid) {
      showError(new Error(validation.error || 'Invalid date'), 'Please select valid time slots')
      isAddingToCart = false
      return
    }

    if (hasOverlapWithCart(startTime, endTime, teacherId, cartItems.value)) {
      showError(
        new Error('Time slot already in cart'),
        'This time slot is already in your cart for this teacher'
      )
      isAddingToCart = false
      return
    }

    const subject = booking.subjects.find((s) => s.id === effectiveSubjectId)
    const branch = booking.branches.find((b) => b.id === effectiveBranchId)
    addToCart({
      teacher_id: teacherId,
      teacher_name: teacherName,
      branch_id: effectiveBranchId,
      branch_name: branch?.name || '',
      subject_id: effectiveSubjectId,
      subject_name: subject?.name || '',
      start_time: startTime,
      end_time: endTime,
      client_name: 'Guest',
    })
    showSuccess('Added to cart!')
    booking.events = []
    isAddingToCart = false
  }

  const confirmBookings = async () => {
    if (cartItems.value.length === 0) return
    if (isConfirming.value) return

    const booking = useBookingStore()

    isConfirming.value = true
    clearMessages()

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

      const successfulItemIds = new Set<number>()
      const conflictItemIds = new Set<number>()
      let otherFailedCount = 0

      results.forEach((result, index) => {
        const item = cartItems.value[index]
        if (result.status === 'fulfilled') {
          successfulItemIds.add(item.id)
        } else {
          const error = result.reason as Error
          const errorMsg = error?.message?.toLowerCase() || ''
          if (
            errorMsg.includes('conflict') ||
            errorMsg.includes('already') ||
            errorMsg.includes('409')
          ) {
            conflictItemIds.add(item.id)
          } else {
            otherFailedCount++
          }
        }
      })

      const idsToRemove = new Set([...successfulItemIds, ...conflictItemIds])
      cartItems.value = cartItems.value.filter((item) => !idsToRemove.has(item.id))
      saveCart()

      const conflictCount = conflictItemIds.size
      const successCount = successfulItemIds.size

      if (conflictCount > 0) {
        errorMessage.value = 'You already booked this subject'
        setTimeout(() => {
          errorMessage.value = ''
        }, 4000)
      } else if (successCount > 0 && otherFailedCount === 0) {
        showSuccess(`Successfully confirmed ${successCount} booking(s)!`)
        booking.fetchConfirmedBookings()
      } else if (otherFailedCount > 0) {
        errorMessage.value = `${otherFailedCount} booking(s) failed. ${successCount} confirmed.`
        setTimeout(() => {
          errorMessage.value = ''
        }, 4000)
      }
    } catch (error) {
      showError(error, getErrorMessage(error, 'Failed to confirm bookings'))
    } finally {
      isConfirming.value = false
    }
  }

  return {
    // State
    cartItems,
    isConfirming,

    // Actions
    loadCart,
    addToCart,
    addSlotToCart,
    removeCartItem,
    clearCart,
    confirmBookings,
  }
})
