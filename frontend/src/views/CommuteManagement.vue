<script setup lang="ts">
  import { ref, onMounted } from 'vue'
  import { useNotification } from '../composables/useNotification'
  import PageLayout from '../components/PageLayout.vue'
  import ManagementNav from '../components/ManagementNav.vue'
  import { commuteApi } from '../services/commuteApi'
  import { branchApi } from '../services/branchApi'
  import type { Branch } from '../types'

  const { showSuccess, showError } = useNotification()

  const currentMinutes = ref<number | null>(null)
  const draft = ref('')
  const isSaving = ref(false)
  const isLoading = ref(false)

  const branches = ref<Branch[]>([])
  const commuteMatrix = ref<Record<string, number>>({})
  const isMatrixLoading = ref(false)

  onMounted(async () => {
    isLoading.value = true
    try {
      const [data, branchData] = await Promise.all([commuteApi.get(), branchApi.getAll()])
      currentMinutes.value = data.commute_time
      draft.value = String(data.commute_time)
      branches.value = branchData
    } catch (err) {
      showError(err, 'Failed to load commute time')
    } finally {
      isLoading.value = false
    }
    fetchMatrix()
  })

  const fetchMatrix = async () => {
    const pairs: Array<[number, number]> = []
    for (const src of branches.value) {
      for (const dest of branches.value) {
        if (src.id !== dest.id) pairs.push([src.id, dest.id])
      }
    }
    if (pairs.length === 0) return

    isMatrixLoading.value = true
    try {
      const results = await Promise.all(pairs.map(([s, d]) => commuteApi.getForBranches(s, d)))
      const matrix: Record<string, number> = {}
      results.forEach((res, i) => {
        const [s, d] = pairs[i]
        matrix[`${s}:${d}`] = res.commute_time
      })
      commuteMatrix.value = matrix
    } catch (err) {
      showError(err, 'Failed to load commute times between branches')
    } finally {
      isMatrixLoading.value = false
    }
  }

  const commuteBetween = (srcId: number, destId: number): number | null => {
    if (srcId === destId) return 0
    const value = commuteMatrix.value[`${srcId}:${destId}`]
    return typeof value === 'number' ? value : null
  }

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
        <div class="matrix-section">
          <h2 class="matrix-title">Commute Time Between Branches</h2>
          <p class="matrix-subtitle">
            Time in minutes from each branch (row) to each branch (column). Same-branch travel is 0.
            Cells without a custom value use the default commute time.
          </p>

          <div v-if="isMatrixLoading" class="manage-empty">Loading commute matrix...</div>
          <div v-else-if="branches.length === 0" class="manage-empty">No branches found</div>
          <div v-else class="matrix-table-wrap">
            <table class="matrix-table">
              <thead>
                <tr>
                  <th class="matrix-corner">From \\ To</th>
                  <th v-for="dest in branches" :key="`h-${dest.id}`" class="matrix-header-cell">
                    {{ dest.name }}
                  </th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="src in branches" :key="`r-${src.id}`">
                  <th class="matrix-row-label">{{ src.name }}</th>
                  <td
                    v-for="dest in branches"
                    :key="`c-${src.id}-${dest.id}`"
                    class="matrix-cell"
                    :class="{
                      'matrix-cell--zero': src.id === dest.id,
                      'matrix-cell--default':
                        src.id !== dest.id && commuteBetween(src.id, dest.id) === currentMinutes,
                      'matrix-cell--custom':
                        src.id !== dest.id &&
                        commuteBetween(src.id, dest.id) !== null &&
                        commuteBetween(src.id, dest.id) !== currentMinutes,
                    }"
                  >
                    <template v-if="commuteBetween(src.id, dest.id) !== null">
                      {{ commuteBetween(src.id, dest.id) }}m
                    </template>
                    <template v-else>—</template>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
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

  .matrix-section {
    margin-top: 1.5rem;
  }

  .matrix-title {
    margin: 0 0 0.25rem;
    font-family: 'Instrument Sans', sans-serif;
    font-size: 1.125rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .matrix-subtitle {
    margin: 0 0 1rem;
    font-family: Inter, sans-serif;
    font-size: 0.8125rem;
    color: var(--text-secondary);
  }

  .matrix-table-wrap {
    overflow-x: auto;
    border-radius: 0.75rem;
    border: 1px solid var(--border-subtle);
    background: var(--bg-card);
  }

  .matrix-table {
    width: 100%;
    border-collapse: collapse;
    font-family: Inter, sans-serif;
    font-size: 0.8125rem;
  }

  .matrix-table th,
  .matrix-table td {
    padding: 0.625rem 0.75rem;
    text-align: center;
    white-space: nowrap;
    border-bottom: 1px solid var(--border-subtle);
  }

  .matrix-table tbody tr:last-child th,
  .matrix-table tbody tr:last-child td {
    border-bottom: none;
  }

  .matrix-corner,
  .matrix-header-cell {
    font-size: 0.6875rem;
    font-weight: 600;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: var(--text-muted);
    background: var(--bg-subtle);
  }

  .matrix-header-cell {
    text-align: center;
  }

  .matrix-row-label {
    text-align: left;
    font-weight: 500;
    color: var(--text-primary);
  }

  .matrix-cell {
    color: var(--text-primary);
  }

  .matrix-cell--zero {
    color: var(--text-muted);
  }

  .matrix-cell--default {
    background: var(--bg-subtle);
  }

  .matrix-cell--custom {
    background: rgba(157, 180, 160, 0.15);
    font-weight: 600;
    color: var(--accent-sage);
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
