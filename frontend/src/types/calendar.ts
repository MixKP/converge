export interface CalendarEvent {
  start: Date
  end: Date
  title: string
}

export interface CleanedEvent {
  teacher: string
  date: string
  start: string
  end: string
}

export interface Booking {
  id: string
  teacher: string
  startTime: Date
  endTime: Date
}

export interface EventCreateParams {
  event: CalendarEvent
  resolve: (e?: CalendarEvent) => void
}
