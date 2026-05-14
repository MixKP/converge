import { defineStore } from 'pinia'
import type { Booking } from '../types/calendar'
import type { Teacher } from '../types/teacher'
import { teacherApi } from '../services/teacherApi'

export const useTeacherStore = defineStore('teacher', {
  state: () => ({
    selectedTeacher: '' as string,
    bookings: [] as Booking[],
    teachers: [] as Teacher[],
    loading: false as boolean,
    error: null as string | null,
  }),

  getters: {
    selectedTeacherBookings: (state) => {
      if (!state.selectedTeacher) return []
      return state.bookings.filter((b) => b.teacher === state.selectedTeacher)
    },
    teacherNames: (state) => state.teachers.map((t) => t.name),
  },

  actions: {
    async fetchTeachers() {
      this.teachers = await teacherApi.getAll()
    },

    setSelectedTeacher(teacher: string) {
      this.selectedTeacher = teacher
    },

    addBooking(booking: Omit<Booking, 'id'>) {
      const newStartTime = booking.startTime.getTime()
      const newEndTime = booking.endTime.getTime()

      const hasConflict = this.bookings.some(
        (existingBooking) =>
          existingBooking.teacher === booking.teacher &&
          newStartTime < existingBooking.endTime.getTime() &&
          newEndTime > existingBooking.startTime.getTime()
      )

      if (hasConflict) {
        return { success: false, message: 'This Time Slot has Overlaps' }
      }

      const newBooking: Booking = {
        ...booking,
        id: crypto.randomUUID(),
      }
      this.bookings.push(newBooking)
      return { success: true, data: newBooking }
    },
  },
})
