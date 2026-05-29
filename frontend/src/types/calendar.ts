export interface WeeklySlot {
  dayOfWeek: number
  start: string
  end: string
}

export interface AvailabilityPayload {
  teacherId: number
  weekly: WeeklySlot[]
}
