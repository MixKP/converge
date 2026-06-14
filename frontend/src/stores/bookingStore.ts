import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'
import { useTeacherStore } from './teacherStore'
import { availabilityApi } from '../services/availabilityApi'
import { subjectApi } from '../services/subjectApi'
import { teacherApi } from '../services/teacherApi'
import { branchApi } from '../services/branchApi'
import { bookingApi } from '../services/bookingApi'
import type { EventInput, BusinessHoursInput } from '@fullcalendar/core'
import type {
  WeeklySlot,
  Subject,
  BookingResponse,
  BookingAlternative,
  Teacher,
  Branch,
  Booking,
} from '../types'
import { transformBackendAvailability } from '../utils/availabilityTransform'
import { debounce } from '../utils/common'
import { getValidatedArray, setItem } from '../utils/storage'
import { getErrorMessage } from '../utils/errorHandler'
import {
  validateDateRange,
  hasOverlapWithCart,
  formatTimeString,
  getNextDateForDayOfWeek,
  isNextWeek,
  formatSuggestionDate,
} from '../utils/dateValidation'

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

export const useBookingStore = defineStore('booking', () => {
  const teacherStore = useTeacherStore()

  // UI State
  const calendarRef = ref()
  const activeTab = ref<'manual' | 'ai'>('manual')
  const isLoadingTeachers = ref(false)
  const isEvaluating = ref(false)
  const isAddingToCart = ref(false)
  const successMessage = ref('')
  const errorMessage = ref('')

  // Selection State
  const subjects = ref<Subject[]>([])
  const branches = ref<Branch[]>([])
  const filteredTeachers = ref<Teacher[]>([])
  const selectedSubjectId = ref<number | null>(null)
  const selectedBranchId = ref<number | null>(null)
  const preferredTeacherId = ref<number | null>(null)

  // Event & Availability State
  const events = ref<EventInput[]>([])
  const businessHours = ref<BusinessHoursInput>([])
  const availabilityCache = ref<Map<number, WeeklySlot[]>>(new Map())
  const suggestions = ref<BookingResponse | null>(null)
  const showDetailedResults = ref(false)

  // Cart State
  const cartItems = ref<CartItem[]>([])
  const cartIsLoading = ref(false)

  // Confirmed Bookings State
  const confirmedBookings = ref<Booking[]>([])
  const isLoadingBookings = ref(false)

  // Internal state
  let availabilityPromise: Promise<void> | null = null
  let cartIdCounter = 0
  let watchersInitialized = false
  let currentSubjectChangePromise: Promise<void> | null = null

  // Getters
  const selectedTeacherId = computed({
    get: () => teacherStore.selectedTeacherId,
    set: (val) => teacherStore.setSelectedTeacherById(val),
  })

  // Helper: Convert day_of_week + time to proper date
  const getDateForDayOfWeek = (dayOfWeek: number, timeStr: string): Date => {
    return getNextDateForDayOfWeek(dayOfWeek, timeStr)
  }

  const createExactMatchEvent = (match: BookingAlternative, slot: WeeklySlot): EventInput => {
    const backendDate = new Date(match.start_time)
    const timeStr = formatTimeString(backendDate)
    const startDate = getDateForDayOfWeek(backendDate.getDay(), timeStr)

    const backendEndDate = new Date(match.end_time)
    const endTimeStr = formatTimeString(backendEndDate)
    const endDate = getDateForDayOfWeek(backendEndDate.getDay(), endTimeStr)

    const dateStr = formatSuggestionDate(startDate, endDate)
    const nextWeekTag = isNextWeek(startDate) ? ' (next week)' : ''
    const title = `${dateStr}${nextWeekTag} Exact match found\n${match.teacher_name}`

    return {
      id: `suggestion-exact-${match.teacher_id}-${slot.day_of_week}-${slot.start}`,
      title,
      start: startDate.toISOString(),
      end: endDate.toISOString(),
      backgroundColor: 'var(--accent-gold)',
      borderColor: 'var(--accent-gold)',
      textColor: '#fff',
      classNames: ['suggestion-exact'],
      extendedProps: {
        isSuggestion: true,
        teacherId: match.teacher_id,
        teacherName: match.teacher_name,
        score: match.score,
      },
    }
  }

  const createAlternativeEvent = (alt: BookingAlternative, slot: WeeklySlot): EventInput => {
    const backendDate = new Date(alt.start_time)
    const timeStr = formatTimeString(backendDate)
    const startDate = getDateForDayOfWeek(backendDate.getDay(), timeStr)

    const backendEndDate = new Date(alt.end_time)
    const endTimeStr = formatTimeString(backendEndDate)
    const endDate = getDateForDayOfWeek(backendEndDate.getDay(), endTimeStr)

    return {
      id: `suggestion-alt-${alt.teacher_id}-${slot.day_of_week}-${slot.start}`,
      title: `${alt.teacher_name} (Score: ${alt.score})`,
      start: startDate.toISOString(),
      end: endDate.toISOString(),
      backgroundColor: 'var(--accent-gold-soft)',
      borderColor: 'var(--accent-gold)',
      textColor: 'var(--ink-primary)',
      classNames: ['suggestion-alt'],
      extendedProps: {
        isSuggestion: true,
        teacherId: alt.teacher_id,
        teacherName: alt.teacher_name,
        score: alt.score,
      },
    }
  }

  const suggestionEvents = computed<EventInput[]>(() => {
    if (!suggestions.value || !showDetailedResults.value) return []
    const result: EventInput[] = []
    for (const slotResult of suggestions.value.results) {
      if (slotResult.exact_match) {
        result.push(createExactMatchEvent(slotResult.exact_match, slotResult.slot))
      }
      if (slotResult.alternatives) {
        for (const alt of slotResult.alternatives) {
          result.push(createAlternativeEvent(alt, slotResult.slot))
        }
      }
    }
    return result
  })

  const cartEvents = computed<EventInput[]>(() => {
    return cartItems.value.map((item) => {
      const startDate = new Date(item.start_time)
      const endDate = new Date(item.end_time)
      return {
        id: `cart-${item.id}`,
        title: `${item.teacher_name} (In Cart)`,
        start: startDate.toISOString(),
        end: endDate.toISOString(),
        editable: false,
        backgroundColor: 'var(--accent-sage)',
        borderColor: 'var(--accent-sage)',
        textColor: '#fff',
        classNames: ['cart-event'],
        extendedProps: {
          isCartItem: true,
          cartId: item.id,
          teacherId: item.teacher_id,
          teacherName: item.teacher_name,
        },
      }
    })
  })

  // Confirmed Bookings Events
  const fetchConfirmedBookings = async () => {
    isLoadingBookings.value = true
    try {
      confirmedBookings.value = await bookingApi.listAll()
    } catch (error) {
      console.error('Failed to fetch bookings:', error)
    } finally {
      isLoadingBookings.value = false
    }
  }

  const bookedEvents = computed<EventInput[]>(() => {
    return confirmedBookings.value.map((booking) => {
      const startDate = new Date(booking.start_time)
      const endDate = new Date(booking.end_time)
      return {
        id: `booked-${booking.id}`,
        title: 'Booked',
        start: startDate.toISOString(),
        end: endDate.toISOString(),
        editable: false,
        backgroundColor: 'var(--text-muted)',
        borderColor: 'var(--border-subtle)',
        textColor: 'var(--text-secondary)',
        classNames: ['booked-event'],
        extendedProps: {
          isBooked: true,
          bookingId: booking.id,
          teacherId: booking.teacher_id,
          subjectId: booking.subject_id,
        },
      }
    })
  })

  const allEvents = computed<EventInput[]>(() => {
    const filteredCartEvents = cartEvents.value.filter((event) => {
      const eventId = event.id
      if (!eventId || typeof eventId !== 'string') return false
      const cartIdStr = eventId.replace('cart-', '')
      const cartId = parseInt(cartIdStr, 10)
      if (isNaN(cartId)) return false
      const cartItem = cartItems.value.find((item) => item.id === cartId)
      if (!cartItem) return false

      // If a teacher is selected, show ALL cart events for that teacher (regardless of subject)
      // This prevents double-booking the same teacher across different subjects
      if (selectedTeacherId.value) {
        return cartItem.teacher_id === selectedTeacherId.value
      }

      // If no teacher selected but a subject is selected:
      // Show cart events that either:
      // 1. Match the current subject, OR
      // 2. Are for a teacher who teaches the current subject (to prevent double-booking)
      if (selectedSubjectId.value) {
        // Match current subject
        if (cartItem.subject_id === selectedSubjectId.value) {
          return true
        }
        // Teacher teaches current subject (check if teacher is in filteredTeachers)
        const teachesCurrentSubject = filteredTeachers.value.some(
          (t) => t.id === cartItem.teacher_id
        )
        return teachesCurrentSubject
      }

      return false
    })

    // Get teacher IDs that are already in cart (all subjects)
    const cartTeacherIds = new Set(cartItems.value.map((item) => item.teacher_id))

    // Filter suggestion events:
    // 1. Exclude teachers already in cart (any subject)
    // 2. If a teacher is selected, only show their suggestions
    const filteredSuggestionEvents = suggestionEvents.value.filter((event) => {
      const props = event.extendedProps
      const teacherId = props?.teacherId

      // Skip if teacher is already in cart (any subject)
      if (teacherId && cartTeacherIds.has(teacherId)) {
        return false
      }

      // If teacher is selected, only show their suggestions
      if (selectedTeacherId.value) {
        return teacherId === selectedTeacherId.value
      }

      return true
    })

    // Filter booked events:
    // Priority: teacher > subject
    // If teacher selected, show ALL their bookings (any subject) - prevents double-booking same teacher
    // If no teacher but subject selected:
    //   - Show bookings for that subject
    //   - PLUS show bookings for teachers who teach the current subject (any subject they're booked for)
    const filteredBookedEvents = bookedEvents.value.filter((event) => {
      const props = event.extendedProps
      if (!props) return false

      // Priority 1: If teacher selected, show all their bookings
      if (selectedTeacherId.value) {
        return props.teacherId === selectedTeacherId.value
      }

      // Priority 2: If subject selected (no teacher selected yet)
      if (selectedSubjectId.value) {
        // Show bookings for current subject
        if (props.subjectId === selectedSubjectId.value) {
          return true
        }
        // Show bookings for teachers who teach the current subject (even if booking is for different subject)
        // This prevents double-booking the same teacher across subjects
        const teachesCurrentSubject = filteredTeachers.value.some((t) => t.id === props.teacherId)
        return teachesCurrentSubject
      }

      return false
    })

    return [
      ...events.value,
      ...filteredSuggestionEvents,
      ...filteredCartEvents,
      ...filteredBookedEvents,
    ]
  })

  // Cart Functions
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

  const showSuccess = (msg: string, duration?: number) => {
    successMessage.value = msg
    if (duration)
      setTimeout(() => {
        successMessage.value = ''
      }, duration)
  }

  const showError = (error: unknown, msg: string) => {
    console.error(msg, error)
    errorMessage.value = msg
  }

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
    if (cartIsLoading.value) return

    cartIsLoading.value = true
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

      const successfulItemIds = new Set<number>()
      const conflictItemIds = new Set<number>() // Already booked (409)
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

      // Remove successful and conflict items (conflict = already booked, no point keeping)
      const idsToRemove = new Set([...successfulItemIds, ...conflictItemIds])
      cartItems.value = cartItems.value.filter((item) => !idsToRemove.has(item.id))
      saveCart()

      const conflictCount = conflictItemIds.size
      const successCount = successfulItemIds.size

      // Check conflicts FIRST (they take priority)
      if (conflictCount > 0) {
        errorMessage.value = 'You already booked this subject'
        setTimeout(() => {
          errorMessage.value = ''
        }, 4000)
      } else if (successCount > 0 && otherFailedCount === 0) {
        successMessage.value = `Successfully confirmed ${successCount} booking(s)!`
      } else if (otherFailedCount > 0) {
        errorMessage.value = `${otherFailedCount} booking(s) failed. ${successCount} confirmed.`
        setTimeout(() => {
          errorMessage.value = ''
        }, 4000)
      }
    } catch (error) {
      errorMessage.value = getErrorMessage(error, 'Failed to confirm bookings')
    } finally {
      cartIsLoading.value = false
    }
  }

  // Booking Functions
  const resetBookingState = () => {
    showDetailedResults.value = false
    suggestions.value = null
    events.value = []
    activeTab.value = 'manual'
  }

  const fetchAvailability = async (): Promise<void> => {
    try {
      const data = await availabilityApi.getAll()
      availabilityCache.value = transformBackendAvailability(data)
    } catch (error) {
      showError(error, 'Failed to load availability')
    }
  }

  const fetchTeachersBySubject = async (subjectId: number): Promise<void> => {
    try {
      isLoadingTeachers.value = true
      filteredTeachers.value = await teacherApi.getBySubject(subjectId)
    } finally {
      isLoadingTeachers.value = false
    }
  }

  const fetchSubjects = async () => {
    const data = await subjectApi.getAll()
    subjects.value = data
  }

  const fetchBranches = async () => {
    const data = await branchApi.getAll()
    branches.value = data
  }

  const updateBusinessHours = (teacherId: number | null): void => {
    if (teacherId) {
      const cached = availabilityCache.value.get(teacherId)
      if (cached) {
        businessHours.value = cached.map((slot) => ({
          daysOfWeek: [slot.day_of_week],
          startTime: slot.start,
          endTime: slot.end,
        }))
      }
    } else {
      businessHours.value = []
    }
  }

  const getAggregatedAvailability = (teacherIds: number[]): WeeklySlot[] => {
    const allSlots: WeeklySlot[] = []
    for (const teacherId of teacherIds) {
      const cached = availabilityCache.value.get(teacherId)
      if (cached) {
        allSlots.push(...cached)
      }
    }
    return allSlots
  }

  const updateBusinessHoursFromTeachers = (teachers: Teacher[]): void => {
    const teacherIds = teachers.map((t) => t.id)
    const slots = getAggregatedAvailability(teacherIds)

    if (slots.length === 0) {
      businessHours.value = []
      return
    }

    businessHours.value = slots.map((slot) => ({
      daysOfWeek: [slot.day_of_week],
      startTime: slot.start,
      endTime: slot.end,
    }))
  }

  const addEvent = (event: EventInput): void => {
    events.value = [...events.value, event]
  }

  const addToCartDirectly = (
    teacherId: number,
    teacherName: string,
    startTime: string,
    endTime: string,
    subjectId?: number,
    branchId?: number
  ): void => {
    if (isAddingToCart.value) return

    const effectiveSubjectId = subjectId ?? selectedSubjectId.value
    const effectiveBranchId = branchId ?? selectedBranchId.value
    if (!effectiveBranchId || !effectiveSubjectId) {
      showError(
        new Error('Branch and subject must be selected'),
        'Please select branch and subject before adding to cart'
      )
      return
    }

    isAddingToCart.value = true

    const newStartDate = new Date(startTime)
    const newEndDate = new Date(endTime)
    const validation = validateDateRange(newStartDate, newEndDate)
    if (!validation.isValid) {
      showError(new Error(validation.error || 'Invalid date'), 'Please select valid time slots')
      isAddingToCart.value = false
      return
    }

    if (hasOverlapWithCart(startTime, endTime, teacherId, cartItems.value)) {
      showError(
        new Error('Time slot already in cart'),
        'This time slot is already in your cart for this teacher'
      )
      isAddingToCart.value = false
      return
    }

    const subject = subjects.value.find((s) => s.id === effectiveSubjectId)
    const branch = branches.value.find((b) => b.id === effectiveBranchId)
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
    events.value = []
    isAddingToCart.value = false
  }

  // Debounced handler for subject changes
  const handleSubjectChange = debounce(async (newSubjectId: number | null) => {
    currentSubjectChangePromise = (async () => {
      if (newSubjectId) {
        if (availabilityPromise) {
          await availabilityPromise
        }
        await fetchTeachersBySubject(newSubjectId)
        updateBusinessHoursFromTeachers(filteredTeachers.value)
      } else {
        filteredTeachers.value = []
        businessHours.value = []
      }
      selectedBranchId.value = null
      teacherStore.setSelectedTeacherById(null)
      preferredTeacherId.value = null
      resetBookingState()
    })()

    await currentSubjectChangePromise
  }, 200)

  // Initialize watchers and load data
  const initialize = () => {
    if (watchersInitialized) return
    watchersInitialized = true

    fetchCartItems()
    availabilityPromise = fetchAvailability()
    fetchSubjects()
    fetchBranches()
    fetchConfirmedBookings()

    watch(selectedSubjectId, (newSubjectId) => {
      handleSubjectChange(newSubjectId)
    })

    watch(selectedTeacherId, async (teacherId) => {
      if (availabilityPromise) {
        await availabilityPromise
      }
      if (teacherId) {
        updateBusinessHours(teacherId)
      } else if (selectedSubjectId.value) {
        updateBusinessHoursFromTeachers(filteredTeachers.value)
      } else {
        businessHours.value = []
      }
      resetBookingState()
    })

    watch(selectedBranchId, async (newBranchId) => {
      if (newBranchId === null) {
        teacherStore.setSelectedTeacherById(null)
        resetBookingState()
      }
    })
  }

  return {
    // UI State
    calendarRef,
    activeTab,
    isLoadingTeachers,
    isEvaluating,
    isAddingToCart,
    successMessage,
    errorMessage,

    // Selection State
    subjects,
    branches,
    filteredTeachers,
    selectedSubjectId,
    selectedBranchId,
    selectedTeacherId,
    preferredTeacherId,

    // Event & Availability
    events,
    businessHours,
    availabilityCache,
    suggestions,
    showDetailedResults,
    suggestionEvents,
    cartEvents,
    allEvents,
    bookedEvents,

    // Cart
    cartItems,
    cartIsLoading,

    // Bookings
    confirmedBookings,
    isLoadingBookings,

    // Actions
    initialize,
    resetBookingState,
    addEvent,
    addToCartDirectly,
    addToCart,
    removeItem,
    clearCart,
    submitBookings,
    fetchCartItems,
    fetchConfirmedBookings,
    showSuccess,
    showError,
  }
})
