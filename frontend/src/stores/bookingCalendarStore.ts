import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'
import { availabilityApi } from '../services/availabilityApi'
import { bookingApi } from '../services/bookingApi'
import { transformBackendAvailability } from '../utils/availabilityTransform'
import type { EventInput, BusinessHoursInput } from '@fullcalendar/core'
import type { WeeklySlot, BookingResponse, BookingAlternative, Booking } from '../types'
import {
  getNextDateForDayOfWeek,
  formatTimeString,
  formatSuggestionDate,
  isNextWeek,
} from '../utils/dateValidation'

import { useBookingSelectionStore } from './bookingSelectionStore'
import { useBookingCartStore } from './bookingCartStore'

export const useBookingCalendarStore = defineStore('bookingCalendar', () => {
  // UI State
  const calendarRef = ref()
  const activeTab = ref<'manual' | 'ai'>('manual')
  const isEvaluating = ref(false)
  const isAddingToCart = ref(false)
  const successMessage = ref('')
  const errorMessage = ref('')

  // Event & Availability State
  const events = ref<EventInput[]>([])
  const businessHours = ref<BusinessHoursInput>([])
  const availabilityCache = ref<Map<number, WeeklySlot[]>>(new Map())
  const suggestions = ref<BookingResponse | null>(null)
  const showDetailedResults = ref(false)

  // Confirmed Bookings State
  const confirmedBookings = ref<Booking[]>([])
  const isLoadingBookings = ref(false)

  // Internal state
  let availabilityPromise: Promise<void> | null = null
  let watchersInitialized = false

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

  // Lazy-access cart store inside computed to avoid circular dependency at module init
  const cartEvents = computed<EventInput[]>(() => {
    const cartStore = useBookingCartStore()
    return cartStore.cartItems.map((item) => {
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

  // Confirmed Bookings
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
    const selectionStore = useBookingSelectionStore()
    const cartStore = useBookingCartStore()

    const filteredCartEvents = cartStore.cartItems
      .map((item) => {
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
      .filter((event) => {
        const eventId = event.id
        if (!eventId || typeof eventId !== 'string') return false
        const cartIdStr = eventId.replace('cart-', '')
        const cartId = parseInt(cartIdStr, 10)
        if (isNaN(cartId)) return false
        const cartItem = cartStore.cartItems.find((item) => item.id === cartId)
        if (!cartItem) return false

        if (selectionStore.selectedTeacherId) {
          return cartItem.teacher_id === selectionStore.selectedTeacherId
        }

        if (selectionStore.selectedSubjectId) {
          if (cartItem.subject_id === selectionStore.selectedSubjectId) {
            return true
          }
          const teachesCurrentSubject = selectionStore.filteredTeachers.some(
            (t: { id: number }) => t.id === cartItem.teacher_id
          )
          return teachesCurrentSubject
        }

        return false
      })

    const cartTeacherIds = new Set(cartStore.cartItems.map((item) => item.teacher_id))

    const filteredSuggestionEvents = suggestionEvents.value.filter((event) => {
      const props = event.extendedProps
      const teacherId = props?.teacherId

      if (teacherId && cartTeacherIds.has(teacherId)) {
        return false
      }

      if (selectionStore.selectedTeacherId) {
        return teacherId === selectionStore.selectedTeacherId
      }

      return true
    })

    const filteredBookedEvents = bookedEvents.value.filter((event) => {
      const props = event.extendedProps
      if (!props) return false

      if (selectionStore.selectedTeacherId) {
        return props.teacherId === selectionStore.selectedTeacherId
      }

      if (selectionStore.selectedSubjectId) {
        if (props.subjectId === selectionStore.selectedSubjectId) {
          return true
        }
        const teachesCurrentSubject = selectionStore.filteredTeachers.some(
          (t: { id: number }) => t.id === props.teacherId
        )
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

  // Actions
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

  const updateBusinessHoursFromTeachers = (teachers: { id: number }[]): void => {
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

  // Initialize watchers and load data
  const initialize = () => {
    if (watchersInitialized) return
    watchersInitialized = true

    availabilityPromise = fetchAvailability()
    fetchConfirmedBookings()

    watch(
      () => useBookingSelectionStore().selectedSubjectId,
      () => {
        resetBookingState()
      }
    )

    watch(
      () => useBookingSelectionStore().selectedTeacherId,
      async (teacherId) => {
        if (availabilityPromise) {
          await availabilityPromise
        }
        if (teacherId) {
          updateBusinessHours(teacherId)
        } else if (useBookingSelectionStore().selectedSubjectId) {
          updateBusinessHoursFromTeachers(useBookingSelectionStore().filteredTeachers)
        } else {
          businessHours.value = []
        }
        resetBookingState()
      }
    )

    watch(
      () => useBookingSelectionStore().filteredTeachers,
      (teachers) => {
        updateBusinessHoursFromTeachers(teachers)
      }
    )

    watch(
      () => useBookingSelectionStore().selectedBranchId,
      async (newBranchId) => {
        if (newBranchId === null) {
          useBookingSelectionStore().selectedTeacherId = null
          resetBookingState()
        }
      }
    )
  }

  return {
    // UI State
    calendarRef,
    activeTab,
    isEvaluating,
    isAddingToCart,
    successMessage,
    errorMessage,

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

    // Bookings
    confirmedBookings,
    isLoadingBookings,

    // Actions
    initialize,
    resetBookingState,
    addEvent,
    fetchConfirmedBookings,
    showSuccess,
    showError,
  }
})
