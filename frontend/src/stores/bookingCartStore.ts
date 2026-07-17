import { defineStore } from 'pinia'
import { ref } from 'vue'
import { bookingApi } from '../services/bookingApi'
import { getValidatedArray, setItem } from '../utils/storage'
import { getErrorMessage } from '../utils/errorHandler'
import { validateDateRange, hasOverlapWithCart } from '../utils/dateValidation'
import type { CartItem } from '../types/booking'

import { useBookingCalendarStore } from './bookingCalendarStore'
import { useBookingSelectionStore } from './bookingSelectionStore'

export const useBookingCartStore = defineStore('bookingCart', () => {
  // Cart State
  const cartItems = ref<CartItem[]>([])
  const cartIsLoading = ref(false)

  // Internal state
  let cartIdCounter = 0

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
  const fetchCartItems = () => {
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

    useBookingCalendarStore().showSuccess('Added to cart', 2000)
  }

  const removeItem = (id: number) => {
    cartItems.value = cartItems.value.filter((item) => item.id !== id)
    saveCart()
  }

  const clearCart = () => {
    cartItems.value = []
    saveCart()
  }

  const addToCartDirectly = (
    teacherId: number,
    teacherName: string,
    startTime: string,
    endTime: string,
    subjectId?: number,
    branchId?: number
  ): void => {
    const calendar = useBookingCalendarStore()
    if (calendar.isAddingToCart) return

    const selection = useBookingSelectionStore()
    const effectiveSubjectId = subjectId ?? selection.selectedSubjectId
    const effectiveBranchId = branchId ?? selection.selectedBranchId
    if (!effectiveBranchId || !effectiveSubjectId) {
      calendar.showError(
        new Error('Branch and subject must be selected'),
        'Please select branch and subject before adding to cart'
      )
      return
    }

    calendar.isAddingToCart = true

    const newStartDate = new Date(startTime)
    const newEndDate = new Date(endTime)
    const validation = validateDateRange(newStartDate, newEndDate)
    if (!validation.isValid) {
      calendar.showError(
        new Error(validation.error || 'Invalid date'),
        'Please select valid time slots'
      )
      calendar.isAddingToCart = false
      return
    }

    if (hasOverlapWithCart(startTime, endTime, teacherId, cartItems.value)) {
      calendar.showError(
        new Error('Time slot already in cart'),
        'This time slot is already in your cart for this teacher'
      )
      calendar.isAddingToCart = false
      return
    }

    const subject = selection.subjects.find((s: { id: number }) => s.id === effectiveSubjectId)
    const branch = selection.branches.find((b: { id: number }) => b.id === effectiveBranchId)
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
    calendar.showSuccess('Added to cart!')
    calendar.events = []
    calendar.isAddingToCart = false
  }

  const submitBookings = async () => {
    if (cartItems.value.length === 0) return
    if (cartIsLoading.value) return

    const calendar = useBookingCalendarStore()

    cartIsLoading.value = true
    calendar.errorMessage = ''
    calendar.successMessage = ''

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
        calendar.errorMessage = 'You already booked this subject'
        setTimeout(() => {
          calendar.errorMessage = ''
        }, 4000)
      } else if (successCount > 0 && otherFailedCount === 0) {
        calendar.successMessage = `Successfully confirmed ${successCount} booking(s)!`
        calendar.fetchConfirmedBookings()
      } else if (otherFailedCount > 0) {
        calendar.errorMessage = `${otherFailedCount} booking(s) failed. ${successCount} confirmed.`
        setTimeout(() => {
          calendar.errorMessage = ''
        }, 4000)
      }
    } catch (error) {
      calendar.errorMessage = getErrorMessage(error, 'Failed to confirm bookings')
    } finally {
      cartIsLoading.value = false
    }
  }

  return {
    // State
    cartItems,
    cartIsLoading,

    // Actions
    fetchCartItems,
    addToCart,
    addToCartDirectly,
    removeItem,
    clearCart,
    submitBookings,
    saveCart,
  }
})
