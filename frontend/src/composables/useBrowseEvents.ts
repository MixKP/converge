import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useBookingStore } from '../stores/bookingStore'
import type { EventInput } from '@fullcalendar/core'

export interface BrowseTeacher {
  id: number
  name: string
}

export interface BrowseGroup {
  day_of_week: number
  start: string
  end: string
  teachers: BrowseTeacher[]
}

/* Browse mode needs no backend call: the store already pulls every teacher's
 * weekly availability into `availabilityCache` on initialize(). The store then
 * flattens it into `businessHours`, which loses the teacher identity — this
 * rebuilds that identity as clickable events instead. */

// One tone for every block; the teacher's name is the label. Per-teacher
// colours would fight the Soft Sage palette, so identity is carried by text.
const BROWSE_BACKGROUND = 'rgba(45, 74, 62, 0.10)'
const BROWSE_BORDER = 'var(--primary-indigo)'
const BROWSE_TEXT = 'var(--primary-indigo)'

/* The backend is not guaranteed to pad hours or omit seconds, and "9:00" vs
 * "09:00" for the same window would otherwise key as two groups and render as
 * two stacked blocks for what is one slot. */
const normalizeTime = (value: string): string => {
  const [hours = '', minutes = '00'] = value.split(':')
  return `${hours.padStart(2, '0')}:${minutes.padStart(2, '0')}`
}

export function useBrowseEvents() {
  const store = useBookingStore()
  const { availabilityCache, genderFilteredTeachers } = storeToRefs(store)

  /* Teachers frequently share an identical weekly window, and rendering one
   * block each turns a busy Tuesday into an unreadable stack. Identical ranges
   * collapse into a single block that names how many teachers it covers. */
  const browseGroups = computed<BrowseGroup[]>(() => {
    const groups = new Map<string, BrowseGroup>()

    for (const teacher of genderFilteredTeachers.value) {
      const slots = availabilityCache.value.get(teacher.id)
      if (!slots) continue

      for (const slot of slots) {
        const start = normalizeTime(slot.start)
        const end = normalizeTime(slot.end)
        const key = `${slot.day_of_week}-${start}-${end}`
        let group = groups.get(key)
        if (!group) {
          group = {
            day_of_week: slot.day_of_week,
            start,
            end,
            teachers: [],
          }
          groups.set(key, group)
        }
        if (!group.teachers.some((t) => t.id === teacher.id)) {
          group.teachers.push({ id: teacher.id, name: teacher.name })
        }
      }
    }

    return [...groups.values()]
  })

  const browseEvents = computed<EventInput[]>(() =>
    browseGroups.value.map((group) => {
      const label =
        group.teachers.length === 1 ? group.teachers[0].name : `${group.teachers.length} teachers`

      /* Weekly availability is recurring, so it is declared as a recurrence
       * rather than resolved to a date. Resolving was actively wrong:
       * getNextDateForDayOfWeek pushes a past time forward a week, and it is
       * called separately for start and end — so a slot already underway today
       * got a start next week and an end today. Recurrence also means the
       * blocks follow the user when they page to another week. */
      return {
        id: `browse-${group.day_of_week}-${group.start}-${group.end}`,
        title: label,
        daysOfWeek: [group.day_of_week],
        startTime: group.start,
        endTime: group.end,
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
