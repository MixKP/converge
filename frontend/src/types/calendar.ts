export interface CalendarEvent {
  start: Date
  end: Date
  title: string
}

export interface WeeklySlot {
  day_of_week: number
  start: string
  end: string
}

export interface AvailabilityPayload {
  teacher_id: number
  weekly: WeeklySlot[]
}
