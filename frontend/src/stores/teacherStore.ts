import { defineStore } from 'pinia'
import type { Teacher } from '../types/teacher'
import { teacherApi } from '../services/teacherApi'

export const useTeacherStore = defineStore('teacher', {
  state: () => ({
    selectedTeacherId: null as number | null,
    teachers: [] as Teacher[],
  }),

  getters: {
    selectedTeacherName: (state) => {
      const teacher = state.teachers.find((t) => t.id === state.selectedTeacherId)
      return teacher?.name || ''
    },
  },

  actions: {
    async fetchTeachers() {
      this.teachers = await teacherApi.getAll()
    },

    setSelectedTeacherById(id: number | null) {
      this.selectedTeacherId = id
    },
  },
})
