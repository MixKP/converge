<script setup lang="ts">
  import { ref, onMounted } from 'vue'
  import { useNotification } from '../composables/useNotification'
  import PageLayout from '../components/PageLayout.vue'
  import ManagementNav from '../components/ManagementNav.vue'
  import { commuteApi } from '../services/commuteApi'

  const { showSuccess, showError } = useNotification()

  const currentMinutes = ref<number | null>(null)
  const draft = ref('')
  const isSaving = ref(false)
  const isLoading = ref(false)

  onMounted(async () => {
    isLoading.value = true
    try {
      const data = await commuteApi.get()
      currentMinutes.value = data.commute_time
      draft.value = String(data.commute_time)
    } catch (err) {
      showError(err, 'Failed to load commute time')
    } finally {
      isLoading.value = false
    }
  })

  const isDirty = () =>
    currentMinutes.value !== null && String(draft.value) !== String(currentMinutes.value)

  const handleSave = async () => {
    const raw = String(draft.value ?? '').trim()
    if (raw === '') return
    const minutes = Number(raw)
    if (!Number.isInteger(minutes) || minutes < 0) {
      showError(null, 'Commute time must be a whole number of 0 or more')
      return
    }

    isSaving.value = true
    try {
      await commuteApi.set(minutes)
      currentMinutes.value = minutes
      draft.value = String(minutes)
      showSuccess(`Commute time updated to ${minutes} minute${minutes === 1 ? '' : 's'}`)
    } catch (err) {
      showError(err, 'Failed to update commute time')
    } finally {
      isSaving.value = false
    }
  }
</script>

<template>
  <PageLayout title="Manage Commute" :show-cart="false" back-to="/dashboard" back-label="Dashboard">
    <div class="manage-root">
      <ManagementNav />
      <div class="manage-section">
        <h2 class="manage-section-title">Commute Time</h2>
        <p class="manage-section-subtitle">
          Travel time in minutes applied between different branches. The smart booking engine uses
          it to pad conflicting bookings when a teacher must travel between branches.
        </p>
        <div v-if="isLoading" class="manage-empty">Loading commute time...</div>
        <div v-else-if="currentMinutes !== null" class="commute-card">
          <div class="commute-card-row">
            <div class="commute-card-info">
              <div class="commute-card-label">Current Commute Time</div>
              <div class="commute-card-current">
                {{ currentMinutes }}
                <span class="commute-card-unit"> minute{{ currentMinutes === 1 ? '' : 's' }} </span>
              </div>
            </div>
          </div>

          <div class="commute-form">
            <div class="commute-form-field">
              <label class="commute-label" for="commute-time-input">New Commute Time</label>
              <input
                id="commute-time-input"
                v-model="draft"
                type="number"
                min="0"
                step="1"
                class="commute-input"
                aria-label="Commute time in minutes"
              />
            </div>
            <button
              type="button"
              class="commute-btn"
              :disabled="isSaving || !isDirty()"
              @click="handleSave"
            >
              {{ isSaving ? 'Saving...' : 'Save' }}
            </button>
          </div>
        </div>
        <div v-else class="manage-empty">Unable to load commute time</div>
      </div>
    </div>
  </PageLayout>
</template>

<style scoped>
  .manage-root {
    width: 100%;
    height: 100%;
    overflow: hidden;
  }

  .manage-section {
    padding: 1.5rem;
    max-width: 64rem;
    margin: 0 auto;
  }

  .manage-section-title {
    margin: 0 0 0.25rem;
    font-family: 'Instrument Sans', sans-serif;
    font-size: 1.125rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .manage-section-subtitle {
    margin: 0 0 1rem;
    font-family: Inter, sans-serif;
    font-size: 0.8125rem;
    color: var(--text-secondary);
  }

  .manage-empty {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 10rem;
    font-family: Inter, sans-serif;
    font-size: 0.875rem;
    color: var(--text-secondary);
    border-radius: 0.75rem;
    border: 2px dashed var(--border-medium);
    background: var(--bg-subtle);
  }

  .commute-card {
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: 0.75rem;
    padding: 1.25rem;
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }

  .commute-card-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .commute-card-info {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .commute-card-label {
    font-family: Inter, sans-serif;
    font-size: 0.75rem;
    font-weight: 600;
    letter-spacing: 0.05em;
    text-transform: uppercase;
    color: var(--text-muted);
  }

  .commute-card-current {
    font-family: 'Instrument Sans', sans-serif;
    font-size: 1.75rem;
    font-weight: 600;
    color: var(--primary-indigo);
    line-height: 1.1;
  }

  .commute-card-unit {
    font-family: Inter, sans-serif;
    font-size: 0.9375rem;
    font-weight: 400;
    color: var(--text-secondary);
  }

  .commute-form {
    display: flex;
    gap: 0.75rem;
    align-items: flex-end;
  }

  .commute-form-field {
    display: flex;
    flex-direction: column;
    gap: 0.375rem;
  }

  .commute-label {
    font-family: Inter, sans-serif;
    font-size: 0.75rem;
    font-weight: 600;
    letter-spacing: 0.05em;
    text-transform: uppercase;
    color: var(--text-muted);
  }

  .commute-input {
    width: 8rem;
    padding: 0.5rem 0.75rem;
    font-size: 0.875rem;
    font-family: Inter, sans-serif;
    color: var(--text-primary);
    background: var(--bg-cream);
    border: 1px solid var(--border-medium);
    border-radius: 0.5rem;
    outline: none;
    transition: border-color 0.15s;
  }

  .commute-input:focus {
    border-color: var(--accent-sage);
    box-shadow: 0 0 0 3px rgba(157, 180, 160, 0.15);
  }

  .commute-btn {
    padding: 0.5rem 1.25rem;
    font-size: 0.875rem;
    font-family: Inter, sans-serif;
    font-weight: 500;
    color: #fff;
    background: var(--primary-indigo);
    border: none;
    border-radius: 0.5rem;
    cursor: pointer;
    transition: all 0.15s;
    white-space: nowrap;
  }

  .commute-btn:hover:not(:disabled) {
    opacity: 0.9;
  }

  .commute-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  @media (max-width: 767px) {
    .manage-section {
      padding: 1rem;
    }

    .manage-section-title {
      font-size: 1rem;
    }

    .commute-form {
      flex-direction: column;
      align-items: stretch;
    }

    .commute-input {
      width: 100%;
    }
  }

  @media (min-width: 768px) and (max-width: 1023px) {
    .manage-section {
      padding: 1.25rem;
    }
  }
</style>
