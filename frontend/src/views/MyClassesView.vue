<script setup lang="ts">
  import { computed, onMounted, ref } from 'vue'
  import PageLayout from '../components/PageLayout.vue'
  import { bookingApi } from '../services/bookingApi'
  import { useAuthStore } from '../stores/authStore'
  import type { Booking } from '../types'

  type Scope = 'upcoming' | 'past'

  const auth = useAuthStore()
  const isParent = computed(() => auth.role === 'parent')

  const bookings = ref<Booking[]>([])
  const isLoading = ref(true)
  const loadError = ref('')
  const scope = ref<Scope>('upcoming')
  const studentFilter = ref<number | null>(null)

  const load = async () => {
    isLoading.value = true
    loadError.value = ''
    try {
      // The endpoint is scoped by role server-side, so a student receives only
      // their own classes and a parent only their linked students'.
      bookings.value = await bookingApi.list()
    } catch (err) {
      loadError.value = err instanceof Error ? err.message : 'Failed to load your classes'
    } finally {
      isLoading.value = false
    }
  }

  onMounted(load)

  // Derived from the bookings themselves: the parent/student link endpoints are
  // admin-only, so a linked student with no classes yet has no chip to show.
  const students = computed(() => {
    const byId = new Map<number, string>()
    for (const b of bookings.value) byId.set(b.student_id, b.student_name)
    return [...byId]
      .map(([id, name]) => ({ id, name }))
      .sort((a, b) => a.name.localeCompare(b.name))
  })

  const scoped = computed(() => {
    const now = Date.now()
    return bookings.value.filter((b) => {
      const isPast = new Date(b.end_time).getTime() < now
      return scope.value === 'past' ? isPast : !isPast
    })
  })

  const visible = computed(() => {
    const list =
      studentFilter.value === null
        ? scoped.value
        : scoped.value.filter((b) => b.student_id === studentFilter.value)
    // Upcoming reads best soonest-first; past reads best most-recent-first.
    return [...list].sort((a, b) => {
      const delta = new Date(a.start_time).getTime() - new Date(b.start_time).getTime()
      return scope.value === 'past' ? -delta : delta
    })
  })

  const dayKey = (iso: string) => new Date(iso).toDateString()

  const groups = computed(() => {
    const out: { key: string; label: string; items: Booking[] }[] = []
    for (const b of visible.value) {
      const key = dayKey(b.start_time)
      const last = out.at(-1)
      if (last?.key === key) last.items.push(b)
      else out.push({ key, label: formatDate(b.start_time), items: [b] })
    }
    return out
  })

  const upcomingCount = computed(
    () => bookings.value.filter((b) => new Date(b.end_time).getTime() >= Date.now()).length
  )

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

  const emptyMessage = computed(() => {
    if (scope.value === 'past') return 'No past classes'
    return isParent.value ? 'No upcoming classes for your students' : 'You have no upcoming classes'
  })
</script>

<template>
  <PageLayout title="My Classes" :show-cart="false">
    <div class="classes-root">
      <div class="classes-section">
        <div class="classes-head">
          <h2 class="classes-title">{{ isParent ? "Your students' classes" : 'Your classes' }}</h2>
          <p class="classes-subtitle">
            {{
              isLoading
                ? 'Loading...'
                : `${upcomingCount} upcoming ${upcomingCount === 1 ? 'class' : 'classes'}`
            }}
          </p>
        </div>

        <div class="classes-tabs" role="tablist">
          <button
            type="button"
            role="tab"
            class="classes-tab"
            :class="{ 'classes-tab--active': scope === 'upcoming' }"
            :aria-selected="scope === 'upcoming'"
            @click="scope = 'upcoming'"
          >
            Upcoming
          </button>
          <button
            type="button"
            role="tab"
            class="classes-tab"
            :class="{ 'classes-tab--active': scope === 'past' }"
            :aria-selected="scope === 'past'"
            @click="scope = 'past'"
          >
            Past
          </button>
        </div>

        <div v-if="isParent && students.length > 1" class="classes-filters">
          <button
            type="button"
            class="classes-chip"
            :class="{ 'classes-chip--active': studentFilter === null }"
            @click="studentFilter = null"
          >
            All students
          </button>
          <button
            v-for="student in students"
            :key="student.id"
            type="button"
            class="classes-chip"
            :class="{ 'classes-chip--active': studentFilter === student.id }"
            @click="studentFilter = student.id"
          >
            {{ student.name }}
          </button>
        </div>

        <div v-if="isLoading" class="classes-empty">Loading your classes...</div>
        <div v-else-if="loadError" class="classes-empty classes-empty--error">
          {{ loadError }}
          <button type="button" class="classes-retry" @click="load">Try again</button>
        </div>
        <div v-else-if="groups.length === 0" class="classes-empty">{{ emptyMessage }}</div>

        <div v-else class="classes-days">
          <section v-for="group in groups" :key="group.key" class="classes-day">
            <h3 class="classes-day-label">{{ group.label }}</h3>
            <div class="classes-card-list">
              <article v-for="booking in group.items" :key="booking.id" class="classes-card">
                <div class="classes-card-time">
                  <span class="classes-card-hour">{{ formatTime(booking.start_time) }}</span>
                  <span class="classes-card-dash">–</span>
                  <span class="classes-card-hour">{{ formatTime(booking.end_time) }}</span>
                </div>
                <div class="classes-card-body">
                  <div class="classes-card-subject">{{ booking.subject_name ?? 'Class' }}</div>
                  <dl class="classes-card-meta">
                    <div class="classes-meta-item">
                      <dt>Teacher</dt>
                      <dd>{{ booking.teacher_name ?? '—' }}</dd>
                    </div>
                    <div class="classes-meta-item">
                      <dt>Branch</dt>
                      <dd>{{ booking.branch_name ?? '—' }}</dd>
                    </div>
                    <div v-if="isParent" class="classes-meta-item">
                      <dt>Student</dt>
                      <dd>{{ booking.student_name }}</dd>
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
  .classes-root {
    width: 100%;
    height: 100%;
  }

  .classes-section {
    padding: 1.5rem;
    max-width: 48rem;
    margin: 0 auto;
  }

  .classes-head {
    margin-bottom: 1rem;
  }

  .classes-title {
    margin: 0 0 0.25rem;
    font-family: 'Instrument Sans', sans-serif;
    font-size: 1.125rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .classes-subtitle {
    margin: 0;
    font-family: Inter, sans-serif;
    font-size: 0.8125rem;
    color: var(--text-secondary);
  }

  .classes-tabs {
    display: flex;
    gap: 0.25rem;
    margin-bottom: 0.75rem;
  }

  .classes-tab {
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

  .classes-tab--active {
    color: #fff;
    background: var(--primary-indigo);
    border-color: var(--primary-indigo);
  }

  .classes-filters {
    display: flex;
    flex-wrap: wrap;
    gap: 0.375rem;
    margin-bottom: 1rem;
  }

  .classes-chip {
    padding: 0.25rem 0.75rem;
    font-family: Inter, sans-serif;
    font-size: 0.75rem;
    font-weight: 500;
    color: var(--text-secondary);
    background: var(--bg-subtle);
    border: 1px solid var(--border-subtle);
    border-radius: 9999px;
    cursor: pointer;
    transition: all 0.15s;
  }

  .classes-chip--active {
    color: var(--primary-indigo);
    background: rgba(62, 76, 122, 0.1);
    border-color: rgba(62, 76, 122, 0.3);
  }

  .classes-empty {
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

  .classes-empty--error {
    color: #b91c1c;
    border-color: #fca5a5;
    background: #fef2f2;
  }

  .classes-retry {
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

  .classes-days {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }

  .classes-day-label {
    margin: 0 0 0.5rem;
    font-family: Inter, sans-serif;
    font-size: 0.75rem;
    font-weight: 600;
    letter-spacing: 0.05em;
    text-transform: uppercase;
    color: var(--text-muted);
  }

  .classes-card-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .classes-card {
    display: flex;
    gap: 1rem;
    padding: 1rem;
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: 0.75rem;
  }

  .classes-card-time {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    flex-shrink: 0;
    min-width: 4rem;
    font-family: 'JetBrains Mono', 'SF Mono', monospace;
    font-size: 0.8125rem;
    color: var(--text-primary);
  }

  .classes-card-dash {
    color: var(--text-muted);
    line-height: 1;
  }

  .classes-card-hour {
    font-weight: 600;
  }

  .classes-card-body {
    min-width: 0;
    flex: 1;
  }

  .classes-card-subject {
    font-family: Inter, sans-serif;
    font-size: 0.9375rem;
    font-weight: 600;
    color: var(--text-primary);
    margin-bottom: 0.5rem;
  }

  .classes-card-meta {
    display: flex;
    flex-wrap: wrap;
    gap: 0.25rem 1.25rem;
    margin: 0;
  }

  .classes-meta-item dt {
    font-family: Inter, sans-serif;
    font-size: 0.6875rem;
    font-weight: 600;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    color: var(--text-muted);
  }

  .classes-meta-item dd {
    margin: 0;
    font-family: Inter, sans-serif;
    font-size: 0.8125rem;
    color: var(--text-secondary);
  }

  @media (max-width: 767px) {
    .classes-section {
      padding: 1rem;
    }

    .classes-card {
      flex-direction: column;
      gap: 0.5rem;
    }

    .classes-card-time {
      flex-direction: row;
      gap: 0.375rem;
    }
  }
</style>
