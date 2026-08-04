import type { CartItem, ConfirmBookingRequest } from '../types'

/**
 * Maps a cart item onto the exact field set `POST /bookings/confirm` accepts.
 * Kept pure and framework-free so the wire shape can be exercised directly.
 *
 * Returns null for an item with no student attached. The cart is persisted in
 * localStorage, so an item written by a build that predates student ownership
 * can outlive the upgrade; confirming it would 400 on `student_id`.
 */
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
