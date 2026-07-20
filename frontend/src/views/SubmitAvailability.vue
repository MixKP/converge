<script setup lang="ts">
  import PageLayout from '../components/PageLayout.vue'
  import Calendar from '../components/Calendar.vue'
  import CalendarDisabledOverlay from '../components/CalendarDisabledOverlay.vue'
  import FormSelect from '../components/form/FormSelect.vue'
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
    showErrors,
    formattedSlots,
    handleSubmit,
    confirmSubmit,
    cancelConfirm,
  } = useSubmitAvailability()
</script>

<template>
  <PageLayout title="Submit Availability" :show-cart="false">
    <div
      class="flex h-[calc(100dvh-60px)] justify-center overflow-hidden p-8 max-[480px]:p-3 max-md:p-4"
    >
      <!-- Unified Card Container -->
      <div
        class="flex h-full w-full max-w-350 flex-col overflow-hidden rounded-[20px] border border-(--border-subtle) bg-(--bg-card) shadow-[0_8px_32px_rgba(45,74,62,0.08),0_2px_8px_rgba(45,74,62,0.04)] max-md:rounded-2xl"
      >
        <!-- Toolbar -->
        <div
          class="flex shrink-0 items-center gap-6 border-b border-(--border-subtle) bg-[linear-gradient(180deg,rgba(157,180,160,0.04)_0%,transparent_100%)] px-8 py-5 max-md:justify-between max-md:gap-2 max-md:px-4 max-md:py-3"
        >
          <!-- Teacher Selector -->
          <div class="flex min-w-0 flex-1 flex-col max-md:flex-auto">
            <FormSelect
              v-model="selectedTeacherId"
              name="teacher_id"
              select-class="teacher-select"
              aria-label="Select teacher"
              :show-error="showErrors"
            >
              <option :value="null">Select Teacher</option>
              <option
                v-for="teacher in teacherStore.teachers"
                :key="teacher.id"
                :value="teacher.id"
              >
                {{ teacher.name }}
              </option>
            </FormSelect>
          </div>

          <!-- Submit Button -->
          <div class="flex shrink-0 items-center max-md:flex-none">
            <button
              type="submit"
              :disabled="isLoading"
              :aria-busy="isLoading"
              class="group relative flex cursor-pointer items-center justify-center overflow-hidden rounded-md bg-(--primary-indigo) px-6 py-3.5 font-[Inter,sans-serif] text-sm font-medium text-white shadow-[0_2px_8px_rgba(62,76,122,0.2),inset_0_-1px_0_rgba(0,0,0,0.1)] transition-all duration-300 ease-[cubic-bezier(0.34,1.56,0.64,1)] focus-visible:shadow-[0_0_0_3px_rgba(62,76,122,0.3),0_2px_8px_rgba(62,76,122,0.2)] focus-visible:outline-none enabled:hover:-translate-y-0.5 enabled:hover:bg-(--primary-indigo-deep) enabled:hover:shadow-[0_4px_16px_rgba(62,76,122,0.3),inset_0_-1px_0_rgba(0,0,0,0.1)] enabled:active:translate-y-0 enabled:active:shadow-[0_2px_8px_rgba(62,76,122,0.2),inset_0_1px_0_rgba(0,0,0,0.1)] disabled:translate-y-0 disabled:cursor-not-allowed disabled:opacity-40 disabled:shadow-none max-[480px]:px-3 max-[480px]:py-2 max-[480px]:text-xs"
              :class="{ 'pointer-events-none': isLoading }"
              @click="handleSubmit"
            >
              <span class="relative z-1 flex items-center gap-2">
                <svg v-if="isLoading" class="h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
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
                class="absolute top-0 -left-full h-full w-full bg-[linear-gradient(90deg,transparent,rgba(255,255,255,0.1),transparent)] transition-[left] duration-600 ease-[ease] group-[:hover:not(:disabled)]:left-full"
              ></span>
            </button>
          </div>
        </div>

        <!-- Calendar -->
        <div
          class="relative flex min-h-0 flex-1 flex-col overflow-hidden bg-[linear-gradient(135deg,rgba(157,180,160,0.03)_0%,rgba(168,201,184,0.02)_100%)] p-0 max-md:overflow-y-auto md:px-4 md:py-2 lg:px-6"
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
        class="anim-fade-in fixed inset-0 z-200 flex items-center justify-center bg-[rgba(45,74,62,0.5)] backdrop-blur-sm"
        role="dialog"
        aria-modal="true"
        aria-labelledby="confirm-title"
        @click.self="cancelConfirm"
      >
        <div
          class="anim-dialog-in mx-4 max-w-105 rounded-2xl border border-t-4 border-(--border-subtle) border-t-(--accent-sage) bg-(--bg-card) p-8 shadow-(--shadow-elevated) max-md:p-7"
        >
          <h3
            id="confirm-title"
            class="m-0 mb-5 font-['Instrument_Sans',sans-serif] text-lg font-semibold text-(--text-primary)"
          >
            Confirm Availability
          </h3>
          <div class="mb-6 flex flex-col gap-3.5">
            <div class="flex items-center justify-between py-2">
              <span
                class="font-['JetBrains_Mono',monospace] text-[0.6875rem] font-semibold tracking-widest text-(--text-secondary) uppercase"
                >TEACHER</span
              >
              <span class="text-[0.9375rem] font-medium text-(--text-primary)">{{
                selectedTeacher?.name
              }}</span>
            </div>
            <div class="flex items-center justify-between py-2">
              <span
                class="font-['JetBrains_Mono',monospace] text-[0.6875rem] font-semibold tracking-widest text-(--text-secondary) uppercase"
                >TIME SLOTS</span
              >
              <span class="text-[0.9375rem] font-medium text-(--text-primary)"
                >{{ events.length }} slot{{ events.length > 1 ? 's' : '' }}</span
              >
            </div>
            <div
              class="flex flex-col gap-2 rounded-[10px] border border-(--border-subtle) bg-[linear-gradient(135deg,rgba(157,180,160,0.08)_0%,rgba(157,180,160,0.04)_100%)] p-4"
            >
              <div
                v-for="(slot, index) in formattedSlots"
                :key="index"
                class="py-1 font-['JetBrains_Mono',monospace] text-[0.8125rem] text-(--text-primary)"
              >
                {{ slot.day }} {{ slot.start }}-{{ slot.end }}
              </div>
            </div>
          </div>
          <div class="flex justify-end gap-3.5">
            <button
              type="button"
              class="cursor-pointer rounded-[10px] border-2 border-(--border-subtle) bg-(--bg-subtle) px-5 py-3 font-[Inter,sans-serif] text-sm font-semibold text-(--text-primary) transition-all duration-300 hover:border-(--border-medium) hover:bg-(--border-subtle)"
              @click="cancelConfirm"
            >
              Cancel
            </button>
            <button
              type="button"
              class="cursor-pointer rounded-[10px] border-none bg-[linear-gradient(135deg,var(--accent-sage)_0%,var(--accent-mint)_100%)] px-5 py-3 font-[Inter,sans-serif] text-sm font-semibold text-white shadow-[0_4px_12px_rgba(157,180,160,0.3)] transition-all duration-300 hover:-translate-y-0.5 hover:shadow-[0_6px_16px_rgba(157,180,160,0.4)]"
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
        class="anim-toast-in fixed right-8 bottom-8 z-100 flex items-center gap-3 rounded-xl bg-[linear-gradient(135deg,var(--accent-sage)_0%,var(--accent-mint)_100%)] px-6 py-4 text-sm font-medium text-white shadow-(--shadow-elevated) max-md:right-4 max-md:bottom-4 max-md:left-4"
        role="status"
        aria-live="polite"
      >
        <svg
          class="h-5 w-5 shrink-0"
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
        class="anim-toast-in fixed right-8 bottom-8 z-100 flex items-center gap-3 rounded-xl bg-[linear-gradient(135deg,var(--accent-coral)_0%,var(--accent-coral-light)_100%)] px-6 py-4 text-sm font-medium text-white shadow-(--shadow-elevated) max-md:right-4 max-md:bottom-4 max-md:left-4"
        role="alert"
        aria-live="assertive"
      >
        <svg
          class="h-5 w-5 shrink-0"
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
          class="ml-3 flex h-6 w-6 cursor-pointer items-center justify-center rounded-[3px] border-none bg-transparent text-xl text-[rgba(255,255,255,0.8)] transition-all duration-200 hover:bg-[rgba(255,255,255,0.15)] focus-visible:bg-[rgba(255,255,255,0.2)] focus-visible:text-white focus-visible:outline-none"
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
