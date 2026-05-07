export interface CalendarEvent {
  start: Date
  end: Date
  title: string
}

export interface CleanedEvent {
  Teacher: string
  Date: string
  start: string
  end: string
}

export interface Booking {
  id: string
  teacher: string
  startTime: Date
  endTime: Date
  date: Date
}
