<script setup lang="ts">
  import { computed, onMounted, ref, watch } from 'vue'
  import PageLayout from '../components/PageLayout.vue'
  import ManagementNav from '../components/ManagementNav.vue'
  import { bookingApi } from '../services/bookingApi'
  import {
    filterBookings,
    groupByDay,
    studentOptions,
    teacherOptions,
    type ScheduleScope,
  } from '../utils/scheduleFilter'
  import type { Booking } from '../types'

  const bookings = ref<Booking[]>([])
  const isLoading = ref(true)
  const loadError = ref('')

  const scope = ref<ScheduleScope>('upcoming')
  const teacherId = ref<number | null>(null)
  const studentId = ref<number | null>(null)

  const load = async () => {
    isLoading.value = true
    loadError.value = ''
    try {
      // Role-scoped server-side: an admin receives every booking, already
      // carrying the teacher, student, subject and branch names.
      bookings.value = await bookingApi.list()
    } catch (err) {
      loadError.value = err instanceof Error ? err.message : 'Failed to load the schedule'
    } finally {
      isLoading.value = false
    }
  }

  onMounted(load)

  // Options come from the bookings themselves, so a teacher or student with no
  // classes in the current scope is never offered as a dead-end filter.
  const scoped = computed(() =>
    filterBookings(bookings.value, { scope: scope.value, teacherId: null, studentId: null })
  )
  const teachers = computed(() => teacherOptions(scoped.value))
  const students = computed(() => studentOptions(scoped.value))

  const visible = computed(() =>
    filterBookings(bookings.value, {
      scope: scope.value,
      teacherId: teacherId.value,
      studentId: studentId.value,
    })
  )

  const groups = computed(() =>
    groupByDay(visible.value).map((day) => ({ ...day, label: formatDate(day.items[0].start_time) }))
  )

  // Switching scope can strip the selected person from the option list, which
  // would otherwise leave an empty page with a filter the user cannot see.
  watch(teachers, (list) => {
    if (teacherId.value !== null && !list.some((t) => t.id === teacherId.value)) {
      teacherId.value = null
    }
  })
  watch(students, (list) => {
    if (studentId.value !== null && !list.some((s) => s.id === studentId.value)) {
      studentId.value = null
    }
  })

  const hasFilters = computed(() => teacherId.value !== null || studentId.value !== null)

  const clearFilters = () => {
    teacherId.value = null
    studentId.value = null
  }

  function formatDate(date: string) {
    return new Date(date).toLocaleDateString('en-US', {
      weekday: 'long',
      month: 'long',
      day: 'numeric',
    })
  }

  function formatTime(date: string) {
    return new Date(date).toLocaleTimeString('en-US', {
      hour: '2-digit',
      minute: '2-digit',
      hour12: false,
    })
  }

  const summary = computed(() => {
    if (isLoading.value) return 'Loading...'
    const count = visible.value.length
    const noun = count === 1 ? 'class' : 'classes'
    return `${count} ${scope.value === 'past' ? 'past' : 'upcoming'} ${noun}`
  })

  const emptyMessage = computed(() => {
    if (hasFilters.value) return 'No classes match these filters'
    return scope.value === 'past' ? 'No past classes' : 'No upcoming classes'
  })
</script>

<template>
  <PageLayout title="Class Schedule" :show-cart="false">
    <div class="manage-root">
      <ManagementNav />
      <div class="manage-section">
        <div class="schedule-head">
          <h2 class="schedule-title">All classes</h2>
          <p class="schedule-subtitle">{{ summary }}</p>
        </div>

        <div class="schedule-tabs" role="tablist">
          <button
            type="button"
            role="tab"
            class="schedule-tab"
            :class="{ 'schedule-tab--active': scope === 'upcoming' }"
            :aria-selected="scope === 'upcoming'"
            @click="scope = 'upcoming'"
          >
            Upcoming
          </button>
          <button
            type="button"
            role="tab"
            class="schedule-tab"
            :class="{ 'schedule-tab--active': scope === 'past' }"
            :aria-selected="scope === 'past'"
            @click="scope = 'past'"
          >
            Past
          </button>
        </div>

        <div class="schedule-filters">
          <label class="schedule-filter">
            <span class="schedule-filter-label">Teacher</span>
            <select v-model="teacherId" class="schedule-select">
              <option :value="null">All teachers</option>
              <option v-for="teacher in teachers" :key="teacher.id" :value="teacher.id">
                {{ teacher.name }}
              </option>
            </select>
          </label>

          <label class="schedule-filter">
            <span class="schedule-filter-label">Student</span>
            <select v-model="studentId" class="schedule-select">
              <option :value="null">All students</option>
              <option v-for="student in students" :key="student.id" :value="student.id">
                {{ student.name }}
              </option>
            </select>
          </label>

          <button v-if="hasFilters" type="button" class="schedule-clear" @click="clearFilters">
            Clear filters
          </button>
        </div>

        <div v-if="isLoading" class="schedule-empty">Loading the schedule...</div>
        <div v-else-if="loadError" class="schedule-empty schedule-empty--error">
          {{ loadError }}
          <button type="button" class="schedule-retry" @click="load">Try again</button>
        </div>
        <div v-else-if="groups.length === 0" class="schedule-empty">{{ emptyMessage }}</div>

        <div v-else class="schedule-days">
          <section v-for="group in groups" :key="group.key" class="schedule-day">
            <h3 class="schedule-day-label">{{ group.label }}</h3>
            <div class="schedule-card-list">
              <article v-for="booking in group.items" :key="booking.id" class="schedule-card">
                <div class="schedule-card-time">
                  <span class="schedule-card-hour">{{ formatTime(booking.start_time) }}</span>
                  <span class="schedule-card-dash">–</span>
                  <span class="schedule-card-hour">{{ formatTime(booking.end_time) }}</span>
                </div>
                <div class="schedule-card-body">
                  <div class="schedule-card-subject">{{ booking.subject_name ?? 'Class' }}</div>
                  <dl class="schedule-card-meta">
                    <div class="schedule-meta-item">
                      <dt>Teacher</dt>
                      <dd>{{ booking.teacher_name ?? '—' }}</dd>
                    </div>
                    <div class="schedule-meta-item">
                      <dt>Student</dt>
                      <dd>{{ booking.student_name || '—' }}</dd>
                    </div>
                    <div class="schedule-meta-item">
                      <dt>Branch</dt>
                      <dd>{{ booking.branch_name ?? '—' }}</dd>
                    </div>
                  </dl>
                </div>
              </article>
            </div>
          </section>
        </div>
      </div>
    </div>
  </PageLayout>
</template>

<style scoped>
  .manage-root {
    width: 100%;
    height: 100%;
  }

  .manage-section {
    padding: 1.5rem;
    max-width: 64rem;
    margin: 0 auto;
  }

  .schedule-head {
    margin-bottom: 1rem;
  }

  .schedule-title {
    margin: 0 0 0.25rem;
    font-family: 'Instrument Sans', sans-serif;
    font-size: 1.125rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .schedule-subtitle {
    margin: 0;
    font-family: Inter, sans-serif;
    font-size: 0.8125rem;
    color: var(--text-secondary);
  }

  .schedule-tabs {
    display: flex;
    gap: 0.25rem;
    margin-bottom: 0.75rem;
  }

  .schedule-tab {
    padding: 0.375rem 1rem;
    font-family: Inter, sans-serif;
    font-size: 0.8125rem;
    font-weight: 500;
    color: var(--text-secondary);
    background: transparent;
    border: 1px solid var(--border-subtle);
    border-radius: 9999px;
    cursor: pointer;
    transition: all 0.15s;
  }

  .schedule-tab--active {
    color: #fff;
    background: var(--primary-indigo);
    border-color: var(--primary-indigo);
  }

  .schedule-filters {
    display: flex;
    flex-wrap: wrap;
    align-items: flex-end;
    gap: 0.75rem;
    margin-bottom: 1rem;
  }

  .schedule-filter {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    min-width: 12rem;
  }

  .schedule-filter-label {
    font-family: Inter, sans-serif;
    font-size: 0.6875rem;
    font-weight: 600;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    color: var(--text-muted);
  }

  .schedule-select {
    padding: 0.5rem 0.75rem;
    font-family: Inter, sans-serif;
    font-size: 0.875rem;
    color: var(--text-primary);
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: 8px;
    cursor: pointer;
  }

  .schedule-clear {
    padding: 0.5rem 0.875rem;
    font-family: Inter, sans-serif;
    font-size: 0.8125rem;
    font-weight: 500;
    color: var(--text-secondary);
    background: var(--bg-subtle);
    border: 1px solid var(--border-subtle);
    border-radius: 8px;
    cursor: pointer;
  }

  .schedule-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 0.75rem;
    min-height: 10rem;
    font-family: Inter, sans-serif;
    font-size: 0.875rem;
    color: var(--text-secondary);
    border-radius: 0.75rem;
    border: 2px dashed var(--border-medium);
    background: var(--bg-subtle);
    text-align: center;
    padding: 1rem;
  }

  .schedule-empty--error {
    color: #b91c1c;
    border-color: #fca5a5;
    background: #fef2f2;
  }

  .schedule-retry {
    padding: 0.375rem 0.875rem;
    font-family: Inter, sans-serif;
    font-size: 0.8125rem;
    font-weight: 500;
    color: #fff;
    background: var(--primary-indigo);
    border: none;
    border-radius: 8px;
    cursor: pointer;
  }

  .schedule-days {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }

  .schedule-day-label {
    margin: 0 0 0.5rem;
    font-family: Inter, sans-serif;
    font-size: 0.75rem;
    font-weight: 600;
    letter-spacing: 0.05em;
    text-transform: uppercase;
    color: var(--text-muted);
  }

  .schedule-card-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .schedule-card {
    display: flex;
    gap: 1rem;
    padding: 1rem;
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: 0.75rem;
  }

  .schedule-card-time {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    flex-shrink: 0;
    min-width: 4rem;
    font-family: 'JetBrains Mono', 'SF Mono', monospace;
    font-size: 0.8125rem;
    color: var(--text-primary);
  }

  .schedule-card-dash {
    color: var(--text-muted);
    line-height: 1;
  }

  .schedule-card-hour {
    font-weight: 600;
  }

  .schedule-card-body {
    min-width: 0;
    flex: 1;
  }

  .schedule-card-subject {
    font-family: Inter, sans-serif;
    font-size: 0.9375rem;
    font-weight: 600;
    color: var(--text-primary);
    margin-bottom: 0.5rem;
  }

  .schedule-card-meta {
    display: flex;
    flex-wrap: wrap;
    gap: 0.25rem 1.25rem;
    margin: 0;
  }

  .schedule-meta-item dt {
    font-family: Inter, sans-serif;
    font-size: 0.6875rem;
    font-weight: 600;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    color: var(--text-muted);
  }

  .schedule-meta-item dd {
    margin: 0;
    font-family: Inter, sans-serif;
    font-size: 0.8125rem;
    color: var(--text-secondary);
  }

  @media (max-width: 767px) {
    .manage-section {
      padding: 1rem;
    }

    .schedule-filter {
      min-width: 100%;
    }

    .schedule-card {
      flex-direction: column;
      gap: 0.5rem;
    }

    .schedule-card-time {
      flex-direction: row;
      gap: 0.375rem;
    }
  }
</style>
