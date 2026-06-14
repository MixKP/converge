// Date validation and formatting utilities

/**
 * Check if a date is valid
 * A date is valid if it's a Date instance and getTime() returns a number (not NaN)
 *
 * @param date - The date to validate
 * @returns True if the date is valid
 *
 * @example
 * ```ts
 * isValidDate(new Date()) // true
 * isValidDate(new Date('invalid')) // false
 * isValidDate(null) // false
 * ```
 */
export function isValidDate(date: Date): boolean {
  return date instanceof Date && !isNaN(date.getTime())
}

/**
 * Validate a date range
 * Checks that both start and end dates are valid and start is before end
 *
 * @param start - The start date
 * @param end - The end date
 * @returns Object with validation result and optional error message
 *
 * @example
 * ```ts
 * const result = validateDateRange(startDate, endDate)
 * if (!result.isValid) {
 *   alert(result.error)
 * }
 * ```
 */
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

/**
 * Check if two time ranges overlap
 * Two ranges overlap if they share any common time period
 *
 * @param start1 - Start of first range
 * @param end1 - End of first range
 * @param start2 - Start of second range
 * @param end2 - End of second range
 * @returns True if the ranges overlap
 *
 * @example
 * ```ts
 * const hasConflict = rangesOverlap(
 *   new Date('2024-01-01T10:00:00'),
 *   new Date('2024-01-01T11:00:00'),
 *   new Date('2024-01-01T10:30:00'),
 *   new Date('2024-01-01T12:00:00')
 * ) // true
 * ```
 */
export function rangesOverlap(start1: Date, end1: Date, start2: Date, end2: Date): boolean {
  if (!isValidDate(start1) || !isValidDate(end1) || !isValidDate(start2) || !isValidDate(end2)) {
    return false
  }
  return start1 < end2 && end1 > start2
}

/**
 * Check if a time slot overlaps with any cart item for the same teacher
 *
 * @param startTime - Start time to check (ISO string or Date constructor compatible)
 * @param endTime - End time to check
 * @param teacherId - Teacher ID to match against
 * @param cartItems - Array of cart items
 * @returns True if there's an overlap with an existing cart item for the same teacher
 *
 * @example
 * ```ts
 * const hasOverlap = hasOverlapWithCart(
 *   '2024-01-01T10:00:00',
 *   '2024-01-01T11:00:00',
 *   teacherId,
 *   cartItems.value
 * )
 * if (hasOverlap) {
 *   alert('This time slot is already in your cart for this teacher')
 * }
 * ```
 */
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
    // Only check items for the same teacher
    if (item.teacher_id !== teacherId) return false

    const itemStart = new Date(item.start_time)
    const itemEnd = new Date(item.end_time)

    // Skip invalid cart item dates
    if (!isValidDate(itemStart) || !isValidDate(itemEnd)) return false

    return rangesOverlap(newStart, newEnd, itemStart, itemEnd)
  })
}

/**
 * Format a time string (HH:MM) from a Date
 *
 * @param date - The date to format
 * @returns Formatted time string in 24-hour format
 *
 * @example
 * ```ts
 * formatTimeString(new Date('2024-01-01T09:30:00')) // '09:30'
 * formatTimeString(new Date('2024-01-01T14:05:00')) // '14:05'
 * ```
 */
export function formatTimeString(date: Date): string {
  if (!isValidDate(date)) return '00:00'
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  return `${hours}:${minutes}`
}

/**
 * Format day name from a Date
 *
 * @param date - The date to format
 * @param format - Format type: 'long', 'short', or 'narrow'
 * @returns Formatted day name
 *
 * @example
 * ```ts
 * formatDayName(date, 'long')    // 'Sunday'
 * formatDayName(date, 'short')   // 'Sun'
 * formatDayName(date, 'narrow') // 'S'
 * ```
 */
export function formatDayName(date: Date, format: 'long' | 'short' | 'narrow' = 'long'): string {
  if (!isValidDate(date)) return ''
  return date.toLocaleDateString('en-US', { weekday: format })
}

/**
 * Format time in 12-hour format with AM/PM
 *
 * @param date - The date to format
 * @returns Formatted time string (e.g., "9:30am", "12:15pm")
 *
 * @example
 * ```ts
 * formatTime12Hour(new Date('2024-01-01T09:30:00')) // '9:30am'
 * formatTime12Hour(new Date('2024-01-01T13:00:00')) // '1pm'
 * formatTime12Hour(new Date('2024-01-01T13:15:00')) // '1:15pm'
 * ```
 */
export function formatTime12Hour(date: Date): string {
  if (!isValidDate(date)) return ''

  const hours = date.getHours()
  const minutes = date.getMinutes()
  const ampm = hours >= 12 ? 'pm' : 'am'
  const displayHours = hours === 0 ? 12 : hours > 12 ? hours - 12 : hours
  const displayMinutes = minutes > 0 ? `:${String(minutes).padStart(2, '0')}` : ''

  return `${displayHours}${displayMinutes}${ampm}`
}

/**
 * Get the next occurrence of a specific day of week and time
 * Returns a date for the next occurrence of the given day/time
 *
 * @param dayOfWeek - Day of week (0=Sunday, 1=Monday, ..., 6=Saturday)
 * @param timeStr - Time string in HH:MM format
 * @returns Date object for the next occurrence
 *
 * @example
 * ```ts
 * // If today is Wednesday
 * getNextDateForDayOfWeek(5, '10:00') // Next Friday at 10am
 * getNextDateForDayOfWeek(1, '14:30') // Next Monday at 2:30pm
 * ```
 */
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

  // If the time has passed today, move to next week
  if (targetDate <= now) {
    targetDate.setDate(targetDate.getDate() + 7)
  }

  return targetDate
}

/**
 * Check if a date is at least one week in the future
 *
 * @param date - The date to check
 * @returns True if the date is at least one week away
 *
 * @example
 * ```ts
 * isNextWeek(new Date(Date.now() + 8 * 24 * 60 * 60 * 1000)) // true
 * isNextWeek(new Date()) // false
 * ```
 */
export function isNextWeek(date: Date): boolean {
  if (!isValidDate(date)) return false
  const now = new Date()
  const oneWeekFromNow = new Date(now)
  oneWeekFromNow.setDate(now.getDate() + 7)
  return date >= oneWeekFromNow
}

/**
 * Format a date range for display
 * Combines day name and time range into a readable string
 *
 * @param start - The start date
 * @param end - The end date
 * @returns Formatted date range string
 *
 * @example
 * ```ts
 * formatSuggestionDate(startDate, endDate)
 * // 'Sun 9:30am - 10:30am'
 * ```
 */
export function formatSuggestionDate(start: Date, end: Date): string {
  const dayName = formatDayName(start, 'short')
  return `${dayName} ${formatTime12Hour(start)} - ${formatTime12Hour(end)}`
}

/**
 * Convert time string (HH:MM) to minutes
 * Useful for calculating time differences and durations
 *
 * @param timeStr - Time string in HH:MM format
 * @returns Time in minutes since midnight
 *
 * @example
 * ```ts
 * toMinutes('09:30')  // 570
 * toMinutes('14:00')  // 840
 * toMinutes('00:00')  // 0
 * ```
 */
export function toMinutes(timeStr: string): number {
  const parts = timeStr.split(':')
  if (parts.length !== 2) return NaN

  const [hourStr, minuteStr] = parts
  const hour = Number(hourStr)
  const minute = Number(minuteStr)

  // Validate numeric values and reasonable ranges
  if (isNaN(hour) || isNaN(minute)) return NaN
  if (hour < 0 || hour > 23 || minute < 0 || minute > 59) return NaN

  return hour * 60 + minute
}

/**
 * Convert minutes to time string (HH:MM)
 * Useful for displaying calculated times
 *
 * @param minutes - Minutes since midnight
 * @returns Time string in HH:MM format
 *
 * @example
 * ```ts
 * toTimeString(570)   // '09:30'
 * toTimeString(840)   // '14:00'
 * toTimeString(0)     // '00:00'
 * ```
 */
export function toTimeString(minutes: number): string {
  if (minutes < 0 || !Number.isFinite(minutes)) {
    throw new Error(`Invalid minutes value: ${minutes}. Must be a non-negative finite number.`)
  }
  const hours = Math.floor(minutes / 60) % 24
  const mins = minutes % 60
  return `${String(hours).padStart(2, '0')}:${String(mins).padStart(2, '0')}`
}
