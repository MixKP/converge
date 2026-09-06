export interface CapacityBooking {
  branch_id: number
  start_time: string
  end_time: string
}

export interface CapacitySpan {
  start: number
  end: number
  occupancy: number
}

export const fullCapacitySpans = (
  branchId: number,
  capacity: number,
  bookings: CapacityBooking[]
): CapacitySpan[] => {
  if (capacity < 1) return []

  const changes = new Map<number, number>()
  for (const booking of bookings) {
    if (booking.branch_id !== branchId) continue
    const start = new Date(booking.start_time).getTime()
    const end = new Date(booking.end_time).getTime()
    if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) continue
    changes.set(start, (changes.get(start) ?? 0) + 1)
    changes.set(end, (changes.get(end) ?? 0) - 1)
  }

  const times = [...changes.keys()].sort((a, b) => a - b)
  const spans: CapacitySpan[] = []
  let occupancy = 0
  for (let i = 0; i < times.length - 1; i++) {
    const start = times[i]
    occupancy += changes.get(start) ?? 0
    const end = times[i + 1]
    if (occupancy >= capacity && end > start) spans.push({ start, end, occupancy })
  }
  return spans
}

export const overlapsFullCapacity = (
  spans: CapacitySpan[],
  startTime: string,
  endTime: string
): boolean => {
  const start = new Date(startTime).getTime()
  const end = new Date(endTime).getTime()
  if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) return false
  return spans.some((span) => start < span.end && end > span.start)
}
