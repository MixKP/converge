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
    <div
      class="flex justify-center p-8 max-md:p-4 max-[480px]:p-3 h-[calc(100dvh-60px)] overflow-hidden"
    >
      <!-- Unified Card Container -->
      <div
        class="flex flex-col w-full max-w-[1400px] h-full bg-(--bg-card) border border-(--border-subtle) rounded-[20px] max-md:rounded-2xl overflow-hidden shadow-[0_8px_32px_rgba(45,74,62,0.08),0_2px_8px_rgba(45,74,62,0.04)]"
      >
        <!-- Toolbar -->
        <div
          class="flex items-center gap-6 px-8 py-5 max-md:justify-between max-md:gap-2 max-md:px-4 max-md:py-3 border-b border-(--border-subtle) shrink-0 bg-[linear-gradient(180deg,rgba(157,180,160,0.04)_0%,transparent_100%)]"
        >
          <!-- Teacher Selector -->
          <div class="flex items-center flex-1 min-w-0 max-md:flex-auto">
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
          <div class="flex items-center shrink-0 max-md:flex-none">
            <button
              type="submit"
              :disabled="!canSubmit"
              :aria-busy="isLoading"
              class="group relative flex items-center justify-center px-6 py-3.5 max-[480px]:px-3 max-[480px]:py-2 font-[Inter,sans-serif] text-sm max-[480px]:text-xs font-medium text-white bg-(--primary-indigo) rounded-md cursor-pointer overflow-hidden transition-all duration-300 ease-[cubic-bezier(0.34,1.56,0.64,1)] shadow-[0_2px_8px_rgba(62,76,122,0.2),inset_0_-1px_0_rgba(0,0,0,0.1)] enabled:hover:bg-(--primary-indigo-deep) enabled:hover:-translate-y-0.5 enabled:hover:shadow-[0_4px_16px_rgba(62,76,122,0.3),inset_0_-1px_0_rgba(0,0,0,0.1)] enabled:active:translate-y-0 enabled:active:shadow-[0_2px_8px_rgba(62,76,122,0.2),inset_0_1px_0_rgba(0,0,0,0.1)] disabled:opacity-40 disabled:cursor-not-allowed disabled:translate-y-0 disabled:shadow-none focus-visible:outline-none focus-visible:shadow-[0_0_0_3px_rgba(62,76,122,0.3),0_2px_8px_rgba(62,76,122,0.2)]"
              :class="{ 'pointer-events-none': isLoading }"
              @click="handleSubmit"
            >
              <span class="flex items-center gap-2 relative z-[1]">
                <svg v-if="isLoading" class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle
                    class="opacity-25"
                    cx="12"
                    cy="12"
                    r="10"
                    stroke="currentColor"
                    stroke-width="4"
                  />
                  <path
                    class="opacity-75"
                    fill="currentColor"
                    d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                  />
                </svg>
                <span class="transition-opacity duration-200">{{
                  isLoading ? '...' : 'Submit'
                }}</span>
              </span>
              <span
                class="absolute top-0 -left-full w-full h-full bg-[linear-gradient(90deg,transparent,rgba(255,255,255,0.1),transparent)] transition-[left] duration-[600ms] ease-[ease] group-[:hover:not(:disabled)]:left-full"
              ></span>
            </button>
          </div>
        </div>

        <!-- Calendar -->
        <div
          class="relative flex flex-col flex-1 min-h-0 p-0 md:px-4 md:py-2 lg:px-6 overflow-hidden max-md:overflow-y-auto bg-[linear-gradient(135deg,rgba(157,180,160,0.03)_0%,rgba(168,201,184,0.02)_100%)]"
        >
          <Calendar
            v-if="selectedTeacherId"
            v-model="events"
            :editable="true"
            class="min-h-0 md:overflow-hidden"
          />
          <CalendarDisabledOverlay v-else message="Select a teacher first" />
        </div>
      </div>

      <!-- Confirm Dialog -->
      <div
        v-if="showConfirm"
        class="anim-fade-in fixed inset-0 flex items-center justify-center z-[200] bg-[rgba(45,74,62,0.5)] backdrop-blur-[8px]"
        role="dialog"
        aria-modal="true"
        aria-labelledby="confirm-title"
        @click.self="cancelConfirm"
      >
        <div
          class="anim-dialog-in bg-(--bg-card) border border-(--border-subtle) border-t-4 border-t-(--accent-sage) rounded-2xl p-8 max-md:p-7 max-w-[420px] mx-4 shadow-[var(--shadow-elevated)]"
        >
          <h3
            id="confirm-title"
            class="text-lg font-semibold font-['Instrument_Sans',sans-serif] text-(--text-primary) m-0 mb-5"
          >
            Confirm Availability
          </h3>
          <div class="flex flex-col gap-3.5 mb-6">
            <div class="flex justify-between items-center py-2">
              <span
                class="font-['JetBrains_Mono',monospace] text-[0.6875rem] font-semibold tracking-[0.1em] text-(--text-secondary) uppercase"
                >TEACHER</span
              >
              <span class="text-[0.9375rem] font-medium text-(--text-primary)">{{
                selectedTeacher?.name
              }}</span>
            </div>
            <div class="flex justify-between items-center py-2">
              <span
                class="font-['JetBrains_Mono',monospace] text-[0.6875rem] font-semibold tracking-[0.1em] text-(--text-secondary) uppercase"
                >TIME SLOTS</span
              >
              <span class="text-[0.9375rem] font-medium text-(--text-primary)"
                >{{ events.length }} slot{{ events.length > 1 ? 's' : '' }}</span
              >
            </div>
            <div
              class="flex flex-col gap-2 p-4 bg-[linear-gradient(135deg,rgba(157,180,160,0.08)_0%,rgba(157,180,160,0.04)_100%)] rounded-[10px] border border-(--border-subtle)"
            >
              <div
                v-for="(slot, index) in formattedSlots"
                :key="index"
                class="font-['JetBrains_Mono',monospace] text-[0.8125rem] text-(--text-primary) py-1"
              >
                {{ slot.day }} {{ slot.start }}-{{ slot.end }}
              </div>
            </div>
          </div>
          <div class="flex gap-3.5 justify-end">
            <button
              type="button"
              class="px-5 py-3 text-sm font-semibold font-[Inter,sans-serif] rounded-[10px] cursor-pointer transition-all duration-300 text-(--text-primary) bg-(--bg-subtle) border-2 border-(--border-subtle) hover:bg-(--border-subtle) hover:border-(--border-medium)"
              @click="cancelConfirm"
            >
              Cancel
            </button>
            <button
              type="button"
              class="px-5 py-3 text-sm font-semibold font-[Inter,sans-serif] rounded-[10px] cursor-pointer transition-all duration-300 text-white border-none bg-[linear-gradient(135deg,var(--accent-sage)_0%,var(--accent-mint)_100%)] shadow-[0_4px_12px_rgba(157,180,160,0.3)] hover:-translate-y-0.5 hover:shadow-[0_6px_16px_rgba(157,180,160,0.4)]"
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
        class="anim-toast-in fixed bottom-8 right-8 max-md:left-4 max-md:right-4 max-md:bottom-4 flex items-center gap-3 px-6 py-4 rounded-xl shadow-[var(--shadow-elevated)] z-[100] text-sm font-medium text-white bg-[linear-gradient(135deg,var(--accent-sage)_0%,var(--accent-mint)_100%)]"
        role="status"
        aria-live="polite"
      >
        <svg
          class="w-5 h-5 shrink-0"
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
        class="anim-toast-in fixed bottom-8 right-8 max-md:left-4 max-md:right-4 max-md:bottom-4 flex items-center gap-3 px-6 py-4 rounded-xl shadow-[var(--shadow-elevated)] z-[100] text-sm font-medium text-white bg-[linear-gradient(135deg,var(--accent-coral)_0%,var(--accent-coral-light)_100%)]"
        role="alert"
        aria-live="assertive"
      >
        <svg
          class="w-5 h-5 shrink-0"
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
        <button
          class="ml-3 text-[rgba(255,255,255,0.8)] bg-transparent border-none text-xl cursor-pointer w-6 h-6 flex items-center justify-center rounded-[3px] transition-all duration-200 hover:bg-[rgba(255,255,255,0.15)] focus-visible:outline-none focus-visible:text-white focus-visible:bg-[rgba(255,255,255,0.2)]"
          aria-label="Dismiss message"
          @click="errorMessage = ''"
        >
          ×
        </button>
      </div>
    </div>
  </PageLayout>
</template>

<style scoped>
  /* Native <select> is styled in CSS: the dropdown arrow is an inline SVG
     data-URI and `appearance: none` / form-control resets don't map cleanly to
     utilities. */
  .teacher-select {
    width: 100%;
    max-width: 320px;
    padding: 0.875rem 2.75rem 0.875rem 1.25rem;
    font-size: 0.9rem;
    font-family: Inter, sans-serif;
    font-weight: 500;
    color: var(--text-primary);
    background-color: var(--bg-card);
    border: 2px solid var(--border-subtle);
    border-radius: 12px;
    cursor: pointer;
    transition: all 0.3s ease;
    appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' fill='none' stroke='%234A5C52' viewBox='0 0 24 24'%3E%3Cpath stroke-linecap='round' stroke-linejoin='round' stroke-width='2.5' d='M8 10l4 4 4-4'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 1rem center;
    background-size: 1rem;
  }

  .teacher-select:hover {
    border-color: var(--accent-sage);
    background-color: var(--bg-subtle);
    transform: translateY(-1px);
  }

  .teacher-select:focus {
    outline: none;
    border-color: var(--accent-sage);
    box-shadow: 0 0 0 4px rgba(157, 180, 160, 0.2);
    background-color: var(--bg-card);
  }

  @media (max-width: 767px) {
    .teacher-select {
      padding: 0.625rem 2.5rem 0.625rem 0.875rem;
    }
  }

  @media (max-width: 480px) {
    .teacher-select {
      padding: 0.5rem 2.25rem 0.5rem 0.75rem;
      font-size: 0.8rem;
    }
  }

  /* Component-specific entrance animations. */
  @keyframes submit-toast-in {
    from {
      opacity: 0;
      transform: translateY(20px) scale(0.92);
    }
    to {
      opacity: 1;
      transform: translateY(0) scale(1);
    }
  }

  @keyframes submit-dialog-in {
    from {
      opacity: 0;
      transform: scale(0.92) translateY(12px);
    }
    to {
      opacity: 1;
      transform: scale(1) translateY(0);
    }
  }

  @keyframes submit-fade-in {
    from {
      opacity: 0;
    }
    to {
      opacity: 1;
    }
  }

  .anim-toast-in {
    animation: submit-toast-in 0.4s cubic-bezier(0.34, 1.56, 0.64, 1);
  }

  .anim-dialog-in {
    animation: submit-dialog-in 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
  }

  .anim-fade-in {
    animation: submit-fade-in 0.2s ease-out;
  }
</style>
