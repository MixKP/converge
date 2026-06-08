import { ref, computed, watch } from 'vue'
import { useTeacherStore } from '../stores/teacherStore'
import { availabilityApi } from '../services/availabilityApi'
import { subjectApi } from '../services/subjectApi'
import { teacherApi } from '../services/teacherApi'
import { bookingApi } from '../services/bookingApi'
import { branchApi } from '../services/branchApi'
import type { Teacher, Branch } from '../types'
import type { EventInput, BusinessHoursInput } from '@fullcalendar/core'
import type { WeeklySlot, Subject, BookingResponse, BookingAlternative } from '../types'
import { transformBackendAvailability } from '../utils/availabilityTransform'
import { useBookingCart } from './useBookingCart'
const DAY_NAMES = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'] as const
export function useBooking() {
  const teacherStore = useTeacherStore()
  const { addToCart } = useBookingCart()

  const calendarRef = ref()
  const activeTab = ref<'manual' | 'ai'>('manual')
  const isLoadingTeachers = ref(false)
  const isEvaluating = ref(false)
  const errorMessage = ref<string>('')
  const successMessage = ref<string>('')

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
  const durationMinutes = ref<number>(60)

  const suggestions = ref<BookingResponse | null>(null)
  const showDetailedResults = ref(false)

  const manualSlots = ref<WeeklySlot[]>([])
  const newSlotDay = ref<number>(0)
  const newSlotStart = ref<string>('09:00')
  const newSlotEnd = ref<string>('10:00')

  const timeOptions = computed(() => {
    const options: string[] = []
    for (let hour = 8; hour <= 18; hour++) {
      const hourStr = String(hour).padStart(2, '0')
      options.push(`${hourStr}:00`, `${hourStr}:30`)
    }
    options.push('19:00')
    return options
  })
  const currentStep = computed(() => {
    if (!selectedSubjectId.value) return 1
    if (!selectedBranchId.value) return 2
    return 3
  })
  const canSelectTeacher = computed(() => !!selectedSubjectId.value)
  const canAddManualSlot = computed(() => newSlotStart.value < newSlotEnd.value)
  const getEffectiveSlots = (): WeeklySlot[] => {
    if (manualSlots.value.length > 0) return manualSlots.value
    return events.value.map((e) => {
      const start = new Date(e.start as Date)
      const end = new Date(e.end as Date)
      // Use Sunday-first (0=Sun, 1=Mon, ..., 6=Sat) to match backend
      const dayOfWeek = start.getDay()
      return {
        day_of_week: dayOfWeek,
        start: start.toTimeString().slice(0, 5),
        end: end.toTimeString().slice(0, 5),
      }
    })
  }
  const canEvaluate = computed(() => {
    return (
      !!selectedSubjectId.value &&
      !!selectedBranchId.value &&
      getEffectiveSlots().length > 0 &&
      !isEvaluating.value
    )
  })
  const calculatedDuration = computed(() => {
    if (events.value.length === 0) return 60
    const event = events.value[0]
    if (!event.start || !event.end) return 60
    const start = new Date(event.start as Date)
    const end = new Date(event.end as Date)
    return Math.round((end.getTime() - start.getTime()) / (1000 * 60))
  })
  // Helper: Convert day_of_week + time to proper date in current week (timezone-safe)
  const getDateForDayOfWeek = (dayOfWeek: number, timeStr: string): Date => {
    const now = new Date()
    const currentDay = now.getDay()
    const diff = dayOfWeek - currentDay
    const targetDate = new Date(now)
    targetDate.setDate(now.getDate() + diff)
    const [hours, minutes] = timeStr.split(':').map(Number)
    targetDate.setHours(hours, minutes, 0, 0)
    return targetDate
  }
  const createExactMatchEvent = (
    match: BookingAlternative,
    slot: WeeklySlot,
    index: number
  ): EventInput => {
    const startDate = getDateForDayOfWeek(slot.day_of_week, slot.start)
    const endDate = getDateForDayOfWeek(slot.day_of_week, slot.end)
    return {
      id: `suggestion-exact-${index}`,
      title: `${match.teacher_name} (Score: ${match.score})`,
      start: startDate.toISOString(),
      end: endDate.toISOString(),
      backgroundColor: 'linear-gradient(135deg, var(--accent-sage) 0%, var(--accent-mint) 100%)',
      borderColor: 'var(--accent-sage)',
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
    index: number
  ): EventInput => {
    const startDate = getDateForDayOfWeek(slot.day_of_week, slot.start)
    const endDate = getDateForDayOfWeek(slot.day_of_week, slot.end)
    return {
      id: `suggestion-alt-${index}`,
      title: `${alt.teacher_name} (Score: ${alt.score})`,
      start: startDate.toISOString(),
      end: endDate.toISOString(),
      backgroundColor: 'var(--accent-sage-soft)',
      borderColor: 'var(--accent-sage)',
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
  const allEvents = computed<EventInput[]>(() => [...events.value, ...suggestionEvents.value])

  const getSlotLabel = (slot: WeeklySlot): string => {
    return `${DAY_NAMES[slot.day_of_week]} ${slot.start} - ${slot.end}`
  }
  const resetBookingState = () => {
    showDetailedResults.value = false
    suggestions.value = null
    events.value = []
    manualSlots.value = []
    activeTab.value = 'manual'
  }
  const showError = (error: unknown, defaultMessage: string): void => {
    errorMessage.value = error instanceof Error ? error.message : defaultMessage
  }
  const showSuccess = (message: string): void => {
    successMessage.value = message
    setTimeout(() => (successMessage.value = ''), 3000)
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
  const getTeachersBySubject = async (subjectId: number): Promise<Teacher[]> => {
    return await teacherApi.getBySubject(subjectId)
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

  const handleEvaluate = async (): Promise<void> => {
    if (!selectedSubjectId.value || !selectedBranchId.value) {
      showError(
        new Error('Subject and branch must be selected'),
        'Please select subject and branch'
      )
      return
    }
    isEvaluating.value = true
    errorMessage.value = ''
    successMessage.value = ''
    try {
      const effectiveSlots = getEffectiveSlots()
      const response = await bookingApi.evaluate({
        subject_id: selectedSubjectId.value,
        branch_id: selectedBranchId.value,
        preferred_slots: effectiveSlots,
        duration_minutes: calculatedDuration.value,
        preferred_teacher_id: preferredTeacherId.value ?? undefined,
      })
      suggestions.value = response
      showDetailedResults.value = true
      activeTab.value = 'ai'
      showSuccess('Booking options generated!')
    } catch (error) {
      showError(error, 'Failed to generate booking options')
    } finally {
      isEvaluating.value = false
    }
  }
  const handleAddManualSlot = (): void => {
    if (!canAddManualSlot.value) return
    manualSlots.value.push({
      day_of_week: newSlotDay.value,
      start: newSlotStart.value,
      end: newSlotEnd.value,
    })
    newSlotStart.value = '09:00'
    newSlotEnd.value = '10:00'
  }
  const removeManualSlot = (index: number): void => {
    manualSlots.value.splice(index, 1)
  }
  const addToCartDirectly = (
    teacherId: number,
    teacherName: string,
    startTime: string,
    endTime: string,
    subjectId?: number,
    branchId?: number
  ): void => {
    const effectiveSubjectId = subjectId ?? selectedSubjectId.value
    const effectiveBranchId = branchId ?? selectedBranchId.value
    if (!effectiveBranchId || !effectiveSubjectId) {
      showError(
        new Error('Branch and subject must be selected'),
        'Please select branch and subject before adding to cart'
      )
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
  }

  const initWatchers = () => {
    availabilityPromise = fetchAvailability()

    subjectApi.getAll().then((data) => {
      subjects.value = data
    })

    branchApi.getAll().then((data) => {
      branches.value = data
    })

    watch(selectedSubjectId, async (newSubjectId) => {
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
  }
  return {
    calendarRef,
    activeTab,
    isLoadingTeachers,
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
    preferredTeacherId,
    durationMinutes,
    suggestions,
    showDetailedResults,
    manualSlots,
    newSlotDay,
    newSlotStart,
    newSlotEnd,
    timeOptions,
    currentStep,
    canSelectTeacher,
    canAddManualSlot,
    canEvaluate,
    calculatedDuration,
    suggestionEvents,
    allEvents,

    getSlotLabel,
    resetBookingState,
    handleEvaluate,
    handleAddManualSlot,
    removeManualSlot,
    addToCartDirectly,
    initWatchers,
    getAggregatedAvailability,
    getTeachersBySubject,
  }
}
