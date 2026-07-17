export function isValidDate(date: Date): boolean {
  return date instanceof Date && !isNaN(date.getTime())
}

export function validateDateRange(
  start: Date,
  end: Date
): {
  isValid: boolean
  error?: string
} {
  if (!isValidDate(start) || !isValidDate(end)) {
    return { isValid: false, error: 'Invalid date format' }
  }
  if (start >= end) {
    return { isValid: false, error: 'Start time must be before end time' }
  }
  return { isValid: true }
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

export function formatTimeString(date: Date): string {
  if (!isValidDate(date)) return '00:00'
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  return `${hours}:${minutes}`
}

export function formatDayName(date: Date, format: 'long' | 'short' | 'narrow' = 'long'): string {
  if (!isValidDate(date)) return ''
  return date.toLocaleDateString('en-US', { weekday: format })
}

export function formatTime12Hour(date: Date): string {
  if (!isValidDate(date)) return ''

  const hours = date.getHours()
  const minutes = date.getMinutes()
  const ampm = hours >= 12 ? 'pm' : 'am'
  const displayHours = hours === 0 ? 12 : hours > 12 ? hours - 12 : hours
  const displayMinutes = minutes > 0 ? `:${String(minutes).padStart(2, '0')}` : ''

  return `${displayHours}${displayMinutes}${ampm}`
}

export function getNextDateForDayOfWeek(dayOfWeek: number, timeStr: string): Date {
  const now = new Date()
  const currentDay = now.getDay()
  const diff = (dayOfWeek - currentDay + 7) % 7
  const targetDate = new Date(now)
  targetDate.setDate(now.getDate() + diff)

  const totalMinutes = toMinutes(timeStr)
  if (isNaN(totalMinutes)) {
    throw new Error(`Invalid time format: ${timeStr}. Expected HH:MM format.`)
  }
  const hours = Math.floor(totalMinutes / 60)
  const minutes = totalMinutes % 60
  targetDate.setHours(hours, minutes, 0, 0)

  if (targetDate <= now) {
    targetDate.setDate(targetDate.getDate() + 7)
  }

  return targetDate
}

export function isNextWeek(date: Date): boolean {
  if (!isValidDate(date)) return false
  const now = new Date()
  const oneWeekFromNow = new Date(now)
  oneWeekFromNow.setDate(now.getDate() + 7)
  return date >= oneWeekFromNow
}

export function formatSuggestionDate(start: Date, end: Date): string {
  const dayName = formatDayName(start, 'short')
  return `${dayName} ${formatTime12Hour(start)} - ${formatTime12Hour(end)}`
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

export function toTimeString(minutes: number): string {
  if (minutes < 0 || !Number.isFinite(minutes)) {
    throw new Error(`Invalid minutes value: ${minutes}. Must be a non-negative finite number.`)
  }
  const hours = Math.floor(minutes / 60) % 24
  const mins = minutes % 60
  return `${String(hours).padStart(2, '0')}:${String(mins).padStart(2, '0')}`
}
