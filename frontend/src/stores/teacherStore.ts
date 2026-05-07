import { defineStore } from 'pinia'
import type { Booking } from '../types/calendar'

export const useTeacherStore = defineStore('teacher', {
  state: () => ({
    selectedTeacher: '' as string,
    bookings: [] as Booking[],
    teachers: ['Teacher A', 'Teacher B', 'Teacher C', 'Teacher D'] as string[],
  }),

  actions: {
    setSelectedTeacher(teacher: string) {
      this.selectedTeacher = teacher
    },

    addBooking(booking: Omit<Booking, 'id'>) {
      const newBooking: Booking = {
        ...booking,
        id: crypto.randomUUID(),
      }
      this.bookings.push(newBooking)
      return newBooking
    },

    removeBooking(id: string) {
      const index = this.bookings.findIndex((b) => b.id === id)
      if (index > -1) this.bookings.splice(index, 1)
    },

    clearBookings() {
      this.bookings = []
    },

    getBookingsByTeacher(teacherName: string) {
      return this.bookings.filter((b) => b.teacher === teacherName)
    },
  },
})
