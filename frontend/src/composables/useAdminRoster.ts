import { ref, computed } from 'vue'
import { bookingApi } from '../services/bookingApi'
import { teacherApi } from '../services/teacherApi'
import { subjectApi } from '../services/subjectApi'
import { userApi } from '../services/userApi'
import {
  bookingsInWeekOf,
  buildEntry,
  groupBookingsBy,
  matchesSearch,
  sortEntriesForDisplay,
  subjectsFromBookings,
} from '../utils/roster'
import { buildSubjectTones, type SubjectToneMap } from '../utils/subjectColor'
import type { AuthUser, Booking, Subject, Teacher } from '../types'
import type { RosterEntry, RosterMode } from '../utils/roster'

export type {
  RosterEntry,
  RosterMode,
  RosterPerson,
  RosterSession,
  RosterStatus,
} from '../utils/roster'

export interface RosterCounts {
  teachers: number
  students: number
  subjects: number
  classes: number
}

export function useAdminRoster() {
  const bookings = ref<Booking[]>([])
  const teachers = ref<Teacher[]>([])
  const students = ref<AuthUser[]>([])
  const subjects = ref<Subject[]>([])
  const teacherSubjects = ref<Map<number, string[]>>(new Map())

  const isLoading = ref(true)
  const loadError = ref('')

  const mode = ref<RosterMode>('teachers')
  const search = ref('')

  const loadTeacherSubjects = async (list: Subject[]): Promise<Map<number, string[]>> => {
    const map = new Map<number, string[]>()
    const results = await Promise.all(
      list.map(async (subject) => ({
        subject,
        taughtBy: await teacherApi.getBySubject(subject.id),
      }))
    )

    for (const { subject, taughtBy } of results) {
      for (const teacher of taughtBy) {
        const existing = map.get(teacher.id)
        if (existing) existing.push(subject.name)
        else map.set(teacher.id, [subject.name])
      }
    }

    for (const names of map.values()) names.sort((a, b) => a.localeCompare(b))
    return map
  }

  const load = async () => {
    isLoading.value = true
    loadError.value = ''
    try {
      const [bookingList, teacherList, studentList, subjectList] = await Promise.all([
        bookingApi.list(),
        teacherApi.getAll(),
        userApi.listStudents(),
        subjectApi.getAll(),
      ])

      bookings.value = bookingsInWeekOf(bookingList, new Date())
      teachers.value = teacherList
      students.value = studentList
      subjects.value = subjectList
      teacherSubjects.value = await loadTeacherSubjects(subjectList)
    } catch (err) {
      loadError.value = err instanceof Error ? err.message : 'Failed to load the roster'
    } finally {
      isLoading.value = false
    }
  }

  const allEntries = computed<RosterEntry[]>(() => {
    if (mode.value === 'teachers') {
      const byTeacher = groupBookingsBy(bookings.value, (b) => b.teacher_id)
      return sortEntriesForDisplay(
        teachers.value.map((teacher) =>
          buildEntry({
            id: teacher.id,
            name: teacher.name,
            subjects: teacherSubjects.value.get(teacher.id) ?? [],
            bookings: byTeacher.get(teacher.id) ?? [],
            mode: 'teachers',
            status: teacher.status,
          })
        )
      )
    }

    const byStudent = groupBookingsBy(bookings.value, (b) => b.student_id)
    return sortEntriesForDisplay(
      students.value.map((student) => {
        const own = byStudent.get(student.id) ?? []
        return buildEntry({
          id: student.id,
          name: student.name,
          subjects: subjectsFromBookings(own),
          bookings: own,
          mode: 'students',
        })
      })
    )
  })

  const entries = computed<RosterEntry[]>(() =>
    allEntries.value.filter((entry) => matchesSearch(entry, search.value))
  )

  // Built from the canonical subject list rather than from the names that turn
  // up in bookings, so a subject keeps its tone even in a card where it is not
  // booked at all.
  const subjectTones = computed<SubjectToneMap>(() => buildSubjectTones(subjects.value))

  const counts = computed<RosterCounts>(() => ({
    teachers: teachers.value.length,
    students: students.value.length,
    subjects: subjects.value.length,
    classes: bookings.value.length,
  }))

  return {
    mode,
    search,
    entries,
    allEntries,
    counts,
    subjectTones,
    isLoading,
    loadError,
    load,
  }
}
