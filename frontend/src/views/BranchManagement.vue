<script setup lang="ts">
  import { ref, onMounted } from 'vue'
  import { useScreenSize } from '../composables/useScreenSize'
  import { useNotification } from '../composables/useNotification'
  import PageLayout from '../components/PageLayout.vue'
  import ManagementNav from '../components/ManagementNav.vue'
  import { branchApi } from '../services/branchApi'
  import type { Branch } from '../types'

  const { isMobile, isTablet } = useScreenSize()
  const { showSuccess, showError } = useNotification()

  const branches = ref<Branch[]>([])
  const drafts = ref<Record<number, string>>({})
  const savingId = ref<number | null>(null)
  const togglingId = ref<number | null>(null)
  const isLoading = ref(false)

  const newName = ref('')
  const newCapacity = ref('0')
  const isCreating = ref(false)

  onMounted(async () => {
    isLoading.value = true
    try {
      branches.value = await branchApi.getAll()
      branches.value.forEach((b) => {
        drafts.value[b.id] = String(b.capacity)
      })
    } catch (err) {
      showError(err, 'Failed to load branches')
    } finally {
      isLoading.value = false
    }
  })

  const handleCreate = async () => {
    const name = newName.value.trim()
    if (name === '') return

    const raw = String(newCapacity.value ?? '').trim()
    const capacity = raw === '' ? 0 : Number(raw)
    if (!Number.isInteger(capacity) || capacity < 0) {
      showError(null, 'Capacity must be a whole number of 0 or more')
      return
    }

    isCreating.value = true
    try {
      const created = await branchApi.create(name, capacity)
      branches.value.push(created)
      drafts.value[created.id] = String(created.capacity)
      newName.value = ''
      newCapacity.value = '0'
      showSuccess(`Branch ${created.name} created successfully`)
    } catch (err) {
      showError(err, 'Failed to create branch')
    } finally {
      isCreating.value = false
    }
  }

  const handleToggleStatus = async (branch: Branch) => {
    const next = branch.status === 'active' ? 'deactivated' : 'active'
    togglingId.value = branch.id
    try {
      await branchApi.setStatus(branch.id, next)
      branch.status = next
      showSuccess(
        next === 'deactivated'
          ? `${branch.name} removed from booking`
          : `${branch.name} restored to booking`
      )
    } catch (err) {
      showError(err, 'Failed to update branch status')
    } finally {
      togglingId.value = null
    }
  }

  const handleSave = async (branch: Branch) => {
    const raw = String(drafts.value[branch.id] ?? '').trim()
    if (raw === '') return
    const capacity = Number(raw)
    if (!Number.isInteger(capacity) || capacity < 0) {
      showError(null, 'Capacity must be a whole number of 0 or more')
      return
    }

    savingId.value = branch.id
    try {
      await branchApi.setCapacity(branch.id, capacity)
      branch.capacity = capacity
      drafts.value[branch.id] = String(capacity)
      showSuccess(
        `Capacity for ${branch.name} updated to ${capacity === 0 ? 'unlimited' : capacity}`
      )
    } catch (err) {
      showError(err, 'Failed to update branch capacity')
    } finally {
      savingId.value = null
    }
  }
</script>

<template>
  <PageLayout
    title="Manage Branches"
    :show-cart="false"
    back-to="/dashboard"
    back-label="Dashboard"
  >
    <div class="manage-root">
      <ManagementNav />
      <div class="manage-section">
        <h2 class="manage-section-title">Add Branch</h2>
        <div class="manage-form">
          <div class="manage-form-row">
            <div class="manage-form-field">
              <label class="manage-label">Name</label>
              <input
                v-model="newName"
                type="text"
                class="manage-input"
                placeholder="Branch name"
                @keyup.enter="handleCreate"
              />
            </div>
            <div class="manage-form-field manage-form-field--capacity">
              <label class="manage-label">Capacity</label>
              <input
                v-model="newCapacity"
                type="number"
                min="0"
                step="1"
                class="manage-input"
                placeholder="0"
                @keyup.enter="handleCreate"
              />
            </div>
            <div class="manage-form-field manage-form-field--action">
              <button
                type="button"
                class="manage-btn manage-btn--primary"
                :disabled="!newName.trim() || isCreating"
                @click="handleCreate"
              >
                {{ isCreating ? 'Adding...' : 'Add Branch' }}
              </button>
            </div>
          </div>
        </div>
      </div>
      <div class="manage-section">
        <h2 class="manage-section-title">Branch Capacity</h2>
        <p class="manage-section-subtitle">
          A capacity of 0 means unlimited concurrent bookings for that branch. Removed branches keep
          their booking history but cannot take new bookings.
        </p>
        <div v-if="isLoading" class="manage-empty">Loading branches...</div>
        <div v-else-if="branches.length === 0" class="manage-empty">No branches found</div>
        <div v-else-if="isMobile || isTablet" class="manage-card-list">
          <div
            v-for="branch in branches"
            :key="branch.id"
            class="manage-card"
            :class="{ 'manage-card--deactivated': branch.status === 'deactivated' }"
          >
            <div class="manage-card-header">
              <div class="manage-card-name">{{ branch.name }}</div>
              <span
                class="manage-badge"
                :class="branch.capacity === 0 ? 'manage-badge--unlimited' : 'manage-badge--capped'"
              >
                {{ branch.capacity === 0 ? 'Unlimited' : `Max ${branch.capacity}` }}
              </span>
            </div>
            <div class="manage-card-row">
              <span class="manage-card-label">Bookable</span>
              <button
                type="button"
                class="manage-toggle"
                :class="{ 'manage-toggle--active': branch.status === 'active' }"
                :disabled="togglingId === branch.id"
                :aria-label="`Toggle ${branch.name} status`"
                @click="handleToggleStatus(branch)"
              />
            </div>
            <div class="manage-card-row">
              <span class="manage-card-label">Capacity</span>
              <div class="manage-card-actions">
                <input
                  v-model="drafts[branch.id]"
                  type="number"
                  min="0"
                  step="1"
                  class="manage-capacity-input"
                  :aria-label="`Capacity for ${branch.name}`"
                />
                <button
                  type="button"
                  class="manage-btn manage-btn--primary"
                  :disabled="
                    savingId === branch.id || drafts[branch.id] === String(branch.capacity)
                  "
                  @click="handleSave(branch)"
                >
                  {{ savingId === branch.id ? 'Saving...' : 'Save' }}
                </button>
              </div>
            </div>
          </div>
        </div>
        <div v-else class="manage-table-wrap">
          <table class="manage-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Capacity</th>
                <th>Bookable</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="branch in branches"
                :key="branch.id"
                :class="{ 'manage-row--deactivated': branch.status === 'deactivated' }"
              >
                <td class="manage-cell-name">{{ branch.name }}</td>
                <td>
                  <span
                    class="manage-badge"
                    :class="
                      branch.capacity === 0 ? 'manage-badge--unlimited' : 'manage-badge--capped'
                    "
                  >
                    {{ branch.capacity === 0 ? 'Unlimited' : `Max ${branch.capacity}` }}
                  </span>
                </td>
                <td>
                  <button
                    type="button"
                    class="manage-toggle"
                    :class="{ 'manage-toggle--active': branch.status === 'active' }"
                    :disabled="togglingId === branch.id"
                    :aria-label="`Toggle ${branch.name} status`"
                    @click="handleToggleStatus(branch)"
                  />
                </td>
                <td>
                  <div class="manage-table-actions">
                    <input
                      v-model="drafts[branch.id]"
                      type="number"
                      min="0"
                      step="1"
                      class="manage-capacity-input"
                      :aria-label="`Capacity for ${branch.name}`"
                    />
                    <button
                      type="button"
                      class="manage-btn manage-btn--primary"
                      :disabled="
                        savingId === branch.id || drafts[branch.id] === String(branch.capacity)
                      "
                      @click="handleSave(branch)"
                    >
                      {{ savingId === branch.id ? 'Saving...' : 'Save' }}
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
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

  .manage-form {
    margin-top: 0.75rem;
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: 0.75rem;
    padding: 1.25rem;
  }

  .manage-form-row {
    display: flex;
    gap: 1rem;
    align-items: flex-end;
  }

  .manage-form-field {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 0.375rem;
  }

  .manage-form-field--capacity {
    flex: 0 0 8rem;
  }

  .manage-form-field--action {
    flex: 0 0 auto;
  }

  .manage-label {
    font-family: Inter, sans-serif;
    font-size: 0.75rem;
    font-weight: 600;
    letter-spacing: 0.05em;
    text-transform: uppercase;
    color: var(--text-muted);
  }

  .manage-input {
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

  .manage-input:focus {
    border-color: var(--accent-sage);
    box-shadow: 0 0 0 3px rgba(157, 180, 160, 0.15);
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

  .manage-card-list {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .manage-card {
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: 0.75rem;
    padding: 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .manage-card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 0.5rem;
  }

  .manage-card-name {
    font-family: Inter, sans-serif;
    font-size: 0.9375rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .manage-card-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 0.5rem;
  }

  .manage-card-label {
    font-family: Inter, sans-serif;
    font-size: 0.75rem;
    font-weight: 600;
    letter-spacing: 0.05em;
    text-transform: uppercase;
    color: var(--text-muted);
  }

  .manage-card-actions {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }

  .manage-table-wrap {
    overflow-x: auto;
    border-radius: 0.75rem;
    border: 1px solid var(--border-subtle);
  }

  .manage-table {
    width: 100%;
    border-collapse: collapse;
    font-family: Inter, sans-serif;
  }

  .manage-table th {
    text-align: left;
    padding: 0.75rem 1rem;
    font-size: 0.6875rem;
    font-weight: 600;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--text-muted);
    background: var(--bg-subtle);
    border-bottom: 1px solid var(--border-subtle);
  }

  .manage-table td {
    padding: 0.75rem 1rem;
    font-size: 0.875rem;
    color: var(--text-primary);
    border-bottom: 1px solid var(--border-subtle);
    vertical-align: middle;
  }

  .manage-table tbody tr:last-child td {
    border-bottom: none;
  }

  .manage-cell-name {
    font-weight: 500;
  }

  .manage-table-actions {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }

  .manage-capacity-input {
    width: 6rem;
    padding: 0.375rem 0.625rem;
    font-size: 0.875rem;
    font-family: Inter, sans-serif;
    color: var(--text-primary);
    background: var(--bg-cream);
    border: 1px solid var(--border-medium);
    border-radius: 0.375rem;
    outline: none;
    transition: border-color 0.15s;
  }

  .manage-capacity-input:focus {
    border-color: var(--accent-sage);
    box-shadow: 0 0 0 3px rgba(157, 180, 160, 0.15);
  }

  .manage-btn {
    padding: 0.5rem 1.25rem;
    font-size: 0.875rem;
    font-family: Inter, sans-serif;
    font-weight: 500;
    border: none;
    border-radius: 0.5rem;
    cursor: pointer;
    transition: all 0.15s;
    white-space: nowrap;
  }

  .manage-btn--primary {
    color: #fff;
    background: var(--primary-indigo);
  }

  .manage-btn--primary:hover:not(:disabled) {
    opacity: 0.9;
  }

  .manage-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .manage-badge {
    display: inline-block;
    padding: 0.1875rem 0.5rem;
    font-size: 0.6875rem;
    font-weight: 600;
    border-radius: 9999px;
    text-transform: capitalize;
  }

  .manage-card--deactivated {
    opacity: 0.6;
  }

  .manage-row--deactivated {
    opacity: 0.55;
  }

  .manage-toggle {
    position: relative;
    width: 2.5rem;
    height: 1.375rem;
    background: #d1d5db;
    border: none;
    border-radius: 9999px;
    cursor: pointer;
    transition: background 0.2s;
    flex-shrink: 0;
    padding: 0;
  }

  .manage-toggle--active {
    background: var(--primary-indigo);
  }

  .manage-toggle:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .manage-toggle::after {
    content: '';
    position: absolute;
    top: 0.1875rem;
    left: 0.1875rem;
    width: 1rem;
    height: 1rem;
    background: #fff;
    border-radius: 50%;
    transition: transform 0.2s;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.15);
  }

  .manage-toggle--active::after {
    transform: translateX(1.125rem);
  }

  .manage-badge--unlimited {
    color: #166534;
    background: #dcfce7;
  }

  .manage-badge--capped {
    color: var(--text-secondary);
    background: var(--bg-subtle);
    border: 1px solid var(--border-subtle);
  }

  @media (max-width: 767px) {
    .manage-section {
      padding: 1rem;
    }

    .manage-section-title {
      font-size: 1rem;
    }

    .manage-form-row {
      flex-direction: column;
      gap: 0.75rem;
      align-items: stretch;
    }

    .manage-form-field--capacity,
    .manage-form-field--action {
      flex: 1;
    }

    .manage-btn {
      width: 100%;
    }
  }

  @media (min-width: 768px) and (max-width: 1023px) {
    .manage-section {
      padding: 1.25rem;
    }
  }
</style>
