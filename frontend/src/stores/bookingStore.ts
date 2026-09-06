import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'
import { availabilityApi } from '../services/availabilityApi'
import { bookingApi } from '../services/bookingApi'
import { subjectApi } from '../services/subjectApi'
import { teacherApi } from '../services/teacherApi'
import { branchApi } from '../services/branchApi'
import { userApi } from '../services/userApi'
import {
  businessHoursForTeachers,
  transformBackendAvailability,
} from '../utils/availabilityTransform'
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

  const isEvaluating = ref(false)
  const events = ref<EventInput[]>([])
  const businessHours = ref<BusinessHoursInput>([])
  const availabilityCache = ref<Map<number, WeeklySlot[]>>(new Map())

  const isAvailabilityLoaded = ref(false)
  const suggestions = ref<BookingResponse | null>(null)
  const showDetailedResults = ref(false)

  const confirmedBookings = ref<Booking[]>([])

  let availabilityPromise: Promise<void> | null = null
  let dataFetched = false
  let subjectRequestId = 0

  const fetchSubjects = async () => {
    subjects.value = await subjectApi.getAll()
  }

  const fetchBranches = async () => {
    branches.value = await branchApi.getAll()
  }

  const fetchStudents = async () => {
    students.value = await userApi.listStudents()
  }

  const handleSubjectChange = async (
    subjectId: number | null,
    requestId: number
  ): Promise<void> => {
    selectedBranchId.value = null
    teacherStore.setSelectedTeacherById(null)

    if (!subjectId) {
      filteredTeachers.value = []
      return
    }

    try {
      isLoadingTeachers.value = true
      const teachers = await teacherApi.getBySubject(subjectId)
      if (requestId !== subjectRequestId) return
      filteredTeachers.value = teachers
    } finally {
      if (requestId === subjectRequestId) isLoadingTeachers.value = false
    }
  }

  watch(selectedSubjectId, (newSubjectId) => {
    const requestId = ++subjectRequestId
    resetBookingState()
    void handleSubjectChange(newSubjectId, requestId)
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

  const matchesCurrentSelection = (teacherId: number, subjectId: number): boolean => {
    if (selectedTeacherId.value) return teacherId === selectedTeacherId.value
    if (!selectedSubjectId.value) return false
    return (
      subjectId === selectedSubjectId.value ||
      filteredTeachers.value.some((teacher) => teacher.id === teacherId)
    )
  }

  const allEvents = computed<EventInput[]>(() => {
    const cartStore = useCartStore()

    const filteredCartEvents = cartStore.cartItems
      .filter((item) => matchesCurrentSelection(item.teacher_id, item.subject_id))
      .map(mapCartItemToEvent)

    const filteredBookedEvents = confirmedBookings.value
      .filter((booking) => matchesCurrentSelection(booking.teacher_id, booking.subject_id))
      .map(mapBookingToEvent)

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
    businessHours.value = businessHoursForTeachers(
      availabilityCache.value,
      teacherId === null ? [] : [teacherId]
    )
  }

  const updateBusinessHoursFromTeachers = (teachers: { id: number }[]): void => {
    businessHours.value = businessHoursForTeachers(
      availabilityCache.value,
      teachers.map((teacher) => teacher.id)
    )
  }

  const fetchConfirmedBookings = async () => {
    try {
      confirmedBookings.value = await bookingApi.list()
    } catch (error) {
      console.error('Failed to fetch bookings:', error)
    }
  }

  const resetBookingState = () => {
    showDetailedResults.value = false
    suggestions.value = null
    events.value = []
  }

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

    isEvaluating,
    events,
    businessHours,
    availabilityCache,
    isAvailabilityLoaded,
    suggestions,
    showDetailedResults,
    allEvents,

    confirmedBookings,
    fetchSubjects,
    fetchBranches,
    fetchStudents,
    fetchConfirmedBookings,
    resetBookingState,
    initialize,
  }
})
