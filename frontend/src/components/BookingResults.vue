<script setup lang="ts">
  import { ref, watch, computed } from 'vue'
  import type { BookingResponse } from '../types'
  import type { CartItem } from '../types/booking'
  import { rangesOverlap } from '../utils/dateValidation'

  interface Props {
    suggestions: BookingResponse | null
    showDetailedResults: boolean
    isEvaluating: boolean
    cartItems: CartItem[]
  }

  const props = defineProps<Props>()

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

  const bookedKeys = ref<Set<string>>(new Set())

  const matchedCount = computed(() => {
    if (!props.suggestions) return 0
    return props.suggestions.results.filter(
      (r) => r.exact_match || (r.alternatives && r.alternatives.length > 0)
    ).length
  })

  const getBookingKey = (teacherId: number, startTime: string): string => {
    return `${teacherId}-${startTime}`
  }

  const isInCart = (teacherId: number, startTime: string, endTime: string): boolean => {
    const newStart = new Date(startTime)
    const newEnd = new Date(endTime)

    return props.cartItems.some((item) => {
      if (item.teacher_id !== teacherId) return false

      const itemStart = new Date(item.start_time)
      const itemEnd = new Date(item.end_time)

      return rangesOverlap(newStart, newEnd, itemStart, itemEnd)
    })
  }

  const isBooked = (teacherId: number, startTime: string, endTime: string): boolean => {
    return (
      bookedKeys.value.has(getBookingKey(teacherId, startTime)) ||
      isInCart(teacherId, startTime, endTime)
    )
  }

  const handleBooking = (
    teacherId: number,
    teacherName: string,
    startTime: string,
    endTime: string
  ): void => {
    const key = getBookingKey(teacherId, startTime)
    if (!bookedKeys.value.has(key)) {
      bookedKeys.value.add(key)
      emit('confirmBooking', teacherId, teacherName, startTime, endTime)
    }
  }

  watch(
    () => props.suggestions,
    () => {
      bookedKeys.value.clear()
    }
  )

  const getScoreColor = (score: number): string => {
    if (score >= 90) return 'score-excellent' // 90-100: Green
    if (score >= 75) return 'score-high' // 75-89: Light green
    if (score >= 60) return 'score-medium' // 60-74: Yellow
    if (score >= 45) return 'score-low' // 45-59: Orange
    return 'score-poor' // 0-44: Red
  }

  const formatTime = (dateStr: string): string => {
    const date = new Date(dateStr)
    const hours = date.getHours()
    const minutes = date.getMinutes()
    const ampm = hours >= 12 ? 'pm' : 'am'
    const displayHours = hours === 0 ? 12 : hours > 12 ? hours - 12 : hours
    const displayMinutes = minutes > 0 ? `.${String(minutes).padStart(2, '0')}` : ''
    return `${displayHours}${displayMinutes}${ampm}`
  }

  const formatDate = (dateStr: string): string => {
    const date = new Date(dateStr)
    const dayName = date.toLocaleDateString('en-US', { weekday: 'long' })
    const timeStr = formatTime(dateStr)
    return `${dayName} ${timeStr}`
  }

  const isNextWeek = (dateStr: string): boolean => {
    const date = new Date(dateStr)
    const now = new Date()
    const oneWeekFromNow = new Date(now)
    oneWeekFromNow.setDate(now.getDate() + 7)
    return date >= oneWeekFromNow
  }

  const DAY_NAMES = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'] as const

  const getDayName = (dayOfWeek: number): string => {
    return DAY_NAMES[dayOfWeek] || 'Unknown'
  }
</script>

<template>
  <div class="results-container">
    <div v-if="isEvaluating" class="results-loading">
      <div class="loading-spinner">
        <div class="spinner-ring"></div>
        <div class="spinner-ring spinner-ring--delay"></div>
        <div class="spinner-ring spinner-ring--delay-2"></div>
      </div>
      <p class="loading-text">Finding available teachers...</p>
    </div>

    <div v-if="showDetailedResults && suggestions" class="results-list">
      <div class="results-summary">
        <h3 class="summary-title">Booking Options Summary</h3>
        <p class="summary-text">
          <template v-if="matchedCount > 0">
            Found {{ matchedCount }} time slot{{ matchedCount !== 1 ? 's' : '' }} with available
            teachers.
          </template>
          <template v-else> No teachers available for the selected time slots. </template>
        </p>
      </div>

      <div
        v-for="(slotResult, index) in suggestions.results"
        :key="index"
        class="result-card"
        :style="{ '--stagger-index': index }"
      >
        <div class="result-header">
          <div v-if="slotResult.exact_match">
            <h4 class="result-title">
              {{ formatDate(slotResult.exact_match.start_time) }} -
              {{ formatTime(slotResult.exact_match.end_time) }}
              <template v-if="isNextWeek(slotResult.exact_match.start_time)">
                (next week)
              </template>
            </h4>
          </div>
          <div v-else>
            <h4 class="result-title">
              {{ getDayName(slotResult.slot.day_of_week) }} {{ slotResult.slot.start }} -
              {{ slotResult.slot.end }}
            </h4>
            <p class="result-message">{{ slotResult.message }}</p>
          </div>
        </div>

        <div
          v-if="slotResult.exact_match"
          class="match-exact"
          :class="getScoreColor(slotResult.exact_match.score)"
        >
          <div class="match-content">
            <div class="match-header">
              <span class="match-name">{{ slotResult.exact_match.teacher_name }}</span>
            </div>
            <p class="match-score">Score: {{ slotResult.exact_match.score }}</p>
            <p class="match-reasons">{{ slotResult.message }}</p>
            <p
              v-if="slotResult.exact_match.reasons && slotResult.exact_match.reasons.length > 0"
              class="match-reasons"
            >
              {{ slotResult.exact_match.reasons.join(' • ') }}
            </p>
          </div>
          <button
            type="button"
            class="match-button"
            :class="{
              'match-button--booked': isBooked(
                slotResult.exact_match.teacher_id,
                slotResult.exact_match.start_time,
                slotResult.exact_match.end_time
              ),
            }"
            :disabled="
              isBooked(
                slotResult.exact_match.teacher_id,
                slotResult.exact_match.start_time,
                slotResult.exact_match.end_time
              )
            "
            @click="
              handleBooking(
                slotResult.exact_match.teacher_id,
                slotResult.exact_match.teacher_name,
                slotResult.exact_match.start_time,
                slotResult.exact_match.end_time
              )
            "
          >
            {{
              isBooked(
                slotResult.exact_match.teacher_id,
                slotResult.exact_match.start_time,
                slotResult.exact_match.end_time
              )
                ? 'Booked'
                : 'Book Now'
            }}
          </button>
        </div>

        <div
          v-if="slotResult.alternatives && slotResult.alternatives.length > 0"
          class="alternatives"
        >
          <p class="alternatives-title">Other Options</p>
          <div
            v-for="(alt, altIndex) in slotResult.alternatives"
            :key="altIndex"
            class="alternative-item"
          >
            <div class="alternative-content">
              <div class="alternative-header">
                <span class="alternative-name">{{ alt.teacher_name }}</span>
                <span
                  v-if="alt.room_available !== undefined"
                  class="badge"
                  :class="alt.room_available ? 'badge-success' : 'badge-error'"
                >
                  {{ alt.room_available ? 'Room' : 'No Room' }}
                </span>
                <span v-if="alt.commute_minutes !== undefined" class="badge badge-info">
                  {{ alt.commute_minutes }}m commute
                </span>
              </div>
              <p class="alternative-score" :class="getScoreColor(alt.score)">
                Score: {{ alt.score }}
              </p>
              <p class="alternative-time">
                {{ formatDate(alt.start_time) }} - {{ formatTime(alt.end_time) }}
                <template v-if="isNextWeek(alt.start_time)"> (next week)</template>
              </p>
              <p v-if="alt.reasons && alt.reasons.length > 0" class="alternative-reasons">
                {{ alt.reasons.join(' • ') }}
              </p>
            </div>
            <button
              type="button"
              class="alternative-button"
              :class="{
                'alternative-button--booked': isBooked(
                  alt.teacher_id,
                  alt.start_time,
                  alt.end_time
                ),
              }"
              :disabled="isBooked(alt.teacher_id, alt.start_time, alt.end_time)"
              @click="handleBooking(alt.teacher_id, alt.teacher_name, alt.start_time, alt.end_time)"
            >
              {{ isBooked(alt.teacher_id, alt.start_time, alt.end_time) ? 'Booked' : 'Book Now' }}
            </button>
          </div>
        </div>
      </div>

      <div v-if="!suggestions || suggestions.results.length === 0" class="empty-state">
        <svg class="empty-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
          />
        </svg>
        <p class="empty-title">No teachers available</p>
        <p class="empty-text">Try adjusting your preferred days or times and search again.</p>
      </div>
    </div>

    <div v-if="showDetailedResults" class="results-actions">
      <button type="button" class="action-button action-button--secondary" @click="emit('reset')">
        Search Again
      </button>
    </div>

    <div v-if="!isEvaluating && !showDetailedResults" class="results-idle">
      <svg class="idle-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="1.5"
          d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09zM18.259 8.715L18 9.75l-.259-1.035a3.375 3.375 0 00-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 002.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 002.456 2.456L21.75 6l-1.035.259a3.375 3.375 0 00-2.456 2.456z"
        />
      </svg>
      <p class="idle-title">No results yet</p>
      <p class="idle-text">Fill out the form and search to see matching teachers here.</p>
    </div>
  </div>
</template>

<style scoped>
  .results-container {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
    flex: 1;
    overflow-y: auto;
  }

  .results-intro p {
    font-size: 0.875rem;
    color: var(--text-secondary);
    line-height: 1.5;
    margin: 0;
  }

  .results-loading {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 3rem 0;
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
    font-family: 'Inter', sans-serif;
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

  .results-summary {
    padding: 1rem;
    background: var(--bg-subtle);
    border-radius: 8px;
    border: 1px solid var(--border-subtle);
    animation: fade-in 0.4s ease-out;
  }

  .summary-title {
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--text-primary);
    margin: 0 0 0.5rem 0;
    font-family: 'Inter', sans-serif;
  }

  .summary-text {
    font-size: 0.8125rem;
    color: var(--text-secondary);
    margin: 0;
  }

  .results-list {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .result-card {
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: 8px;
    padding: 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    animation: result-card-enter 0.5s cubic-bezier(0.25, 1, 0.5, 1) backwards;
    animation-delay: calc(var(--stagger-index, 0) * 80ms + 0.2s);
  }

  /* Result card background colors based on score */

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

  .result-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
  }

  .result-title {
    font-size: 0.9375rem;
    font-weight: 600;
    color: var(--text-primary);
    margin: 0;
    font-family: 'Inter', sans-serif;
  }

  .result-message {
    font-size: 0.8125rem;
    color: var(--text-secondary);
    margin: 0.25rem 0 0 0;
  }

  .match-exact {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
    padding: 0.75rem;
    background: var(--bg-subtle);
    border-radius: 6px;
    border: 2px solid var(--accent-sage);
  }

  .match-content {
    flex: 1;
  }

  .match-header {
    display: flex;
    align-items: center;
    width: 100%;
  }

  .match-name {
    font-weight: 600;
    color: var(--text-primary);
    font-size: 0.95rem;
  }

  .match-score {
    font-size: 0.8rem;
    font-weight: 500;
    margin: 0.15rem 0 0 0;
    font-family: 'JetBrains Mono', monospace;
    color: var(--accent-sage);
  }

  /* Score colors for exact matches: green → yellow → orange → red */
  .match-score.score-excellent {
    color: #22c55e; /* Green for 90-100 */
  }

  .match-score.score-high {
    color: #84cc16; /* Light green for 75-89 */
  }

  .match-score.score-medium {
    color: #eab308; /* Yellow for 60-74 */
  }

  .match-score.score-low {
    color: #f97316; /* Orange for 45-59 */
  }

  .match-score.score-poor {
    color: #ef4444; /* Red for 0-44 */
  }

  .match-reasons {
    font-size: 0.75rem;
    margin: 0.15rem 0 0 0;
    font-weight: 500;
    color: var(--text-secondary);
  }

  .match-button {
    padding: 0.5rem 1rem;
    background: var(--accent-sage);
    color: white;
    border: none;
    border-radius: 6px;
    font-size: 0.8125rem;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s cubic-bezier(0.25, 1, 0.5, 1);
    font-family: 'Inter', sans-serif;
    white-space: nowrap;
  }

  .match-button:hover {
    background: var(--primary-navy);
  }

  .match-button:focus-visible {
    outline: none;
    box-shadow: 0 0 0 2px rgba(157, 180, 160, 0.4);
  }

  .match-button--booked,
  .match-button:disabled {
    background: var(--border-medium);
    color: var(--text-muted);
    cursor: not-allowed;
    opacity: 0.7;
  }

  .match-button--booked:hover,
  .match-button:disabled:hover {
    background: var(--border-medium);
  }

  .alternatives {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .alternatives-title {
    font-size: 0.75rem;
    font-weight: 500;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin: 0;
    font-family: 'JetBrains Mono', monospace;
  }

  .alternative-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
    padding: 0.75rem;
    background: var(--bg-subtle);
    border-radius: 6px;
  }

  .alternative-content {
    flex: 1;
  }

  .alternative-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex-wrap: wrap;
  }

  .alternative-bulb {
    font-size: 0.875rem;
  }

  .alternative-name {
    font-weight: 500;
    color: var(--text-primary);
    font-size: 0.875rem;
  }

  .badge {
    font-size: 0.75rem;
    padding: 0.125rem 0.5rem;
    border-radius: 999px;
    font-family: 'JetBrains Mono', monospace;
  }

  .badge-success {
    background: var(--accent-sage);
    color: white;
  }

  .badge-error {
    background: var(--accent-coral);
    color: white;
  }

  .badge-info {
    background: var(--primary-indigo);
    color: white;
  }

  .alternative-score {
    font-size: 0.8125rem;
    margin: 0.25rem 0 0 0;
  }

  /* Score colors: green → yellow → orange → red */
  .alternative-score.score-excellent {
    color: #22c55e; /* Green for 90-100 */
  }

  .alternative-score.score-high {
    color: #84cc16; /* Light green for 75-89 */
  }

  .alternative-score.score-medium {
    color: #eab308; /* Yellow for 60-74 */
  }

  .alternative-score.score-low {
    color: #f97316; /* Orange for 45-59 */
  }

  .alternative-score.score-poor {
    color: #ef4444; /* Red for 0-44 */
  }

  .alternative-time {
    font-size: 0.75rem;
    color: var(--text-secondary);
    margin: 0;
    font-family: 'JetBrains Mono', monospace;
  }

  .alternative-reasons {
    font-size: 0.75rem;
    color: var(--text-secondary);
    margin: 0.25rem 0 0 0;
  }

  .alternative-button {
    padding: 0.5rem 1rem;
    background: var(--accent-coral);
    color: white;
    border: none;
    border-radius: 6px;
    font-size: 0.8125rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s cubic-bezier(0.25, 1, 0.5, 1);
    font-family: 'Inter', sans-serif;
    white-space: nowrap;
  }

  .alternative-button:hover {
    filter: brightness(0.9);
  }

  .alternative-button:focus-visible {
    outline: none;
    box-shadow: 0 0 0 2px rgba(201, 109, 93, 0.4);
  }

  .alternative-button--booked,
  .alternative-button:disabled {
    background: var(--border-medium);
    color: var(--text-muted);
    cursor: not-allowed;
    opacity: 0.7;
  }

  .alternative-button--booked:hover,
  .alternative-button:disabled:hover {
    background: var(--border-medium);
    filter: none;
  }

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    padding: 3rem 1rem;
    animation: fade-in 0.4s ease-out;
  }

  .empty-icon {
    width: 4rem;
    height: 4rem;
    color: var(--border-strong);
    margin: 0 0 1rem 0;
  }

  .empty-title {
    font-size: 0.9375rem;
    font-weight: 500;
    color: var(--text-primary);
    margin: 0 0 0.5rem 0;
  }

  .empty-text {
    font-size: 0.8125rem;
    color: var(--text-secondary);
    margin: 0;
  }

  .results-idle {
    display: flex;
    flex: 1;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    text-align: center;
    padding: 3rem 1rem;
    min-height: 200px;
    animation: fade-in 0.4s ease-out;
  }

  .idle-icon {
    width: 2.5rem;
    height: 2.5rem;
    color: var(--border-strong);
    margin: 0 0 1rem 0;
  }

  .idle-title {
    font-size: 0.9375rem;
    font-weight: 500;
    color: var(--text-primary);
    margin: 0 0 0.5rem 0;
  }

  .idle-text {
    font-size: 0.8125rem;
    color: var(--text-secondary);
    margin: 0;
    max-width: 22rem;
  }

  .results-actions {
    display: flex;
    gap: 0.75rem;
    padding-top: 1rem;
    border-top: 1px solid var(--border-subtle);
  }

  .action-button {
    flex: 1;
    padding: 0.5rem 1rem;
    font-size: 0.875rem;
    font-weight: 500;
    border-radius: 8px;
    cursor: pointer;
    transition: all 0.2s cubic-bezier(0.25, 1, 0.5, 1);
    font-family: 'Inter', sans-serif;
  }

  .action-button--secondary {
    background: var(--bg-card);
    color: var(--text-primary);
    border: 1px solid var(--border-strong);
  }

  .action-button--secondary:hover {
    background: var(--bg-cream);
  }

  .action-button--secondary:focus-visible {
    outline: none;
    box-shadow: 0 0 0 3px rgba(62, 76, 122, 0.2);
  }

  @media (max-width: 767px) {
    .match-exact,
    .alternative-item {
      flex-direction: column;
      align-items: stretch;
    }

    .match-button,
    .alternative-button {
      width: 100%;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    *,
    *::before,
    *::after {
      animation-duration: 0.01ms !important;
      animation-iteration-count: 1 !important;
      transition-duration: 0.01ms !important;
    }
  }
</style>
