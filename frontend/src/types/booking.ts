import type { WeeklySlot } from './calendar'

export interface CartItem {
  id: number
  teacher_id: number
  teacher_name: string
  branch_id: number
  branch_name: string
  subject_id: number
  subject_name: string
  start_time: string
  end_time: string
  client_name: string
  status: 'pending' | 'confirmed'
}

export interface BookingRequest {
  subject_id: number
  branch_id: number
  preferred_slots: WeeklySlot[]
  duration_minutes?: number
  preferred_teacher_id?: number
}

export interface BookingAlternative {
  teacher_id: number
  teacher_name: string
  branch_id: number
  subject_id: number
  start_time: string
  end_time: string
  score: number
  reasons: string[]
  room_available?: boolean
  commute_minutes?: number
}

export interface SlotResult {
  slot: WeeklySlot
  exact_match?: BookingAlternative
  alternatives?: BookingAlternative[]
  message: string
}

export interface BookingResponse {
  results: SlotResult[]
}

export interface ConfirmBookingRequest {
  teacher_id: number
  branch_id: number
  subject_id: number
  start_time: string
  end_time: string
  client_name: string
}

export interface Booking {
  id: number
  teacher_id: number
  branch_id: number
  subject_id: number
  start_time: string
  end_time: string
  client_name: string
  created_at: string
}

export interface Branch {
  id: number
  name: string
}
