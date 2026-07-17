<script setup lang="ts">
  import PageLayout from '../components/PageLayout.vue'
  import Calendar from '../components/Calendar.vue'
  import CalendarDisabledOverlay from '../components/CalendarDisabledOverlay.vue'
  import { useSubmitAvailability } from '../composables/useSubmitAvailability'

  const {
    teacherStore,
    isLoading,
    errorMessage,
    successMessage,
    events,
    showConfirm,
    selectedTeacherId,
    selectedTeacher,
    canSubmit,
    formattedSlots,
    handleSubmit,
    confirmSubmit,
    cancelConfirm,
  } = useSubmitAvailability()
</script>

<template>
  <PageLayout title="Submit Availability" :show-cart="false">
    <div class="submit-page">
      <!-- Unified Card Container -->
      <div class="submit-card">
        <!-- Toolbar -->
        <div class="submit-toolbar">
          <!-- Teacher Selector -->
          <div class="toolbar-section toolbar-section--teacher">
            <select v-model="selectedTeacherId" class="teacher-select" aria-label="Select teacher">
              <option :value="null">Select Teacher</option>
              <option
                v-for="teacher in teacherStore.teachers"
                :key="teacher.id"
                :value="teacher.id"
              >
                {{ teacher.name }}
              </option>
            </select>
          </div>

          <!-- Submit Button -->
          <div class="toolbar-section toolbar-section--action">
            <button
              type="submit"
              :disabled="!canSubmit"
              :aria-busy="isLoading"
              class="submit-button"
              :class="{ 'submit-button--loading': isLoading }"
              @click="handleSubmit"
            >
              <span class="button-content">
                <svg v-if="isLoading" class="spinner" fill="none" viewBox="0 0 24 24">
                  <circle
                    class="spinner-track"
                    cx="12"
                    cy="12"
                    r="10"
                    stroke="currentColor"
                    stroke-width="4"
                  />
                  <path
                    class="spinner-fill"
                    fill="currentColor"
                    d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                  />
                </svg>
                <span class="button-text">{{ isLoading ? '...' : 'Submit' }}</span>
              </span>
              <span class="button-shine"></span>
            </button>
          </div>
        </div>

        <!-- Calendar -->
        <div class="submit-calendar-container">
          <Calendar
            v-if="selectedTeacherId"
            v-model="events"
            :editable="true"
            class="submit-calendar"
          />
          <CalendarDisabledOverlay v-else message="Select a teacher first" />
        </div>
      </div>

      <!-- Confirm Dialog -->
      <div
        v-if="showConfirm"
        class="submit-confirm-overlay"
        role="dialog"
        aria-modal="true"
        aria-labelledby="confirm-title"
        @click.self="cancelConfirm"
      >
        <div class="submit-confirm-dialog">
          <h3 id="confirm-title" class="submit-confirm-title">Confirm Availability</h3>
          <div class="submit-confirm-content">
            <div class="submit-confirm-row">
              <span class="submit-confirm-label">TEACHER</span>
              <span class="submit-confirm-value">{{ selectedTeacher?.name }}</span>
            </div>
            <div class="submit-confirm-row">
              <span class="submit-confirm-label">TIME SLOTS</span>
              <span class="submit-confirm-value"
                >{{ events.length }} slot{{ events.length > 1 ? 's' : '' }}</span
              >
            </div>
            <div class="submit-confirm-slots">
              <div v-for="(slot, index) in formattedSlots" :key="index" class="submit-confirm-slot">
                {{ slot.day }} {{ slot.start }}-{{ slot.end }}
              </div>
            </div>
          </div>
          <div class="submit-confirm-actions">
            <button
              type="button"
              class="submit-confirm-btn submit-confirm-btn--cancel"
              @click="cancelConfirm"
            >
              Cancel
            </button>
            <button
              type="button"
              class="submit-confirm-btn submit-confirm-btn--confirm"
              @click="confirmSubmit"
            >
              Confirm
            </button>
          </div>
        </div>
      </div>

      <!-- Toast Notifications -->
      <div
        v-if="successMessage"
        class="submit-toast submit-toast--success"
        role="status"
        aria-live="polite"
      >
        <svg
          class="submit-toast-icon"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          aria-hidden="true"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
          />
        </svg>
        {{ successMessage }}
      </div>

      <div
        v-if="errorMessage"
        class="submit-toast submit-toast--error"
        role="alert"
        aria-live="assertive"
      >
        <svg
          class="submit-toast-icon"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          aria-hidden="true"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z"
          />
        </svg>
        {{ errorMessage }}
        <button class="submit-toast-close" aria-label="Dismiss message" @click="errorMessage = ''">
          ×
        </button>
      </div>
    </div>
  </PageLayout>
</template>

<style src="../css/SubmitAvailability.css"></style>
