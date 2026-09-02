<script setup lang="ts">
  import { ref, computed, watch, nextTick } from 'vue'
  import { Loader2, Pencil, UserPlus } from '@lucide/vue'
  import { useBooking } from '../../composables/useBooking'
  import { useBookingContext } from '../../composables/useBookingContext'
  import { useNotification } from '../../composables/useNotification'
  import { useNumberSelect, NONE } from '../../composables/useSelectProxy'
  import { authApi } from '../../services/authApi'
  import { registerRequestSchema } from '../../schemas/auth'
  import { Badge } from '@/components/ui/badge'
  import { Button } from '@/components/ui/button'
  import { Card, CardContent } from '@/components/ui/card'
  import { Label } from '@/components/ui/label'
  import {
    Select,
    SelectContent,
    SelectItem,
    SelectSeparator,
    SelectTrigger,
    SelectValue,
  } from '@/components/ui/select'

  const {
    subjects,
    branches,
    students,
    selectedStudentId,
    selectedSubjectId,
    selectedBranchId,
    fetchStudents,
  } = useBooking()

  const { contextComplete } = useBookingContext()
  const { showSuccess } = useNotification()

  const subjectValue = useNumberSelect(selectedSubjectId)
  const branchValue = useNumberSelect(selectedBranchId)

  const CREATE_STUDENT = '__create_student'

  const isCreatingStudent = ref(false)
  const newStudentName = ref('')
  const newStudentPassword = ref('')
  const createError = ref('')
  const isSubmitting = ref(false)
  const nameInput = ref<HTMLInputElement | null>(null)

  const studentValue = computed<string>({
    get: () => (selectedStudentId.value === null ? NONE : String(selectedStudentId.value)),
    set: (val) => {
      if (val === CREATE_STUDENT) {
        openCreateStudent()
        return
      }
      selectedStudentId.value = val === NONE ? null : Number(val)
    },
  })

  function openCreateStudent() {
    createError.value = ''
    newStudentName.value = ''
    newStudentPassword.value = ''
    isCreatingStudent.value = true
    nextTick(() => nameInput.value?.focus())
  }

  const cancelCreateStudent = () => {
    if (isSubmitting.value) return
    isCreatingStudent.value = false
    createError.value = ''
  }

  const createStudent = async () => {
    createError.value = ''
    const parsed = registerRequestSchema.safeParse({
      name: newStudentName.value,
      password: newStudentPassword.value,
      role: 'student',
    })
    if (!parsed.success) {
      createError.value = parsed.error.issues[0]?.message ?? 'Please check the form'
      return
    }

    isSubmitting.value = true
    try {
      const created = await authApi.register(parsed.data)
      await fetchStudents()
      selectedStudentId.value = created.id
      isCreatingStudent.value = false
      showSuccess(`Student "${created.name}" created and selected`, 4000)
    } catch (err) {
      createError.value = err instanceof Error ? err.message : 'Failed to create the student'
    } finally {
      isSubmitting.value = false
    }
  }

  const isEditing = ref(false)

  watch(contextComplete, (complete) => {
    if (complete && !isCreatingStudent.value) isEditing.value = false
  })

  const expanded = computed(() => isEditing.value || !contextComplete.value)

  const chips = computed(() => [
    { label: 'Student', value: students.value.find((s) => s.id === selectedStudentId.value)?.name },
    { label: 'Subject', value: subjects.value.find((s) => s.id === selectedSubjectId.value)?.name },
    { label: 'Branch', value: branches.value.find((b) => b.id === selectedBranchId.value)?.name },
  ])
</script>

<template>
  <Card>
    <CardContent class="py-4">
      <div v-if="expanded" class="flex flex-col gap-4">
        <div class="grid gap-4 sm:grid-cols-3">
          <div class="flex flex-col gap-2">
            <Label for="v3-student">Student</Label>
            <Select v-model="studentValue">
              <SelectTrigger id="v3-student" class="w-full">
                <SelectValue placeholder="Select a student" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem
                  v-for="student in students"
                  :key="student.id"
                  :value="String(student.id)"
                >
                  {{ student.name }}
                </SelectItem>
                <SelectSeparator v-if="students.length" />
                <SelectItem :value="CREATE_STUDENT">
                  <span class="flex items-center gap-2">
                    <UserPlus class="size-4" />
                    New student…
                  </span>
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div class="flex flex-col gap-2">
            <Label for="v3-subject">Subject</Label>
            <Select v-model="subjectValue">
              <SelectTrigger id="v3-subject" class="w-full">
                <SelectValue placeholder="Select a subject" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem
                  v-for="subject in subjects"
                  :key="subject.id"
                  :value="String(subject.id)"
                >
                  {{ subject.name }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div class="flex flex-col gap-2">
            <Label for="v3-branch">Branch</Label>
            <Select v-model="branchValue" :disabled="selectedSubjectId === null">
              <SelectTrigger id="v3-branch" class="w-full">
                <SelectValue
                  :placeholder="
                    selectedSubjectId === null ? 'Pick a subject first' : 'Select a branch'
                  "
                />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="branch in branches" :key="branch.id" :value="String(branch.id)">
                  {{ branch.name }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
        <form
          v-if="isCreatingStudent"
          class="border-border flex flex-col gap-3 rounded-lg border p-3"
          @submit.prevent="createStudent"
        >
          <p class="text-sm font-medium">New student</p>
          <div class="grid gap-3 sm:grid-cols-2">
            <div class="flex flex-col gap-2">
              <Label for="v3-new-student-name">Username</Label>
              <input
                id="v3-new-student-name"
                ref="nameInput"
                v-model="newStudentName"
                type="text"
                autocomplete="off"
                class="border-input bg-background focus-visible:border-ring focus-visible:ring-ring/50 h-9 w-full rounded-md border px-3 py-1 text-sm shadow-xs outline-none focus-visible:ring-3"
              />
            </div>
            <div class="flex flex-col gap-2">
              <Label for="v3-new-student-password">Password</Label>
              <input
                id="v3-new-student-password"
                v-model="newStudentPassword"
                type="password"
                autocomplete="new-password"
                class="border-input bg-background focus-visible:border-ring focus-visible:ring-ring/50 h-9 w-full rounded-md border px-3 py-1 text-sm shadow-xs outline-none focus-visible:ring-3"
              />
            </div>
          </div>

          <p class="text-muted-foreground text-xs">
            This creates a login account. Pass the password on to the student — there is no reset
            flow.
          </p>

          <p v-if="createError" class="text-destructive text-sm" role="alert">{{ createError }}</p>

          <div class="flex items-center justify-end gap-2">
            <Button type="button" variant="ghost" size="sm" @click="cancelCreateStudent">
              Cancel
            </Button>
            <Button type="submit" size="sm" :disabled="isSubmitting">
              <Loader2 v-if="isSubmitting" class="size-4 animate-spin" />
              Create student
            </Button>
          </div>
        </form>

        <div v-if="contextComplete" class="flex justify-end">
          <Button variant="ghost" size="sm" @click="isEditing = false">Done</Button>
        </div>
      </div>

      <div v-else class="flex flex-wrap items-center gap-2">
        <Badge v-for="chip in chips" :key="chip.label" variant="secondary" class="font-normal">
          <span class="text-muted-foreground mr-1">{{ chip.label }}:</span>{{ chip.value }}
        </Badge>
        <Button variant="ghost" size="sm" class="ml-auto" @click="isEditing = true">
          <Pencil class="size-3.5" />
          Change
        </Button>
      </div>
    </CardContent>
  </Card>
</template>
