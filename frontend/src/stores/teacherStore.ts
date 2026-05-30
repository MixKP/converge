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
    setSelectedTeacherById(id: number | null) {
      this.selectedTeacherId = id
    },
  },
})
