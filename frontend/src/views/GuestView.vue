<script setup lang="ts">
  import { ref, watch, onMounted } from 'vue'
  import { useScreenSize } from '../composables/useScreenSize'
  import { subjectApi } from '../services/subjectApi'
  import { teacherApi } from '../services/teacherApi'
  import { availabilityApi } from '../services/availabilityApi'
  import { transformBackendAvailability } from '../utils/availabilityTransform'
  import type { Subject, Teacher, WeeklySlot } from '../types'
  import type { BusinessHoursInput } from '@fullcalendar/core'
  import PageLayout from '../components/PageLayout.vue'
  import Calendar from '../components/Calendar.vue'
  import CalendarDisabledOverlay from '../components/CalendarDisabledOverlay.vue'
  import FormSelect from '../components/form/FormSelect.vue'

  const { isMobile, isTablet } = useScreenSize()

  const subjects = ref<Subject[]>([])
  const selectedSubjectId = ref<number | null>(null)
  const filteredTeachers = ref<Teacher[]>([])
  const availabilityCache = ref<Map<number, WeeklySlot[]>>(new Map())
  const businessHours = ref<BusinessHoursInput>([])
  const isLoading = ref(false)
  const isDataReady = ref(false)

  onMounted(async () => {
    try {
      const [subjectData, availabilityData] = await Promise.all([
        subjectApi.getAll(),
        availabilityApi.getAll(),
      ])
      subjects.value = subjectData
      availabilityCache.value = transformBackendAvailability(availabilityData)
    } finally {
      isDataReady.value = true
    }
  })

  watch(selectedSubjectId, async (subjectId) => {
    if (!subjectId) {
      filteredTeachers.value = []
      businessHours.value = []
      return
    }

    isLoading.value = true
    try {
      filteredTeachers.value = await teacherApi.getBySubject(subjectId)

      const teacherIds = filteredTeachers.value.map((t) => t.id)
      const allSlots: WeeklySlot[] = []
      for (const tid of teacherIds) {
        const cached = availabilityCache.value.get(tid)
        if (cached) allSlots.push(...cached)
      }

      businessHours.value = allSlots.map((slot) => ({
        daysOfWeek: [slot.day_of_week],
        startTime: slot.start,
        endTime: slot.end,
      }))
    } finally {
      isLoading.value = false
    }
  })
</script>

<template>
  <PageLayout title="Converge" :show-cart="false">
    <div class="guest-root">
      <!-- Mobile Layout (≤425px) -->
      <div v-if="isMobile" class="guest-mobile">
        <div class="guest-select-section">
          <label
            class="guest-label"
          >
            <span class="h-1 w-1 rounded-full bg-(--text-muted)"></span>
            <span>SUBJECT</span>
          </label>
          <FormSelect
            v-model="selectedSubjectId"
            name="subject_id"
            select-class="mobile-select"
            aria-label="Select subject"
            :show-error="false"
          >
            <option :value="null">Select a subject</option>
            <option v-for="subject in subjects" :key="subject.id" :value="subject.id">
              {{ subject.name }}
            </option>
          </FormSelect>
        </div>

        <div v-if="!selectedSubjectId" class="guest-empty-state">
          <svg
            class="h-7 w-7 text-(--accent-coral) opacity-80"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="1.5"
              d="M15.042 21.672L13.684 16.6m0 0l-2.51 2.225.569-9.47 5.227 7.917-3.286-.672zM12 2.25V4.5m5.834.166l-1.591 1.591M20.25 10.5H18M7.757 14.743l-1.59 1.59M6 10.5H3.75m4.007-4.243l-1.59-1.59"
            />
          </svg>
          <p class="m-0 font-[Inter,sans-serif] text-[0.8125rem] text-(--text-secondary)">
            Select a subject to browse availability
          </p>
        </div>

        <div v-else class="guest-mobile-calendar">
          <Calendar
            :business-hours="businessHours"
            constraint="businessHours"
          />
        </div>
      </div>

      <!-- Tablet Layout (426px - 1023px) -->
      <div v-else-if="isTablet" class="guest-tablet">
        <div class="guest-select-section">
          <label class="guest-label">
            <span class="h-1 w-1 rounded-full bg-(--text-muted)"></span>
            <span>SUBJECT</span>
          </label>
          <FormSelect
            v-model="selectedSubjectId"
            name="subject_id"
            select-class="tablet-select"
            aria-label="Select subject"
            :show-error="false"
          >
            <option :value="null">Select a subject</option>
            <option v-for="subject in subjects" :key="subject.id" :value="subject.id">
              {{ subject.name }}
            </option>
          </FormSelect>
        </div>

        <div v-if="!selectedSubjectId" class="guest-empty-state">
          <svg
            class="h-9 w-9 text-(--accent-coral) opacity-80"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="1.5"
              d="M15.042 21.672L13.684 16.6m0 0l-2.51 2.225.569-9.47 5.227 7.917-3.286-.672zM12 2.25V4.5m5.834.166l-1.591 1.591M20.25 10.5H18M7.757 14.743l-1.59 1.59M6 10.5H3.75m4.007-4.243l-1.59-1.59"
            />
          </svg>
          <p class="m-0 font-[Inter,sans-serif] text-[0.9375rem] text-(--text-secondary)">
            Select a subject to browse availability
          </p>
        </div>

        <div v-else class="guest-tablet-calendar">
          <Calendar
            :business-hours="businessHours"
            constraint="businessHours"
          />
        </div>
      </div>

      <!-- Desktop Layout (≥1024px) -->
      <div v-else class="guest-desktop">
        <div class="guest-desktop-inner">
          <div class="guest-header">
            <div>
              <h2
                class="font-['Instrument_Sans',sans-serif] text-xl font-semibold tracking-[-0.01em] text-(--text-primary)"
              >
                Teacher Availability
              </h2>
              <p
                class="mt-1 font-['JetBrains_Mono',monospace] text-[0.8125rem] tracking-wider text-(--text-secondary)"
              >
                Browse available time slots by subject
              </p>
            </div>
            <div class="guest-header-select">
              <FormSelect
                v-model="selectedSubjectId"
                name="subject_id"
                select-class="field-select"
                aria-label="Select subject"
                :show-error="false"
              >
                <option :value="null">Select a subject</option>
                <option v-for="subject in subjects" :key="subject.id" :value="subject.id">
                  {{ subject.name }}
                </option>
              </FormSelect>
            </div>
          </div>

          <div class="guest-desktop-calendar-wrap">
            <div class="guest-calendar-inner">
              <Calendar
                v-if="selectedSubjectId"
                :business-hours="businessHours"
                constraint="businessHours"
              />
              <CalendarDisabledOverlay
                v-else
                message="Select a subject to browse availability"
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  </PageLayout>
</template>

<style scoped>
  .guest-root {
    width: 100%;
    height: 100%;
    overflow: hidden;
  }

  /* --- Mobile --- */
  .guest-mobile {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    padding: 1rem;
    padding-bottom: calc(1rem + env(safe-area-inset-bottom));
    max-width: 100%;
    overflow-x: hidden;
  }

  .guest-select-section {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .guest-label {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-family: Inter, sans-serif;
    font-size: 0.6875rem;
    font-weight: 600;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--text-muted);
  }

  .guest-empty-state {
    display: flex;
    min-height: 16rem;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 0.875rem;
    border-radius: 0.5rem;
    border: 2px dashed var(--border-medium);
    background: var(--bg-subtle);
    padding: 1.5rem;
  }

  .guest-mobile-calendar {
    display: flex;
    height: 25rem;
    flex-direction: column;
    overflow: hidden;
    border-radius: 0.5rem;
    border: 1px solid var(--border-subtle);
    background: var(--bg-card);
  }

  .guest-mobile-calendar :deep(.fc) {
    height: 100% !important;
    font-size: 0.75rem;
    touch-action: pan-y pinch-zoom;
  }

  .guest-mobile-calendar :deep(.fc-view-harness),
  .guest-mobile-calendar :deep(.fc-view),
  .guest-mobile-calendar :deep(.fc-timegrid),
  .guest-mobile-calendar :deep(.fc-timegrid-body),
  .guest-mobile-calendar :deep(.fc-timegrid-slots) {
    touch-action: pan-y pinch-zoom;
  }

  .guest-mobile-calendar :deep(.fc-timegrid-slot),
  .guest-mobile-calendar :deep(.fc-timegrid-slot-lane) {
    min-height: 36px;
    touch-action: pan-y;
  }

  .guest-mobile-calendar :deep(.fc-col-header-cell) {
    touch-action: manipulation;
    padding: 0.25rem 0.0625rem !important;
    height: 24px !important;
  }

  .guest-mobile-calendar :deep(.fc-col-header-cell-cushion) {
    font-size: 0.5rem !important;
    padding: 0 !important;
    font-weight: 600 !important;
    letter-spacing: 0.05em !important;
    line-height: 1 !important;
  }

  .guest-mobile-calendar :deep(.fc-timegrid-axis) {
    width: 24px !important;
    touch-action: pan-y;
  }

  .guest-mobile-calendar :deep(.fc-timegrid-slot-label) {
    font-size: 0.5rem !important;
    padding: 0 !important;
  }

  .guest-mobile-calendar :deep(.fc-timegrid-slot-label-cushion) {
    padding: 0 0.0625rem !important;
    line-height: 1 !important;
  }

  .guest-mobile-calendar :deep(.fc-timegrid-axis-frame) {
    padding: 0 !important;
  }

  .guest-mobile-calendar :deep(.fc .fc-toolbar.fc-header-toolbar) {
    padding: 0.125rem 0.375rem !important;
    gap: 0.125rem;
    margin: 0 !important;
  }

  .guest-mobile-calendar :deep(.fc-toolbar-chunk) {
    margin: 0 !important;
  }

  .guest-mobile-calendar :deep(.fc .fc-toolbar-title) {
    font-size: 0.75rem !important;
    font-weight: 600 !important;
    margin: 0 !important;
    line-height: 1.2 !important;
  }

  .guest-mobile-calendar :deep(.fc-button-group) {
    gap: 0.0625rem;
  }

  .guest-mobile-calendar :deep(.fc .fc-button) {
    min-width: 2rem;
    min-height: 2rem;
    padding: 0.25rem !important;
    background: transparent !important;
    border: none !important;
    font-size: 0.875rem !important;
  }

  .guest-mobile-calendar :deep(.fc-icon) {
    font-size: 1em !important;
  }

  .guest-mobile-calendar :deep(.fc-event) {
    font-size: 0.625rem !important;
    padding: 0.25rem 0.375rem !important;
    touch-action: manipulation;
  }

  .guest-mobile-calendar :deep(.fc-timegrid-slot-minor),
  .guest-mobile-calendar :deep(.fc-timegrid-slot-major) {
    border-top-width: 1px !important;
  }

  /* --- Tablet --- */
  .guest-tablet {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
    padding: 1.5rem;
    max-width: 100%;
    overflow-x: hidden;
  }

  .guest-tablet-calendar {
    display: flex;
    height: 26.25rem;
    flex-direction: column;
    overflow: hidden;
    border-radius: 0.75rem;
    border: 1px solid var(--border-subtle);
    background: var(--bg-card);
  }

  .guest-tablet-calendar :deep(.fc) {
    height: 100% !important;
    font-size: 0.875rem;
    touch-action: pan-y pinch-zoom;
  }

  .guest-tablet-calendar :deep(.fc-view-harness),
  .guest-tablet-calendar :deep(.fc-view),
  .guest-tablet-calendar :deep(.fc-timegrid),
  .guest-tablet-calendar :deep(.fc-timegrid-body),
  .guest-tablet-calendar :deep(.fc-timegrid-slots) {
    touch-action: pan-y pinch-zoom;
  }

  .guest-tablet-calendar :deep(.fc-col-header-cell) {
    padding: 0.5rem 0.25rem !important;
    height: 32px !important;
    touch-action: manipulation;
  }

  .guest-tablet-calendar :deep(.fc-col-header-cell-cushion) {
    font-size: 0.625rem !important;
    padding: 0 0.25rem !important;
    font-weight: 600 !important;
  }

  .guest-tablet-calendar :deep(.fc-timegrid-axis) {
    width: 32px !important;
    touch-action: pan-y;
  }

  .guest-tablet-calendar :deep(.fc-timegrid-slot-label) {
    font-size: 0.625rem !important;
  }

  .guest-tablet-calendar :deep(.fc-timegrid-slot) {
    min-height: 40px;
    touch-action: pan-y;
  }

  .guest-tablet-calendar :deep(.fc-timegrid-slot-lane) {
    touch-action: pan-y;
  }

  .guest-tablet-calendar :deep(.fc .fc-toolbar.fc-header-toolbar) {
    padding: 0.25rem 0.5rem !important;
    gap: 0.25rem;
    margin: 0 !important;
  }

  .guest-tablet-calendar :deep(.fc-toolbar-chunk) {
    margin: 0 !important;
  }

  .guest-tablet-calendar :deep(.fc .fc-toolbar-title) {
    font-size: 0.875rem !important;
    font-weight: 600 !important;
    margin: 0 !important;
    line-height: 1.2 !important;
  }

  .guest-tablet-calendar :deep(.fc-button-group) {
    gap: 0.125rem;
  }

  .guest-tablet-calendar :deep(.fc .fc-button) {
    min-width: 2rem;
    min-height: 2rem;
    padding: 0.25rem !important;
    background: transparent !important;
    border: none !important;
    font-size: 0.875rem !important;
  }

  .guest-tablet-calendar :deep(.fc-icon) {
    font-size: 1em !important;
  }

  /* --- Desktop --- */
  .guest-desktop {
    display: flex;
    justify-content: center;
    padding: 2rem 3rem;
  }

  .guest-desktop-inner {
    display: flex;
    width: 100%;
    max-width: 80rem;
    flex-direction: column;
    gap: 1.5rem;
  }

  .guest-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.5rem 0;
  }

  .guest-header-select {
    width: 14rem;
    flex-shrink: 0;
  }

  .guest-desktop-calendar-wrap {
    position: relative;
    height: 38.75rem;
  }

  .guest-calendar-inner {
    position: relative;
    height: 100%;
    overflow: hidden;
    border-radius: 0.5rem;
    border: 1px solid var(--border-medium);
    background: var(--bg-card);
    box-shadow: var(--shadow-card);
    transition: all 0.4s;
  }

  .guest-calendar-inner::before {
    content: '';
    position: absolute;
    inset-inline: 0;
    top: 0;
    z-index: 1;
    height: 1px;
    background: linear-gradient(
      90deg,
      transparent,
      var(--border-technical) 20%,
      var(--border-technical) 80%,
      transparent
    );
  }

  .guest-calendar-inner :deep(.fc .fc-toolbar.fc-header-toolbar) {
    margin: 0 !important;
  }
</style>
