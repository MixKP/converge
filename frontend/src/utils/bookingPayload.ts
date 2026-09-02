import type { CartItem, ConfirmBookingRequest } from '../types'

export function toConfirmRequest(item: CartItem): ConfirmBookingRequest | null {
  if (!Number.isInteger(item.student_id) || item.student_id <= 0) return null

  return {
    teacher_id: item.teacher_id,
    branch_id: item.branch_id,
    subject_id: item.subject_id,
    start_time: item.start_time,
    end_time: item.end_time,
    student_id: item.student_id,
  }
}
