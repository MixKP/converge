import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'
import { useTeacherStore } from './teacherStore'
import { subjectApi } from '../services/subjectApi'
import { teacherApi } from '../services/teacherApi'
import { branchApi } from '../services/branchApi'
import { debounce } from '../utils/common'
import type { Subject, Teacher, Branch } from '../types'

export const useBookingSelectionStore = defineStore('bookingSelection', () => {
  const teacherStore = useTeacherStore()

  const subjects = ref<Subject[]>([])
  const branches = ref<Branch[]>([])
  const filteredTeachers = ref<Teacher[]>([])
  const selectedSubjectId = ref<number | null>(null)
  const selectedBranchId = ref<number | null>(null)
  const preferredTeacherId = ref<number | null>(null)
  const isLoadingTeachers = ref(false)

  const selectedTeacherId = computed({
    get: () => teacherStore.selectedTeacherId,
    set: (val) => teacherStore.setSelectedTeacherById(val),
  })

  const fetchSubjects = async () => {
    const data = await subjectApi.getAll()
    subjects.value = data
  }

  const fetchBranches = async () => {
    const data = await branchApi.getAll()
    branches.value = data
  }

  const fetchTeachersBySubject = async (subjectId: number): Promise<void> => {
    try {
      isLoadingTeachers.value = true
      filteredTeachers.value = await teacherApi.getBySubject(subjectId)
    } finally {
      isLoadingTeachers.value = false
    }
  }

  let currentSubjectChangePromise: Promise<void> | null = null

  const handleSubjectChange = debounce(async (newSubjectId: number | null) => {
    currentSubjectChangePromise = (async () => {
      if (newSubjectId) {
        await fetchTeachersBySubject(newSubjectId)
      } else {
        filteredTeachers.value = []
      }
      selectedBranchId.value = null
      teacherStore.setSelectedTeacherById(null)
      preferredTeacherId.value = null
    })()

    await currentSubjectChangePromise
  }, 200)

  watch(selectedSubjectId, (newSubjectId) => {
    handleSubjectChange(newSubjectId)
  })

  return {
    // Selection State
    subjects,
    branches,
    filteredTeachers,
    selectedSubjectId,
    selectedBranchId,
    selectedTeacherId,
    preferredTeacherId,
    isLoadingTeachers,

    // Actions
    fetchSubjects,
    fetchBranches,
    fetchTeachersBySubject,
    handleSubjectChange,
  }
})
