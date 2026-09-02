import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useBookingStore } from '../stores/bookingStore'
import { useCartStore } from '../stores/cartStore'
import { useCommute } from './useCommute'
import { mergeSpans, subtractSpans, type Span } from '../utils/intervals'
import type { EventInput } from '@fullcalendar/core'

const COMMUTE_BACKGROUND = 'rgba(192, 87, 74, 0.14)'
const COMMUTE_BORDER = 'var(--destructive)'
const COMMUTE_TEXT = 'var(--destructive)'

const MIN_VISIBLE_MS = 10 * 60 * 1000

interface Engagement {
  start: number
  end: number
  branchId: number
}

export function useCommuteBlocks() {
  const store = useBookingStore()
  const cartStore = useCartStore()
  const { confirmedBookings, selectedTeacherId, selectedBranchId } = storeToRefs(store)
  const { cartItems } = storeToRefs(cartStore)
  const { commuteMinutes } = useCommute()

  const engagements = computed<Engagement[]>(() => {
    const teacherId = selectedTeacherId.value
    if (teacherId === null) return []

    const rows: Engagement[] = []
    const add = (branchId: number, startTime: string, endTime: string) => {
      const start = new Date(startTime).getTime()
      const end = new Date(endTime).getTime()
      if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) return
      rows.push({ start, end, branchId })
    }

    for (const booking of confirmedBookings.value) {
      if (booking.teacher_id === teacherId) {
        add(booking.branch_id, booking.start_time, booking.end_time)
      }
    }
    for (const item of cartItems.value) {
      if (item.teacher_id === teacherId) {
        add(item.branch_id, item.start_time, item.end_time)
      }
    }
    return rows
  })

  const commuteSpans = computed<Span[]>(() => {
    const minutes = commuteMinutes.value
    const branchId = selectedBranchId.value
    if (!minutes || branchId === null) return []

    const buffer = minutes * 60 * 1000
    const spans: Span[] = []
    for (const engagement of engagements.value) {
      if (engagement.branchId === branchId) continue
      spans.push({ start: engagement.start - buffer, end: engagement.start })
      spans.push({ start: engagement.end, end: engagement.end + buffer })
    }

    const booked = engagements.value.map((e) => ({ start: e.start, end: e.end }))
    return mergeSpans(spans)
      .flatMap((span) => subtractSpans(span, booked))
      .filter((span) => span.end - span.start >= MIN_VISIBLE_MS)
  })

  const commuteEvents = computed<EventInput[]>(() =>
    commuteSpans.value.map((span) => ({
      id: `commute-${span.start}-${span.end}`,
      title: `Commute ${commuteMinutes.value} min`,
      start: new Date(span.start).toISOString(),
      end: new Date(span.end).toISOString(),
      editable: false,
      backgroundColor: COMMUTE_BACKGROUND,
      borderColor: COMMUTE_BORDER,
      textColor: COMMUTE_TEXT,
      classNames: ['commute-event'],
      extendedProps: {
        isCommute: true,
        commuteLabel: `Commute ${commuteMinutes.value} min`,
      },
    }))
  )

  const overlapsCommute = (start: Date, end: Date): boolean => {
    const from = start.getTime()
    const to = end.getTime()
    return commuteSpans.value.some((span) => from < span.end && to > span.start)
  }

  return { commuteEvents, commuteSpans, overlapsCommute }
}
