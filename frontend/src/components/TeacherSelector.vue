<script setup lang="ts">
  import { computed } from 'vue'
  import type { Teacher } from '../types/teacher'

  interface Props {
    teachers: Teacher[]
    selectedTeacherId: number | null
    slotsCount?: number
  }

  const props = defineProps<Props>()

  const emit = defineEmits<{
    'update:selectedTeacherId': [value: number | null]
    'clear-slots': []
  }>()

  const selectedTeacher = computed(() =>
    props.teachers.find((t) => t.id === props.selectedTeacherId)
  )
</script>

<template>
  <div class="selector-section">
    <div v-if="!selectedTeacher" class="teacher-selector-form">
      <label class="form-label">Teacher</label>
      <select
        :value="selectedTeacherId"
        class="teacher-select"
        @change="
          (() => {
            const value = ($event.target as HTMLSelectElement).value
            emit('update:selectedTeacherId', value === '' ? null : Number(value))
          })()
        "
      >
        <option value="">Select your name</option>
        <option v-for="teacher in teachers" :key="teacher.id" :value="teacher.id">
          {{ teacher.name }}
        </option>
      </select>
    </div>
    <div v-else class="teacher-display-box">
      <span class="teacher-label">TEACHER</span>
      <span class="teacher-name">{{ selectedTeacher.name }}</span>
      <button
        type="button"
        class="change-teacher-btn"
        @click="emit('update:selectedTeacherId', null)"
      >
        Change
      </button>
    </div>
    <div v-if="slotsCount && slotsCount > 0" class="slots-info">
      <span class="slots-count">{{ slotsCount }} slot{{ slotsCount > 1 ? 's' : '' }}</span>
      <button type="button" class="clear-btn" @click="emit('clear-slots')">Clear slots</button>
    </div>
  </div>
</template>

<style scoped>
  .selector-section {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 1rem;
    padding: 0.5rem 0.75rem;
    position: relative;
    z-index: 1;
  }

  .teacher-selector-form {
    display: flex;
    flex-direction: column;
    gap: 0.375rem;
  }

  .form-label {
    font-family: 'Inter', sans-serif;
    font-size: 0.8125rem;
    font-weight: 500;
    color: var(--text-primary);
  }

  .teacher-display-box {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 0.5rem;
    padding: 0.75rem 1rem;
    background: var(--bg-card);
    border: 1px solid var(--border-medium);
    border-radius: 6px;
    min-width: 280px;
  }

  .teacher-label {
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.6875rem;
    font-weight: 600;
    letter-spacing: 0.1em;
    color: var(--text-secondary);
    text-transform: uppercase;
  }

  .teacher-name {
    flex: 1;
    font-family: 'Inter', sans-serif;
    font-size: 0.875rem;
    font-weight: 400;
    color: var(--text-primary);
  }

  .change-teacher-btn {
    padding: 0.375rem 0.75rem;
    font-size: 0.75rem;
    font-weight: 500;
    color: var(--primary-indigo);
    background: transparent;
    border: 1px solid var(--border-medium);
    border-radius: 4px;
    cursor: pointer;
    transition: all 0.2s ease;
    font-family: 'Inter', sans-serif;
  }

  .change-teacher-btn:hover {
    background: var(--bg-subtle);
    border-color: var(--primary-indigo);
  }

  .teacher-select {
    flex: 1;
    max-width: 320px;
    padding: 0.75rem 1rem;
    font-size: 0.875rem;
    color: var(--text-primary);
    background: var(--bg-card);
    border: 1px solid var(--border-medium);
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.2s ease;
    font-family: 'Inter', sans-serif;
    appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' fill='none' stroke='%233e4c7a' viewBox='0 0 24 24'%3E%3Cpath stroke-linecap='round' stroke-linejoin='round' stroke-width='2' d='M8 10l4 4 4-4'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 0.75rem center;
    background-size: 1rem;
    padding-right: 2.25rem;
  }

  .teacher-select:hover {
    border-color: var(--primary-indigo);
  }

  .teacher-select:focus {
    outline: none;
    border-color: var(--primary-indigo);
    box-shadow: 0 0 0 3px rgba(62, 76, 122, 0.1);
  }

  .slots-info {
    display: flex;
    align-items: center;
    gap: 1rem;
  }

  .slots-count {
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--primary-indigo);
    font-family: 'JetBrains Mono', monospace;
  }

  .clear-btn {
    font-size: 0.75rem;
    font-weight: 500;
    color: var(--accent-coral);
    background: transparent;
    border: none;
    cursor: pointer;
    padding: 0.375rem 0.625rem;
    border-radius: 4px;
    transition: all 0.2s ease;
  }

  .clear-btn:hover {
    background: rgba(201, 109, 93, 0.1);
  }

  @media (max-width: 640px) {
    .selector-section {
      flex-direction: column;
      align-items: stretch;
      gap: 0.75rem;
    }

    .teacher-selector-form,
    .teacher-display-box {
      width: 100%;
    }

    .teacher-display-box {
      min-width: unset;
    }

    .teacher-select {
      max-width: 100%;
    }

    .slots-info {
      justify-content: space-between;
    }
  }
</style>
