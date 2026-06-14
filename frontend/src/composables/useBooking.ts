import { ref, computed, watch } from 'vue'
import { useTeacherStore } from '../stores/teacherStore'
import { availabilityApi } from '../services/availabilityApi'
import { subjectApi } from '../services/subjectApi'
import { teacherApi } from '../services/teacherApi'
import { branchApi } from '../services/branchApi'
import type { Teacher, Branch } from '../types'
import type { EventInput, BusinessHoursInput } from '@fullcalendar/core'
import type { WeeklySlot, Subject, BookingResponse, BookingAlternative } from '../types'
import { transformBackendAvailability } from '../utils/availabilityTransform'
import { useBookingCart } from './useBookingCart'
import { debounce } from '../utils/common'
import {
  validateDateRange,
  hasOverlapWithCart,
  formatTimeString,
  getNextDateForDayOfWeek,
  isNextWeek,
  formatSuggestionDate,
} from '../utils/dateValidation'
import { useMessages } from './useMessages'

export function useBooking() {
  const teacherStore = useTeacherStore()
  const { addToCart, cartItems } = useBookingCart()
  const { successMessage, errorMessage, showSuccess, showError } = useMessages()

  const calendarRef = ref()
  const activeTab = ref<'manual' | 'ai'>('manual')
  const isLoadingTeachers = ref(false)
  const isEvaluating = ref(false)
  const isAddingToCart = ref(false) // Prevent multiple add to cart clicks

  // Track availability loading promise
  let availabilityPromise: Promise<void> | null = null

  const events = ref<EventInput[]>([])
  const businessHours = ref<BusinessHoursInput>([])
  const availabilityCache = ref<Map<number, WeeklySlot[]>>(new Map())

  const subjects = ref<Subject[]>([])
  const branches = ref<Branch[]>([])
  const filteredTeachers = ref<Teacher[]>([])
  const selectedSubjectId = ref<number | null>(null)
  const selectedBranchId = ref<number | null>(null)
  const selectedTeacherId = computed({
    get: () => teacherStore.selectedTeacherId,
    set: (val) => teacherStore.setSelectedTeacherById(val),
  })
  const preferredTeacherId = ref<number | null>(null)

  const suggestions = ref<BookingResponse | null>(null)
  const showDetailedResults = ref(false)

  // Helper: Convert day_of_week + time to proper date (always next occurrence)
  const getDateForDayOfWeek = (dayOfWeek: number, timeStr: string): Date => {
    return getNextDateForDayOfWeek(dayOfWeek, timeStr)
  }
  const createExactMatchEvent = (
    match: BookingAlternative,
    slot: WeeklySlot,
    _index: number
  ): EventInput => {
    // Extract time from backend's start_time and place it in current/next week
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
      backgroundColor:
        'linear-gradient(135deg, var(--accent-gold) 0%, var(--accent-gold-light) 100%)',
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
  const createAlternativeEvent = (
    alt: BookingAlternative,
    slot: WeeklySlot,
    _index: number
  ): EventInput => {
    // Extract time from backend's start_time and place it in current/next week
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
    let index = 0
    for (const slotResult of suggestions.value.results) {
      if (slotResult.exact_match) {
        result.push(createExactMatchEvent(slotResult.exact_match, slotResult.slot, index++))
      }
      if (slotResult.alternatives) {
        for (const alt of slotResult.alternatives) {
          result.push(createAlternativeEvent(alt, slotResult.slot, index++))
        }
      }
    }
    return result
  })

  // Convert cart items to calendar events
  const cartEvents = computed<EventInput[]>(() => {
    return cartItems.value.map((item) => {
      const startDate = new Date(item.start_time)
      const endDate = new Date(item.end_time)
      return {
        id: `cart-${item.id}`,
        title: `${item.teacher_name} (In Cart)`,
        start: startDate.toISOString(),
        end: endDate.toISOString(),
        editable: false, // Cart items cannot be edited
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

  const allEvents = computed<EventInput[]>(() => {
    // Filter cart events to only show those matching the current subject selection
    const filteredCartEvents = selectedSubjectId.value
      ? cartEvents.value.filter((event) => {
          // Safely extract cart ID from event ID
          const eventId = event.id
          if (!eventId || typeof eventId !== 'string') return false
          const cartIdStr = eventId.replace('cart-', '')
          const cartId = parseInt(cartIdStr, 10)
          if (isNaN(cartId)) return false
          const cartItem = cartItems.value.find((item) => item.id === cartId)
          return cartItem?.subject_id === selectedSubjectId.value
        })
      : []

    return [...events.value, ...suggestionEvents.value, ...filteredCartEvents]
  })

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
    // Prevent multiple concurrent calls
    if (isAddingToCart.value) {
      return
    }

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

    // Validate date strings
    const newStartDate = new Date(startTime)
    const newEndDate = new Date(endTime)
    const validation = validateDateRange(newStartDate, newEndDate)
    if (!validation.isValid) {
      showError(new Error(validation.error || 'Invalid date'), 'Please select valid time slots')
      isAddingToCart.value = false
      return
    }

    // Check for duplicate time slot in cart (same teacher + overlapping time)
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
    // Clear events after adding to cart
    events.value = []
    isAddingToCart.value = false
  }

  let watchersInitialized = false
  let currentSubjectChangePromise: Promise<void> | null = null

  // Debounced handler for subject changes
  const handleSubjectChange = debounce(async (newSubjectId: number | null) => {
    // Cancel any pending subject change
    if (currentSubjectChangePromise) {
      // We'll let it complete but won't use its result
    }

    currentSubjectChangePromise = (async () => {
      if (newSubjectId) {
        // Wait for availability to load before updating business hours
        if (availabilityPromise) {
          await availabilityPromise
        }
        await fetchTeachersBySubject(newSubjectId)
        updateBusinessHoursFromTeachers(filteredTeachers.value)
      } else {
        filteredTeachers.value = []
        businessHours.value = []
      }
      // Reset branch and teacher when subject changes
      selectedBranchId.value = null
      teacherStore.setSelectedTeacherById(null)
      preferredTeacherId.value = null
      resetBookingState()
    })()

    await currentSubjectChangePromise
  }, 200) // 200ms debounce to prevent rapid API calls

  const initWatchers = () => {
    // Prevent multiple initialization
    if (watchersInitialized) return
    watchersInitialized = true

    availabilityPromise = fetchAvailability()

    subjectApi.getAll().then((data) => {
      subjects.value = data
    })

    branchApi.getAll().then((data) => {
      branches.value = data
    })

    watch(selectedSubjectId, (newSubjectId) => {
      handleSubjectChange(newSubjectId)
    })

    watch(selectedTeacherId, async (teacherId) => {
      // Wait for availability to load before updating business hours
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
      // Reset teacher and events when branch changes
      if (newBranchId === null) {
        teacherStore.setSelectedTeacherById(null)
        resetBookingState()
      }
    })
  }
  return {
    calendarRef,
    activeTab,
    isEvaluating,
    errorMessage,
    successMessage,
    events,
    businessHours,
    subjects,
    branches,
    filteredTeachers,
    selectedSubjectId,
    selectedBranchId,
    selectedTeacherId,
    suggestions,
    showDetailedResults,
    suggestionEvents,
    allEvents,

    resetBookingState,
    addEvent,
    addToCartDirectly,
    initWatchers,
    showSuccess,
    showError,
  }
}
