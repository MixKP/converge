<script setup lang="ts">
  import { computed } from 'vue'
  import type { Subject, Branch, Teacher, WeeklySlot } from '../types'

  interface Props {
    subjects: Subject[]
    branches: Branch[]
    filteredTeachers: Teacher[]
    selectedSubjectId: number | null
    selectedBranchId: number | null
    selectedTeacherId: number | null
    preferredTeacherId: number | null
    durationMinutes: number
    manualSlots: WeeklySlot[]
    newSlotDay: number
    newSlotStart: string
    newSlotEnd: string
    currentStep: number
    canSelectTeacher: boolean
    canAddManualSlot: boolean
  }

  defineProps<Props>()

  const emit = defineEmits<{
    'update:selectedSubjectId': [value: number | null]
    'update:selectedBranchId': [value: number | null]
    'update:selectedTeacherId': [value: number | null]
    'update:preferredTeacherId': [value: number | null]
    'update:durationMinutes': [value: number]
    'update:newSlotDay': [value: number]
    'update:newSlotStart': [value: string]
    'update:newSlotEnd': [value: string]
    addManualSlot: []
    removeManualSlot: [index: number]
  }>()

  const DAY_NAMES = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']

  const getSlotLabel = (slot: WeeklySlot): string => {
    return `${DAY_NAMES[slot.day_of_week]} ${slot.start} - ${slot.end}`
  }

  const timeOptions = computed(() => {
    const options: string[] = []
    for (let hour = 8; hour <= 18; hour++) {
      const hourStr = String(hour).padStart(2, '0')
      options.push(`${hourStr}:00`, `${hourStr}:30`)
    }
    options.push('19:00')
    return options
  })
</script>

<template>
  <div class="space-y-6 flex-1 overflow-y-auto">
    <div class="space-y-3">
      <label class="text-sm font-semibold text-(--ink-primary) flex items-center gap-2">
        <span
          :class="[
            'w-5 h-5 rounded-full text-xs flex items-center justify-center',
            currentStep >= 1
              ? 'bg-(--accent-terracotta) text-white'
              : 'bg-gray-200 text-(--text-secondary)',
          ]"
        >
          1
        </span>
        Subject
      </label>
      <select
        :value="selectedSubjectId"
        class="w-full bg-white border border-(--border-strong) rounded-lg px-4 py-3 text-sm focus:outline-none focus:ring-2 focus:ring-(--accent-terracotta)/20 focus:border-(--accent-terracotta) transition-all"
        @change="
          emit(
            'update:selectedSubjectId',
            ($event.target as HTMLSelectElement).value
              ? Number(($event.target as HTMLSelectElement).value)
              : null
          )
        "
      >
        <option :value="null">Choose your subject</option>
        <option v-for="subject in subjects" :key="subject.id" :value="subject.id">
          {{ subject.name }}
        </option>
      </select>
      <p class="text-xs text-(--text-secondary)">Select the subject you want to book</p>
    </div>

    <div class="space-y-3">
      <label class="text-sm font-semibold text-(--ink-primary) flex items-center gap-2">
        <span
          :class="[
            'w-5 h-5 rounded-full border-2 text-xs flex items-center justify-center',
            currentStep >= 2
              ? 'bg-(--accent-terracotta) border-(--accent-terracotta) text-white'
              : 'border-(--border-strong) text-(--text-secondary)',
          ]"
        >
          2
        </span>
        Branch
      </label>
      <select
        :value="selectedBranchId"
        :disabled="!selectedSubjectId"
        class="w-full bg-white border border-(--border-strong) rounded-lg px-4 py-3 text-sm focus:outline-none focus:ring-2 focus:ring-(--accent-terracotta)/20 focus:border-(--accent-terracotta) transition-all disabled:opacity-50 disabled:cursor-not-allowed"
        @change="
          emit(
            'update:selectedBranchId',
            ($event.target as HTMLSelectElement).value
              ? Number(($event.target as HTMLSelectElement).value)
              : null
          )
        "
      >
        <option :value="null">Choose your branch</option>
        <option v-for="branch in branches" :key="branch.id" :value="branch.id">
          {{ branch.name }}
        </option>
      </select>
      <p v-if="!selectedSubjectId" class="text-xs text-(--text-secondary)">
        Select a subject first
      </p>
    </div>

    <div v-if="canSelectTeacher" class="space-y-3">
      <label class="text-sm font-semibold text-(--ink-primary) flex items-center gap-2">
        <span
          :class="[
            'w-5 h-5 rounded-full border-2 text-xs flex items-center justify-center',
            currentStep >= 3
              ? 'bg-(--accent-terracotta) border-(--accent-terracotta) text-white'
              : 'border-(--border-strong) text-(--text-secondary)',
          ]"
        >
          3
        </span>
        Preferred Teacher
        <span class="text-xs font-normal text-(--text-secondary)">(Optional)</span>
      </label>
      <select
        :value="selectedTeacherId"
        class="w-full bg-white border border-(--border-strong) rounded-lg px-4 py-3 text-sm focus:outline-none focus:ring-2 focus:ring-(--accent-terracotta)/20 focus:border-(--accent-terracotta) transition-all"
        @change="
          emit(
            'update:selectedTeacherId',
            ($event.target as HTMLSelectElement).value
              ? Number(($event.target as HTMLSelectElement).value)
              : null
          )
        "
      >
        <option :value="null">Any available teacher</option>
        <option v-for="teacher in filteredTeachers" :key="teacher.id" :value="teacher.id">
          {{ teacher.name }}
        </option>
      </select>
    </div>

    <div class="space-y-3">
      <label class="text-sm font-semibold text-(--ink-primary) flex items-center gap-2">
        <span
          :class="[
            'w-5 h-5 rounded-full border-2 text-xs flex items-center justify-center',
            currentStep >= 4
              ? 'bg-(--accent-terracotta) border-(--accent-terracotta) text-white'
              : 'border-(--border-strong) text-(--text-secondary)',
          ]"
        >
          4
        </span>
        Preferred Time Slots
      </label>

      <div v-if="manualSlots.length > 0" class="flex flex-wrap gap-2 mb-2">
        <span
          v-for="(slot, index) in manualSlots"
          :key="index"
          class="inline-flex items-center gap-1 bg-(--accent-terracotta-soft) text-(--ink-primary) text-xs px-2 py-1 rounded-full"
        >
          {{ getSlotLabel(slot) }}
          <button
            type="button"
            class="text-(--accent-terracotta) hover:text-red-600"
            @click="emit('removeManualSlot', index)"
          >
            ×
          </button>
        </span>
      </div>

      <div class="flex gap-2">
        <select
          :value="newSlotDay"
          :disabled="!selectedBranchId"
          class="flex-1 bg-white border border-(--border-strong) rounded-lg px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-(--accent-terracotta)/20 focus:border-(--accent-terracotta) disabled:opacity-50"
          @change="emit('update:newSlotDay', Number(($event.target as HTMLSelectElement).value))"
        >
          <option v-for="(day, index) in DAY_NAMES" :key="day" :value="index">
            {{ day }}
          </option>
        </select>
        <select
          :value="newSlotStart"
          :disabled="!selectedBranchId"
          class="flex-1 bg-white border border-(--border-strong) rounded-lg px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-(--accent-terracotta)/20 focus:border-(--accent-terracotta) disabled:opacity-50"
          @change="emit('update:newSlotStart', ($event.target as HTMLSelectElement).value)"
        >
          <option v-for="time in timeOptions" :key="time" :value="time">
            {{ time }}
          </option>
        </select>
        <select
          :value="newSlotEnd"
          :disabled="!selectedBranchId"
          class="flex-1 bg-white border border-(--border-strong) rounded-lg px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-(--accent-terracotta)/20 focus:border-(--accent-terracotta) disabled:opacity-50"
          @change="emit('update:newSlotEnd', ($event.target as HTMLSelectElement).value)"
        >
          <option v-for="time in timeOptions" :key="time" :value="time">
            {{ time }}
          </option>
        </select>
        <button
          type="button"
          :disabled="!canAddManualSlot || !selectedBranchId"
          class="px-4 py-2 bg-(--accent-terracotta) text-white rounded-lg hover:bg-(--accent-terracotta-dark) disabled:opacity-50 disabled:cursor-not-allowed transition-all"
          @click="emit('addManualSlot')"
        >
          +
        </button>
      </div>
      <p v-if="!selectedBranchId" class="text-xs text-(--text-secondary)">Select a branch first</p>
    </div>

    <div class="space-y-3">
      <label class="text-sm font-semibold text-(--ink-primary)">Duration (minutes)</label>
      <select
        :value="durationMinutes"
        class="w-full bg-white border border-(--border-strong) rounded-lg px-4 py-3 text-sm focus:outline-none focus:ring-2 focus:ring-(--accent-terracotta)/20 focus:border-(--accent-terracotta) transition-all"
        @change="emit('update:durationMinutes', Number(($event.target as HTMLSelectElement).value))"
      >
        <option :value="30">30 minutes</option>
        <option :value="60">1 hour</option>
        <option :value="90">1.5 hours</option>
        <option :value="120">2 hours</option>
      </select>
    </div>
  </div>
</template>
