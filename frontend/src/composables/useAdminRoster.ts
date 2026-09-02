import { ref, computed } from 'vue'
import { bookingApi } from '../services/bookingApi'
import { teacherApi } from '../services/teacherApi'
import { subjectApi } from '../services/subjectApi'
import { userApi } from '../services/userApi'
import type { AuthUser, Booking, Subject, Teacher } from '../types'

export type RosterMode = 'teachers' | 'students'

export interface RosterClass {
  id: number
  startTime: string
  endTime: string
  subject: string
  branch: string
  /** Student in teacher mode, teacher in student mode. */
  counterpart: string
}

export interface RosterEntry {
  id: number
  name: string
  /** Subjects the teacher teaches; empty for students. */
  subjects: string[]
  classes: RosterClass[]
}

/* An unparsable timestamp sorts last rather than throwing, so one malformed
 * booking cannot hide the rest of a person's schedule. */
const timeOf = (iso: string): number => {
  const ms = new Date(iso).getTime()
  return Number.isNaN(ms) ? Number.POSITIVE_INFINITY : ms
}

// The name fields are omitempty server-side, so everything falls back to the id
// rather than rendering a blank that cannot be told apart from another.
const orFallback = (value: string | undefined, label: string, id: number): string =>
  value || `${label} #${id}`

export function useAdminRoster() {
  const bookings = ref<Booking[]>([])
  const teachers = ref<Teacher[]>([])
  const students = ref<AuthUser[]>([])
  const teacherSubjects = ref<Map<number, string[]>>(new Map())

  const isLoading = ref(true)
  const loadError = ref('')

  const mode = ref<RosterMode>('teachers')
  const search = ref('')

  /* There is no endpoint returning a teacher's subjects, so the mapping is
   * rebuilt by asking each subject who teaches it. The requests go out in
   * parallel and the subject list is short, but it is a fan-out — if a
   * teacher-subjects endpoint ever lands, this is the thing to replace. */
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

      bookings.value = bookingList
      teachers.value = teacherList
      students.value = studentList
      teacherSubjects.value = await loadTeacherSubjects(subjectList)
    } catch (err) {
      loadError.value = err instanceof Error ? err.message : 'Failed to load the dashboard'
    } finally {
      isLoading.value = false
    }
  }

  const classesFor = (personBookings: Booking[], mode: RosterMode): RosterClass[] =>
    personBookings
      .slice()
      .sort((a, b) => timeOf(a.start_time) - timeOf(b.start_time))
      .map((booking) => ({
        id: booking.id,
        startTime: booking.start_time,
        endTime: booking.end_time,
        subject: orFallback(booking.subject_name, 'Subject', booking.subject_id),
        branch: orFallback(booking.branch_name, 'Branch', booking.branch_id),
        counterpart:
          mode === 'teachers'
            ? orFallback(booking.student_name, 'Student', booking.student_id)
            : orFallback(booking.teacher_name, 'Teacher', booking.teacher_id),
      }))

  const groupBookings = (key: (booking: Booking) => number): Map<number, Booking[]> => {
    const map = new Map<number, Booking[]>()
    for (const booking of bookings.value) {
      const id = key(booking)
      const existing = map.get(id)
      if (existing) existing.push(booking)
      else map.set(id, [booking])
    }
    return map
  }

  /* Built from the full teacher and student lists rather than from the
   * bookings, so somebody with no classes yet still appears — that absence is
   * usually the thing an admin is looking for. */
  const allEntries = computed<RosterEntry[]>(() => {
    if (mode.value === 'teachers') {
      const byTeacher = groupBookings((b) => b.teacher_id)
      return teachers.value
        .map((teacher) => ({
          id: teacher.id,
          name: teacher.name,
          subjects: teacherSubjects.value.get(teacher.id) ?? [],
          classes: classesFor(byTeacher.get(teacher.id) ?? [], 'teachers'),
        }))
        .sort((a, b) => a.name.localeCompare(b.name))
    }

    const byStudent = groupBookings((b) => b.student_id)
    return students.value
      .map((student) => ({
        id: student.id,
        name: student.name,
        subjects: [],
        classes: classesFor(byStudent.get(student.id) ?? [], 'students'),
      }))
      .sort((a, b) => a.name.localeCompare(b.name))
  })

  // Teachers also match on what they teach, so "Physics" finds everyone who
  // could take a physics booking.
  const entries = computed<RosterEntry[]>(() => {
    const term = search.value.trim().toLowerCase()
    if (!term) return allEntries.value
    return allEntries.value.filter(
      (entry) =>
        entry.name.toLowerCase().includes(term) ||
        entry.subjects.some((subject) => subject.toLowerCase().includes(term))
    )
  })

  const totalClasses = computed(() => bookings.value.length)

  return {
    mode,
    search,
    entries,
    allEntries,
    totalClasses,
    isLoading,
    loadError,
    load,
  }
}
