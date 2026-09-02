export function isValidDate(date: Date): boolean {
  return date instanceof Date && !isNaN(date.getTime())
}

export function rangesOverlap(start1: Date, end1: Date, start2: Date, end2: Date): boolean {
  if (!isValidDate(start1) || !isValidDate(end1) || !isValidDate(start2) || !isValidDate(end2)) {
    return false
  }
  return start1 < end2 && end1 > start2
}

export function hasOverlapWithCart(
  startTime: string,
  endTime: string,
  teacherId: number,
  cartItems: Array<{
    id: number
    teacher_id: number
    start_time: string
    end_time: string
  }>
): boolean {
  const newStart = new Date(startTime)
  const newEnd = new Date(endTime)

  if (!isValidDate(newStart) || !isValidDate(newEnd)) {
    return false
  }

  return cartItems.some((item) => {
    if (item.teacher_id !== teacherId) return false

    const itemStart = new Date(item.start_time)
    const itemEnd = new Date(item.end_time)

    if (!isValidDate(itemStart) || !isValidDate(itemEnd)) return false

    return rangesOverlap(newStart, newEnd, itemStart, itemEnd)
  })
}

export function toMinutes(timeStr: string): number {
  const parts = timeStr.split(':')
  if (parts.length !== 2) return NaN

  const [hourStr, minuteStr] = parts
  const hour = Number(hourStr)
  const minute = Number(minuteStr)

  if (isNaN(hour) || isNaN(minute)) return NaN
  if (hour < 0 || hour > 23 || minute < 0 || minute > 59) return NaN

  return hour * 60 + minute
}
