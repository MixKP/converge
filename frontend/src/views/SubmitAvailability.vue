<script setup lang="ts">
  import { ref, computed, onMounted } from 'vue'
  import { useTeacherStore } from '../stores/teacherStore'
  import { availabilityApi } from '../services/availabilityApi'
  import PageLayout from '../components/PageLayout.vue'
  import Calendar from '../components/Calendar.vue'
  import CalendarDisabledOverlay from '../components/CalendarDisabledOverlay.vue'
  import SubmitButton from '../components/SubmitButton.vue'
  import type { EventInput } from '@fullcalendar/core'
  import { generateAvailabilityPayload } from '../utils/calendarHelpers'

  const teacherStore = useTeacherStore()

  const isLoading = ref(false)
  const events = ref<EventInput[]>([])

  const selectedTeacherId = computed({
    get: () => teacherStore.selectedTeacherId,
    set: (val) => teacherStore.setSelectedTeacherById(val),
  })

  const canSubmit = computed(
    () => !!selectedTeacherId.value && !isLoading.value && events.value.length > 0
  )

  onMounted(() => {
    teacherStore.fetchTeachers()
  })

  const handleSubmit = async () => {
    if (!selectedTeacherId.value || !events.value || isLoading.value) return
    const payload = generateAvailabilityPayload(events.value, selectedTeacherId.value)
    await availabilityApi.submitAvailability(payload)

    events.value = []
    teacherStore.setSelectedTeacherById(null)
  }
</script>

<template>
  <PageLayout title="Submit Availability">
    <form class="flex flex-col gap-3 flex-1 min-h-0" @submit.prevent="handleSubmit">
      <div class="bg-slate-50/50 p-3 rounded-lg border border-slate-100 shrink-0">
        <label class="font-semibold text-slate-700 text-sm block mb-1">
          <span class="bg-blue-100 text-blue-700 px-2 py-0.5 rounded text-xs mr-1">1</span>
          Teacher
        </label>
        <select
          v-model="selectedTeacherId"
          class="w-full md:w-1/2 bg-white border border-slate-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 outline-none transition-all text-slate-700 shadow-sm cursor-pointer"
        >
          <option :value="null" disabled>-- Choose Your Name --</option>
          <option v-for="teacher in teacherStore.teachers" :key="teacher.id" :value="teacher.id">
            {{ teacher.name }}
          </option>
        </select>
      </div>

      <div class="flex-1 flex flex-col min-h-0">
        <div class="flex items-center justify-between mb-2 shrink-0">
          <label class="font-semibold text-slate-700 text-sm">
            <span class="bg-blue-100 text-blue-700 px-2 py-0.5 rounded text-xs mr-1">2</span>
            Time Slots
          </label>
          <span
            v-if="selectedTeacherId"
            class="text-xs font-medium text-blue-600 bg-blue-50 px-2 py-0.5 rounded border border-blue-100"
          >
            Click and drag to select
          </span>
        </div>

        <div
          class="relative border border-slate-200 rounded-lg overflow-hidden shadow-sm bg-white flex-1 min-h-0"
        >
          <div
            :class="{
              'opacity-40 grayscale-30% pointer-events-none transition-opacity duration-300':
                !selectedTeacherId,
            }"
            class="h-full"
          >
            <Calendar v-model="events" :editable="!!selectedTeacherId" class="h-full" />
          </div>

          <CalendarDisabledOverlay v-if="!selectedTeacherId" message="Select a teacher first" />
        </div>
      </div>

      <div class="flex justify-end pt-2 border-t border-slate-100 shrink-0">
        <SubmitButton
          :is-disabled="!canSubmit"
          :is-loading="isLoading"
          loading-text="Submitting..."
          normal-text="Submit Availability"
        />
      </div>
    </form>
  </PageLayout>
</template>
