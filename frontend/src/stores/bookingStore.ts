import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'
import { availabilityApi } from '../services/availabilityApi'
import { bookingApi } from '../services/bookingApi'
import { subjectApi } from '../services/subjectApi'
import { teacherApi } from '../services/teacherApi'
import { branchApi } from '../services/branchApi'
import { transformBackendAvailability } from '../utils/availabilityTransform'
import { debounce } from '../utils/common'
import { useNotification } from '../composables/useNotification'
import type { EventInput, BusinessHoursInput } from '@fullcalendar/core'
import type {
  WeeklySlot,
  BookingResponse,
  BookingAlternative,
  Booking,
  Subject,
  Teacher,
  Branch,
  CartItem,
} from '../types'
import {
  getNextDateForDayOfWeek,
  formatTimeString,
  formatSuggestionDate,
  isNextWeek,
} from '../utils/dateValidation'

import { useTeacherStore } from './teacherStore'
import { useCartStore } from './cartStore'

export const useBookingStore = defineStore('booking', () => {
  const teacherStore = useTeacherStore()
  const { showError } = useNotification()

  // Selection State
  const subjects = ref<Subject[]>([])
  const branches = ref<Branch[]>([])
  const filteredTeachers = ref<Teacher[]>([])
  const selectedSubjectId = ref<number | null>(null)
  const selectedBranchId = ref<number | null>(null)
  const isLoadingTeachers = ref(false)

  const selectedTeacherId = computed({
    get: () => teacherStore.selectedTeacherId,
    set: (val) => teacherStore.setSelectedTeacherById(val),
  })

  // Calendar & Availability State
  const calendarRef = ref()
  const isEvaluating = ref(false)
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

  // --- Selection actions ---
  const fetchSubjects = async () => {
    subjects.value = await subjectApi.getAll()
  }

  const fetchBranches = async () => {
    branches.value = await branchApi.getAll()
  }

  const fetchTeachersBySubject = async (subjectId: number): Promise<void> => {
    try {
      isLoadingTeachers.value = true
      filteredTeachers.value = await teacherApi.getBySubject(subjectId)
    } finally {
      isLoadingTeachers.value = false
    }
  }

  const handleSubjectChange = debounce(async (newSubjectId: number | null) => {
    if (newSubjectId) {
      await fetchTeachersBySubject(newSubjectId)
    } else {
      filteredTeachers.value = []
    }
    selectedBranchId.value = null
    teacherStore.setSelectedTeacherById(null)
  }, 200)

  watch(selectedSubjectId, (newSubjectId) => {
    handleSubjectChange(newSubjectId)
  })

  // --- Suggestion events ---
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

  // --- Cart & booked event mappers (shared by allEvents) ---
  const mapCartItemToEvent = (item: CartItem): EventInput => {
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
  }

  const mapBookingToEvent = (booking: Booking): EventInput => {
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
  }

  const bookedEvents = computed<EventInput[]>(() => confirmedBookings.value.map(mapBookingToEvent))

  const allEvents = computed<EventInput[]>(() => {
    // Lazy-access cart store to avoid circular dependency at module init
    const cartStore = useCartStore()

    const filteredCartEvents = cartStore.cartItems.map(mapCartItemToEvent).filter((event) => {
      const eventId = event.id
      if (!eventId || typeof eventId !== 'string') return false
      const cartId = parseInt(eventId.replace('cart-', ''), 10)
      if (isNaN(cartId)) return false
      const cartItem = cartStore.cartItems.find((item) => item.id === cartId)
      if (!cartItem) return false

      if (selectedTeacherId.value) {
        return cartItem.teacher_id === selectedTeacherId.value
      }

      if (selectedSubjectId.value) {
        if (cartItem.subject_id === selectedSubjectId.value) {
          return true
        }
        return filteredTeachers.value.some((t) => t.id === cartItem.teacher_id)
      }

      return false
    })

    const cartTeacherIds = new Set(cartStore.cartItems.map((item) => item.teacher_id))

    const filteredSuggestionEvents = suggestionEvents.value.filter((event) => {
      const teacherId = event.extendedProps?.teacherId

      if (teacherId && cartTeacherIds.has(teacherId)) {
        return false
      }

      if (selectedTeacherId.value) {
        return teacherId === selectedTeacherId.value
      }

      return true
    })

    const filteredBookedEvents = bookedEvents.value.filter((event) => {
      const props = event.extendedProps
      if (!props) return false

      if (selectedTeacherId.value) {
        return props.teacherId === selectedTeacherId.value
      }

      if (selectedSubjectId.value) {
        if (props.subjectId === selectedSubjectId.value) {
          return true
        }
        return filteredTeachers.value.some((t) => t.id === props.teacherId)
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

  // --- Availability actions ---
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

  // --- Confirmed bookings ---
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

  const resetBookingState = () => {
    showDetailedResults.value = false
    suggestions.value = null
    events.value = []
  }

  // Initialize watchers and load data (guarded to run once)
  const initialize = () => {
    if (watchersInitialized) return
    watchersInitialized = true

    availabilityPromise = fetchAvailability()
    fetchConfirmedBookings()

    watch(selectedSubjectId, () => {
      resetBookingState()
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

    watch(filteredTeachers, (teachers) => {
      updateBusinessHoursFromTeachers(teachers)
    })

    watch(selectedBranchId, (newBranchId) => {
      if (newBranchId === null) {
        selectedTeacherId.value = null
        resetBookingState()
      }
    })
  }

  return {
    // Selection State
    subjects,
    branches,
    filteredTeachers,
    selectedSubjectId,
    selectedBranchId,
    selectedTeacherId,
    isLoadingTeachers,

    // Calendar & Availability State
    calendarRef,
    isEvaluating,
    events,
    businessHours,
    availabilityCache,
    suggestions,
    showDetailedResults,
    suggestionEvents,
    allEvents,
    bookedEvents,

    // Confirmed Bookings
    confirmedBookings,
    isLoadingBookings,

    // Actions
    fetchSubjects,
    fetchBranches,
    fetchTeachersBySubject,
    handleSubjectChange,
    fetchConfirmedBookings,
    resetBookingState,
    initialize,
  }
})
