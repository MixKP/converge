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
    (e: 'reset'): void
  }>()

  const getScoreColor = (score: number): string => {
    if (score >= 80) return 'text-green-600'
    if (score >= 60) return 'text-yellow-600'
    return 'text-red-600'
  }

  const formatTime = (dateStr: string): string => {
    const date = new Date(dateStr)
    return date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: false })
  }
</script>

<template>
  <div class="space-y-5 flex-1 overflow-y-auto">
    <div v-if="!showDetailedResults && !isEvaluating" class="space-y-4">
      <p class="text-sm text-(--text-secondary)">
        Choose your subject, branch, and when you'd like to learn. We'll find available teachers for
        your preferred times.
      </p>
    </div>

    <div v-if="isEvaluating" class="flex items-center justify-center py-12">
      <div class="loading-state">
        <div class="loading-spinner">
          <div class="spinner-ring"></div>
          <div class="spinner-ring spinner-ring--delay"></div>
          <div class="spinner-ring spinner-ring--delay-2"></div>
        </div>
        <p class="loading-text">Finding available teachers...</p>
      </div>
    </div>

    <div v-if="showDetailedResults && suggestions" class="space-y-4">
      <div
        class="bg-(--paper-cream) p-4 rounded-lg border border-(--border-subtle) results-summary"
      >
        <h3 class="font-semibold text-(--ink-primary) mb-2">Booking Options Summary</h3>
        <p class="text-sm text-(--text-secondary)">
          Found {{ suggestions.results.length }} time slot(s) with available teachers.
        </p>
      </div>

      <div
        v-for="(slotResult, index) in suggestions.results"
        :key="index"
        class="bg-white border border-(--border-subtle) rounded-lg p-4 space-y-3 result-card"
        :style="{ '--stagger-index': index }"
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
          <div class="flex-1">
            <div class="flex items-center gap-2">
              <span class="text-white">⭐</span>
              <span class="font-semibold text-white">{{
                slotResult.exact_match.teacher_name
              }}</span>
            </div>
            <p class="text-sm text-white/80">Score: {{ slotResult.exact_match.score }}</p>
            <p
              v-if="slotResult.exact_match.reasons && slotResult.exact_match.reasons.length > 0"
              class="text-xs text-white/70 mt-1"
            >
              {{ slotResult.exact_match.reasons.join(' • ') }}
            </p>
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
            <div class="flex-1">
              <div class="flex items-center gap-2">
                <span>💡</span>
                <span class="font-medium text-(--ink-primary)">{{ alt.teacher_name }}</span>
                <span
                  v-if="alt.room_available !== undefined"
                  class="text-xs px-2 py-0.5 rounded-full"
                  :class="
                    alt.room_available ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'
                  "
                >
                  {{ alt.room_available ? 'Room' : 'No Room' }}
                </span>
                <span
                  v-if="alt.commute_minutes !== undefined"
                  class="text-xs px-2 py-0.5 bg-blue-100 text-blue-700 rounded-full"
                >
                  {{ alt.commute_minutes }}m commute
                </span>
              </div>
              <p class="text-sm" :class="getScoreColor(alt.score)">Score: {{ alt.score }}</p>
              <p class="text-xs text-(--text-secondary)">
                {{ formatTime(alt.start_time) }} - {{ formatTime(alt.end_time) }}
              </p>
              <p
                v-if="alt.reasons && alt.reasons.length > 0"
                class="text-xs text-(--text-secondary) mt-1"
              >
                {{ alt.reasons.join(' • ') }}
              </p>
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
        <p>No teachers available for these time slots.</p>
        <p class="text-sm mt-2">Try adjusting your preferred days or times and search again.</p>
      </div>
    </div>

    <div v-if="showDetailedResults" class="flex gap-3 pt-4 border-t border-(--border-subtle)">
      <button
        type="button"
        class="flex-1 px-4 py-2 text-(--ink-primary) bg-white border border-(--border-strong) rounded-lg hover:bg-(--paper-cream) transition-all"
        @click="emit('reset')"
      >
        Search Again
      </button>
    </div>
  </div>
</template>

<style scoped>
  .loading-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1rem;
  }

  .loading-spinner {
    position: relative;
    width: 48px;
    height: 48px;
  }

  .spinner-ring {
    position: absolute;
    inset: 0;
    border: 2px solid transparent;
    border-top-color: var(--primary-indigo);
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }

  .spinner-ring--delay {
    inset: 4px;
    border-top-color: var(--accent-sage);
    animation: spin 1.2s linear infinite reverse;
  }

  .spinner-ring--delay-2 {
    inset: 8px;
    border-top-color: var(--text-muted);
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  .loading-text {
    font-size: 0.875rem;
    color: var(--text-secondary);
    font-family: 'IBM Plex Sans', sans-serif;
    animation: pulse 2s ease-in-out infinite;
  }

  @keyframes pulse {
    0%,
    100% {
      opacity: 1;
    }
    50% {
      opacity: 0.6;
    }
  }

  .empty-state {
    animation: fade-in 0.4s ease-out;
  }

  @keyframes fade-in {
    from {
      opacity: 0;
      transform: translateY(8px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .results-summary {
    animation: fade-in 0.4s ease-out;
  }

  .result-card {
    animation: result-card-enter 0.5s cubic-bezier(0.25, 1, 0.5, 1) backwards;
    animation-delay: calc(var(--stagger-index, 0) * 80ms + 0.2s);
  }

  @keyframes result-card-enter {
    from {
      opacity: 0;
      transform: translateY(16px) scale(0.98);
    }
    to {
      opacity: 1;
      transform: translateY(0) scale(1);
    }
  }
</style>
