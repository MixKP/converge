<script setup lang="ts">
  import type { BookingResponse } from '../types'

  interface Props {
    suggestions: BookingResponse | null
    showDetailedResults: boolean
    isEvaluating: boolean
  }

  defineProps<Props>()

  const emit = defineEmits<{
    (
      e: 'confirmBooking',
      teacherId: number,
      teacherName: string,
      startTime: string,
      endTime: string
    ): void
    (e: 'goBack'): void
    (e: 'reset'): void
  }>()

  const getScoreColor = (score: number): string => {
    if (score >= 80) return 'text-green-600'
    if (score >= 60) return 'text-yellow-600'
    return 'text-red-600'
  }
</script>

<template>
  <div class="space-y-5 flex-1 overflow-y-auto">
    <div v-if="!showDetailedResults && !isEvaluating" class="space-y-4">
      <p class="text-sm text-(--text-secondary)">
        Select your subject, branch, and preferred time slots, then click "Evaluate Options" to see
        available booking options.
      </p>
    </div>

    <div v-if="isEvaluating" class="flex items-center justify-center py-12">
      <div
        class="w-8 h-8 border-4 border-(--accent-terracotta) border-t-transparent rounded-full animate-spin"
      ></div>
    </div>

    <div v-if="showDetailedResults && suggestions" class="space-y-4">
      <div class="bg-(--paper-cream) p-4 rounded-lg border border-(--border-subtle)">
        <h3 class="font-semibold text-(--ink-primary) mb-2">Booking Options Summary</h3>
        <p class="text-sm text-(--text-secondary)">
          Found {{ suggestions.results.length }} time slot(s) with available teachers.
        </p>
      </div>

      <div
        v-for="(slotResult, index) in suggestions.results"
        :key="index"
        class="bg-white border border-(--border-subtle) rounded-lg p-4 space-y-3"
      >
        <div class="flex items-center justify-between">
          <div>
            <h4 class="font-semibold text-(--ink-primary)">
              {{
                slotResult.slot.day_of_week === 0
                  ? 'Sun'
                  : slotResult.slot.day_of_week === 1
                    ? 'Mon'
                    : slotResult.slot.day_of_week === 2
                      ? 'Tue'
                      : slotResult.slot.day_of_week === 3
                        ? 'Wed'
                        : slotResult.slot.day_of_week === 4
                          ? 'Thu'
                          : slotResult.slot.day_of_week === 5
                            ? 'Fri'
                            : 'Sat'
              }}
              {{ slotResult.slot.start }} - {{ slotResult.slot.end }}
            </h4>
            <p class="text-sm text-(--text-secondary)">{{ slotResult.message }}</p>
          </div>
        </div>

        <div
          v-if="slotResult.exact_match"
          class="bg-(--accent-terracotta) p-3 rounded-md flex items-center justify-between"
        >
          <div>
            <div class="flex items-center gap-2">
              <span class="text-white">⭐</span>
              <span class="font-semibold text-white">{{
                slotResult.exact_match.teacher_name
              }}</span>
            </div>
            <p class="text-sm text-white/80">Score: {{ slotResult.exact_match.score }}</p>
          </div>
          <button
            type="button"
            class="px-4 py-2 bg-white text-(--accent-terracotta) rounded-md font-semibold hover:bg-(--paper-cream) transition-all"
            @click="
              emit(
                'confirmBooking',
                slotResult.exact_match!.teacher_id,
                slotResult.exact_match!.teacher_name,
                slotResult.exact_match!.start_time,
                slotResult.exact_match!.end_time
              )
            "
          >
            Book Now
          </button>
        </div>

        <div v-if="slotResult.alternatives && slotResult.alternatives.length > 0" class="space-y-2">
          <p class="text-xs font-semibold text-(--text-secondary) uppercase">Other Options</p>
          <div
            v-for="(alt, altIndex) in slotResult.alternatives"
            :key="altIndex"
            class="flex items-center justify-between p-3 bg-(--accent-terracotta-soft) rounded-md"
          >
            <div>
              <div class="flex items-center gap-2">
                <span>💡</span>
                <span class="font-medium text-(--ink-primary)">{{ alt.teacher_name }}</span>
              </div>
              <p class="text-sm" :class="getScoreColor(alt.score)">Score: {{ alt.score }}</p>
            </div>
            <button
              type="button"
              class="px-4 py-2 bg-(--accent-terracotta) text-white rounded-md font-semibold hover:bg-(--accent-terracotta-dark) transition-all"
              @click="
                emit(
                  'confirmBooking',
                  alt.teacher_id,
                  alt.teacher_name,
                  alt.start_time,
                  alt.end_time
                )
              "
            >
              Book Now
            </button>
          </div>
        </div>
      </div>

      <div
        v-if="!suggestions || suggestions.results.length === 0"
        class="text-center py-12 text-(--text-secondary)"
      >
        <svg
          class="w-16 h-16 mx-auto mb-4 text-(--border-strong)"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
          />
        </svg>
        <p>No booking options found for your selected time slots.</p>
        <p class="text-sm mt-2">Try different time slots or check back later.</p>
      </div>
    </div>

    <div v-if="showDetailedResults" class="flex gap-3 pt-4 border-t border-(--border-subtle)">
      <button
        type="button"
        class="flex-1 px-4 py-2 text-(--ink-primary) bg-white border border-(--border-strong) rounded-lg hover:bg-(--paper-cream) transition-all"
        @click="emit('goBack')"
      >
        Back to Details
      </button>
      <button
        type="button"
        class="flex-1 px-4 py-2 text-(--ink-primary) bg-white border border-(--border-strong) rounded-lg hover:bg-(--paper-cream) transition-all"
        @click="emit('reset')"
      >
        Start Over
      </button>
    </div>
  </div>
</template>
