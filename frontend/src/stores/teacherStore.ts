import { defineStore } from 'pinia'
import type { Teacher } from '../types'
import { teacherApi } from '../services/teacherApi'
export const useTeacherStore = defineStore('teacher', {
  state: () => ({
    selectedTeacherId: null as number | null,
    teachers: [] as Teacher[],
  }),
  actions: {
    async fetchTeachers() {
      if (this.teachers.length === 0) {
        this.teachers = await teacherApi.getAll()
      }
    },
    async reloadTeachers() {
      this.teachers = await teacherApi.getAll()
    },
    async toggleTeacherStatus(id: number, currentStatus: string) {
      const newStatus = currentStatus === 'active' ? 'deactivated' : 'active'
      await teacherApi.setStatus(id, newStatus)
      const teacher = this.teachers.find((t) => t.id === id)
      if (teacher) teacher.status = newStatus as 'active' | 'deactivated'
    },
    async updateTeacherGender(id: number, gender: string) {
      await teacherApi.setGender(id, gender)
      const teacher = this.teachers.find((t) => t.id === id)
      if (teacher) teacher.gender = gender as 'male' | 'female' | 'lgbtq+'
    },
    async createTeacher(name: string, email: string, gender: string) {
      const created = await teacherApi.create(name, email, gender)
      this.teachers.push(created)
    },
    setSelectedTeacherById(id: number | null) {
      this.selectedTeacherId = id
    },
  },
})
