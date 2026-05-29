import { defineStore } from 'pinia'
import type { Teacher } from '../types'
import { teacherApi } from '../services/teacherApi'

export const useTeacherStore = defineStore('teacher', {
  state: () => ({
    selectedTeacherId: null as number | null,
    teachers: [] as Teacher[],
    teachersLoaded: false,
  }),

  actions: {
    async fetchTeachers() {
      if (!this.teachersLoaded) {
        this.teachers = await teacherApi.getAll()
        this.teachersLoaded = true
      }
    },

    setSelectedTeacherById(id: number | null) {
      this.selectedTeacherId = id
    },
  },
})
