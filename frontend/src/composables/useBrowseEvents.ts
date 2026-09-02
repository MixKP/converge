import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useBookingStore } from '../stores/bookingStore'
import { useCartStore } from '../stores/cartStore'
import { subtractSpans, type Span } from '../utils/intervals'
import type { EventInput } from '@fullcalendar/core'

export interface BrowseTeacher {
  id: number
  name: string
}

export interface BrowseGroup {
  start: Date
  end: Date
  teachers: BrowseTeacher[]
}

export interface VisibleRange {
  start: Date
  end: Date
}

const BROWSE_BACKGROUND = 'rgba(45, 74, 62, 0.10)'
const BROWSE_BORDER = 'var(--primary-indigo)'
const BROWSE_TEXT = 'var(--primary-indigo)'

const MIN_REMAINDER_MS = 30 * 60 * 1000

const normalizeTime = (value: string): string => {
  const [hours = '', minutes = '00'] = value.split(':')
  return `${hours.padStart(2, '0')}:${minutes.padStart(2, '0')}`
}

const atTimeOnDate = (date: Date, time: string): Date => {
  const [hours, minutes] = time.split(':').map(Number)
  const result = new Date(date)
  result.setHours(hours, minutes, 0, 0)
  return result
}

export function useBrowseEvents(getRange: () => VisibleRange | null) {
  const store = useBookingStore()
  const cartStore = useCartStore()
  const { availabilityCache, genderFilteredTeachers, confirmedBookings } = storeToRefs(store)
  const { cartItems } = storeToRefs(cartStore)

  const busyByTeacher = computed<Map<number, Span[]>>(() => {
    const map = new Map<number, Span[]>()

    const add = (teacherId: number, startTime: string, endTime: string) => {
      const start = new Date(startTime).getTime()
      const end = new Date(endTime).getTime()
      if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) return
      const list = map.get(teacherId)
      if (list) list.push({ start, end })
      else map.set(teacherId, [{ start, end }])
    }

    for (const booking of confirmedBookings.value) {
      add(booking.teacher_id, booking.start_time, booking.end_time)
    }
    for (const item of cartItems.value) {
      add(item.teacher_id, item.start_time, item.end_time)
    }

    return map
  })

  const browseGroups = computed<BrowseGroup[]>(() => {
    const range = getRange()
    if (!range) return []

    const groups = new Map<string, BrowseGroup>()

    for (const teacher of genderFilteredTeachers.value) {
      const slots = availabilityCache.value.get(teacher.id)
      if (!slots) continue

      const busy = busyByTeacher.value.get(teacher.id) ?? []

      for (const slot of slots) {
        const startTime = normalizeTime(slot.start)
        const endTime = normalizeTime(slot.end)

        const cursor = new Date(range.start)
        cursor.setHours(0, 0, 0, 0)

        while (cursor < range.end) {
          if (cursor.getDay() === slot.day_of_week) {
            const occurrenceStart = atTimeOnDate(cursor, startTime)
            const occurrenceEnd = atTimeOnDate(cursor, endTime)

            if (occurrenceEnd > occurrenceStart) {
              const free = subtractSpans(
                { start: occurrenceStart.getTime(), end: occurrenceEnd.getTime() },
                busy
              ).filter((piece) => piece.end - piece.start >= MIN_REMAINDER_MS)

              for (const piece of free) {
                const key = `${piece.start}-${piece.end}`
                let group = groups.get(key)
                if (!group) {
                  group = {
                    start: new Date(piece.start),
                    end: new Date(piece.end),
                    teachers: [],
                  }
                  groups.set(key, group)
                }
                if (!group.teachers.some((t) => t.id === teacher.id)) {
                  group.teachers.push({ id: teacher.id, name: teacher.name })
                }
              }
            }
          }
          cursor.setDate(cursor.getDate() + 1)
        }
      }
    }

    return [...groups.values()]
  })

  const browseEvents = computed<EventInput[]>(() =>
    browseGroups.value.map((group) => {
      const label =
        group.teachers.length === 1 ? group.teachers[0].name : `${group.teachers.length} teachers`

      return {
        id: `browse-${group.start.getTime()}-${group.end.getTime()}`,
        title: label,
        start: group.start.toISOString(),
        end: group.end.toISOString(),
        editable: false,
        backgroundColor: BROWSE_BACKGROUND,
        borderColor: BROWSE_BORDER,
        textColor: BROWSE_TEXT,
        classNames: ['browse-event'],
        extendedProps: {
          isBrowse: true,
          browseLabel: label,
          teachers: group.teachers,
        },
      }
    })
  )

  return { browseGroups, browseEvents }
}
