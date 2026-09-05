import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'
import { availabilityApi } from '../services/availabilityApi'
import { bookingApi } from '../services/bookingApi'
import { subjectApi } from '../services/subjectApi'
import { teacherApi } from '../services/teacherApi'
import { branchApi } from '../services/branchApi'
import { userApi } from '../services/userApi'
import { transformBackendAvailability } from '../utils/availabilityTransform'
import { debounce } from '../utils/common'
import { useNotification } from '../composables/useNotification'
import type { EventInput, BusinessHoursInput } from '@fullcalendar/core'
import type {
  WeeklySlot,
  BookingResponse,
  Booking,
  Subject,
  Teacher,
  Branch,
  CartItem,
  AuthUser,
} from '../types'

import { useTeacherStore } from './teacherStore'
import { useCartStore } from './cartStore'

export const useBookingStore = defineStore('booking', () => {
  const teacherStore = useTeacherStore()
  const { showError } = useNotification()

  const subjects = ref<Subject[]>([])
  const branches = ref<Branch[]>([])
  const students = ref<AuthUser[]>([])
  const filteredTeachers = ref<Teacher[]>([])

  const activeBranches = computed(() => branches.value.filter((b) => b.status === 'active'))

  const selectedStudentId = ref<number | null>(null)
  const selectedSubjectId = ref<number | null>(null)
  const selectedBranchId = ref<number | null>(null)
  const isLoadingTeachers = ref(false)
  const requiredGender = ref<'male' | 'female' | 'lgbtq+' | null>(null)

  const genderFilteredTeachers = computed(() => {
    if (!requiredGender.value) return filteredTeachers.value
    return filteredTeachers.value.filter((t) => t.gender === requiredGender.value)
  })

  const selectedStudent = computed<AuthUser | null>(
    () => students.value.find((s) => s.id === selectedStudentId.value) ?? null
  )

  const selectedTeacherId = computed({
    get: () => teacherStore.selectedTeacherId,
    set: (val) => teacherStore.setSelectedTeacherById(val),
  })

  const calendarRef = ref()
  const isEvaluating = ref(false)
  const events = ref<EventInput[]>([])
  const businessHours = ref<BusinessHoursInput>([])
  const availabilityCache = ref<Map<number, WeeklySlot[]>>(new Map())

  const isAvailabilityLoaded = ref(false)
  const suggestions = ref<BookingResponse | null>(null)
  const showDetailedResults = ref(false)

  const confirmedBookings = ref<Booking[]>([])
  const isLoadingBookings = ref(false)

  let availabilityPromise: Promise<void> | null = null
  let dataFetched = false

  const fetchSubjects = async () => {
    subjects.value = await subjectApi.getAll()
  }

  const fetchBranches = async () => {
    branches.value = await branchApi.getAll()
  }

  const fetchStudents = async () => {
    students.value = await userApi.listStudents()
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

    return [...events.value, ...filteredCartEvents, ...filteredBookedEvents]
  })

  const fetchAvailability = async (): Promise<void> => {
    try {
      const data = await availabilityApi.getAll()
      availabilityCache.value = transformBackendAvailability(data)
    } catch (error) {
      showError(error, 'Failed to load availability')
    } finally {
      isAvailabilityLoaded.value = true
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

  const fetchConfirmedBookings = async () => {
    isLoadingBookings.value = true
    try {
      confirmedBookings.value = await bookingApi.list()
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
      updateBusinessHoursFromTeachers(genderFilteredTeachers.value)
    } else {
      businessHours.value = []
    }
    resetBookingState()
  })

  watch(genderFilteredTeachers, (teachers) => {
    updateBusinessHoursFromTeachers(teachers)
  })

  watch(requiredGender, () => {
    if (selectedTeacherId.value) {
      const match = filteredTeachers.value.find((t) => t.id === selectedTeacherId.value)
      if (match && requiredGender.value && match.gender !== requiredGender.value) {
        teacherStore.setSelectedTeacherById(null)
        resetBookingState()
      }
    }
  })

  watch(selectedBranchId, (newBranchId) => {
    if (newBranchId === null) {
      selectedTeacherId.value = null
      resetBookingState()
    }
  })

  const initialize = () => {
    if (dataFetched) return
    dataFetched = true

    availabilityPromise = fetchAvailability()
    fetchConfirmedBookings()
  }

  return {
    subjects,
    branches,
    activeBranches,
    students,
    filteredTeachers,
    genderFilteredTeachers,
    selectedStudentId,
    selectedStudent,
    selectedSubjectId,
    selectedBranchId,
    selectedTeacherId,
    isLoadingTeachers,
    requiredGender,

    calendarRef,
    isEvaluating,
    events,
    businessHours,
    availabilityCache,
    isAvailabilityLoaded,
    suggestions,
    showDetailedResults,
    allEvents,

    confirmedBookings,
    isLoadingBookings,

    fetchSubjects,
    fetchBranches,
    fetchStudents,
    fetchTeachersBySubject,
    handleSubjectChange,
    fetchConfirmedBookings,
    resetBookingState,
    initialize,
  }
})
