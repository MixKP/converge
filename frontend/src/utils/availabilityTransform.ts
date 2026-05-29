import type { WeeklySlot } from '../types'
interface BackendWeeklySlot {
  day_of_week: number
  start: string
  end: string
}
interface BackendTeacherAvailability {
  teacher: {
    id: number
    name: string
    email: string
  }
  weekly: BackendWeeklySlot[]
}
export type { BackendTeacherAvailability }
export function transformBackendSlot(slot: BackendWeeklySlot): WeeklySlot {
  return {
    day_of_week: slot.day_of_week,
    start: slot.start,
    end: slot.end,
  }
}
export function transformBackendAvailability(
  data: BackendTeacherAvailability[]
): Map<number, WeeklySlot[]> {
  const map = new Map<number, WeeklySlot[]>()
  for (const item of data) {
    map.set(item.teacher.id, item.weekly.map(transformBackendSlot))
  }
  return map
}
