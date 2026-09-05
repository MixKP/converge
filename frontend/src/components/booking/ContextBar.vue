<script setup lang="ts">
  import { ref, computed, watch } from 'vue'
  import { Pencil } from '@lucide/vue'
  import { useBooking } from '../../composables/useBooking'
  import { useBookingContext } from '../../composables/useBookingContext'
  import { useNumberSelect } from '../../composables/useSelectProxy'
  import { Button } from '@/components/ui/button'
  import { Card, CardContent } from '@/components/ui/card'
  import { Label } from '@/components/ui/label'
  import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
  } from '@/components/ui/select'

  const {
    subjects,
    branches,
    activeBranches,
    students,
    selectedStudentId,
    selectedSubjectId,
    selectedBranchId,
  } = useBooking()

  const { contextComplete } = useBookingContext()

  const studentValue = useNumberSelect(selectedStudentId)
  const subjectValue = useNumberSelect(selectedSubjectId)
  const branchValue = useNumberSelect(selectedBranchId)

  const isEditing = ref(false)

  watch(contextComplete, (complete) => {
    if (complete) isEditing.value = false
  })

  const expanded = computed(() => isEditing.value || !contextComplete.value)

  const chips = computed(() => [
    { label: 'Student', value: students.value.find((s) => s.id === selectedStudentId.value)?.name },
    { label: 'Subject', value: subjects.value.find((s) => s.id === selectedSubjectId.value)?.name },
    { label: 'Branch', value: branches.value.find((b) => b.id === selectedBranchId.value)?.name },
  ])
</script>

<template>
  <Card v-if="expanded">
    <CardContent class="py-4">
      <div class="flex flex-col gap-4">
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
                <SelectItem
                  v-for="branch in activeBranches"
                  :key="branch.id"
                  :value="String(branch.id)"
                >
                  {{ branch.name }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
        <div v-if="contextComplete" class="flex justify-end">
          <Button variant="ghost" size="sm" @click="isEditing = false">Done</Button>
        </div>
      </div>
    </CardContent>
  </Card>

  <div v-else class="flex flex-wrap items-center gap-x-2 gap-y-1 px-1 text-xs">
    <template v-for="(chip, index) in chips" :key="chip.label">
      <span v-if="index > 0" class="text-muted-foreground/40" aria-hidden="true">·</span>
      <span class="text-muted-foreground">
        {{ chip.label }}:
        <span class="text-foreground font-medium">{{ chip.value }}</span>
      </span>
    </template>
    <Button
      variant="ghost"
      size="sm"
      class="text-muted-foreground ml-auto h-6 gap-1 px-2 text-xs"
      @click="isEditing = true"
    >
      <Pencil class="size-3" />
      Change
    </Button>
  </div>
</template>
