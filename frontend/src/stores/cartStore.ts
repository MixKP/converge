import { defineStore } from 'pinia'
import { ref } from 'vue'
import { bookingApi } from '../services/bookingApi'
import { getValidatedArray, setItem } from '../utils/storage'
import { getErrorMessage } from '../utils/errorHandler'
import { hasOverlapWithCart } from '../utils/dateValidation'
import { toConfirmRequest } from '../utils/bookingPayload'
import { useNotification } from '../composables/useNotification'
import { cartItemSchema, cartItemInputSchema } from '../schemas/booking'
import { dateRangeSchema } from '../schemas/common'
import type { CartItem } from '../types/booking'

import { useBookingStore } from './bookingStore'

export const useCartStore = defineStore('cart', () => {
  const { showSuccess, showError, clearMessages } = useNotification()

  const cartItems = ref<CartItem[]>([])
  const isConfirming = ref(false)

  let cartIdCounter = 0
  let isAddingToCart = false

  const isCartItem = (value: unknown): value is CartItem => cartItemSchema.safeParse(value).success

  const loadCart = () => {
    const loaded = getValidatedArray('bookingCart', isCartItem)
    cartItems.value = loaded
    if (loaded.length > 0) {
      const maxId = Math.max(...loaded.map((item) => item.id))
      cartIdCounter = Math.max(cartIdCounter, maxId)
    }
  }

  const saveCart = () => {
    setItem('bookingCart', cartItems.value)
  }

  const addToCart = (item: Omit<CartItem, 'id' | 'status'>) => {
    const parsed = cartItemInputSchema.safeParse(item)
    if (!parsed.success) {
      console.error('Invalid cart item:', parsed.error.issues)
      return
    }

    const newItem: CartItem = {
      ...parsed.data,
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
    startTime: string | Date,
    endTime: string | Date,
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

    // Confirming rejects a booking with no student, so refuse to build a cart
    // item that could never be confirmed.
    const student = booking.selectedStudent
    if (!student) {
      showError(
        new Error('Student must be selected'),
        'Please select the student this booking is for'
      )
      return
    }

    isAddingToCart = true

    const toIso = (value: string | Date): string | null => {
      const date = new Date(value)
      return Number.isNaN(date.getTime()) ? null : date.toISOString()
    }
    const startIso = toIso(startTime)
    const endIso = toIso(endTime)

    const validation = dateRangeSchema.safeParse({ start_time: startIso, end_time: endIso })
    if (!validation.success) {
      const error = validation.error.issues[0]?.message || 'Invalid date'
      showError(new Error(error), 'Please select valid time slots')
      isAddingToCart = false
      return
    }
    const { start_time: normalizedStart, end_time: normalizedEnd } = validation.data

    if (hasOverlapWithCart(normalizedStart, normalizedEnd, teacherId, cartItems.value)) {
      showError(
        new Error('Time slot already in cart'),
        'This time slot is already in your cart for this teacher'
      )
      isAddingToCart = false
      return
    }

    const subject = booking.subjects.find((s) => s.id === effectiveSubjectId)
    const branch = booking.branches.find((b) => b.id === effectiveBranchId)
    const before = cartItems.value.length
    addToCart({
      teacher_id: teacherId,
      teacher_name: teacherName,
      branch_id: effectiveBranchId,
      branch_name: branch?.name || '',
      subject_id: effectiveSubjectId,
      subject_name: subject?.name || '',
      start_time: normalizedStart,
      end_time: normalizedEnd,
      student_id: student.id,
      student_name: student.name,
    })
    if (cartItems.value.length > before) {
      booking.events = []
    }
    isAddingToCart = false
  }

  const confirmBookings = async () => {
    if (cartItems.value.length === 0) return
    if (isConfirming.value) return

    const booking = useBookingStore()

    isConfirming.value = true
    clearMessages()

    const items = cartItems.value.slice()

    try {
      const promises = items.map((item) => {
        const request = toConfirmRequest(item)
        return request
          ? bookingApi.confirm(request)
          : Promise.reject(new Error('This booking is missing a student and cannot be confirmed'))
      })

      const results = await Promise.allSettled(promises)

      const successfulItemIds = new Set<number>()
      const conflictItemIds = new Set<number>()
      let otherFailedCount = 0

      results.forEach((result, index) => {
        const item = items[index]
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
        showError(null, 'You already booked this subject', 4000)
      } else if (successCount > 0 && otherFailedCount === 0) {
        showSuccess(`Successfully confirmed ${successCount} booking(s)!`)
        booking.fetchConfirmedBookings()
      } else if (otherFailedCount > 0) {
        showError(null, `${otherFailedCount} booking(s) failed. ${successCount} confirmed.`, 4000)
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
