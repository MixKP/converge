<script setup lang="ts">
  import { ref, onMounted } from 'vue'
  import { useTeacherStore } from '../stores/teacherStore'
  import { useScreenSize } from '../composables/useScreenSize'
  import { useNotification } from '../composables/useNotification'
  import PageLayout from '../components/PageLayout.vue'
  import ManagementNav from '../components/ManagementNav.vue'
  import FormSelect from '../components/form/FormSelect.vue'

  const store = useTeacherStore()
  const { isMobile, isTablet } = useScreenSize()
  const { showSuccess, showError } = useNotification()

  const newName = ref('')
  const newEmail = ref('')
  const newGender = ref<'male' | 'female' | 'lgbtq+'>('male')

  const isLoading = ref(false)

  onMounted(async () => {
    store.teachers = []
    await store.reloadTeachers()
  })

  const handleToggle = async (id: number, currentStatus: string) => {
    try {
      await store.toggleTeacherStatus(id, currentStatus)
      showSuccess(`Teacher deactivated successfully`)
    } catch (err) {
      showError(err, 'Failed to update teacher status')
    }
  }

  const handleGenderChange = async (id: number, gender: string) => {
    try {
      await store.updateTeacherGender(id, gender)
      showSuccess('Teacher gender updated successfully')
    } catch (err) {
      showError(err, 'Failed to update teacher gender')
    }
  }

  const handleCreate = async () => {
    if (!newName.value.trim() || !newEmail.value.trim()) return
    isLoading.value = true
    try {
      await store.createTeacher(newName.value.trim(), newEmail.value.trim(), newGender.value)
      newName.value = ''
      newEmail.value = ''
      newGender.value = 'male'
      showSuccess('Teacher created successfully')
    } catch (err) {
      showError(err, 'Failed to create teacher')
    } finally {
      isLoading.value = false
    }
  }
</script>

<template>
  <PageLayout title="Manage Teachers" :show-cart="false">
    <div class="manage-root">
      <ManagementNav />
      <!-- Add Teacher Form -->
      <div class="manage-section">
        <h2 class="manage-section-title">Add Teacher</h2>
        <div class="manage-form">
          <div class="manage-form-row">
            <div class="manage-form-field">
              <label class="manage-label">Name</label>
              <input
                v-model="newName"
                type="text"
                class="manage-input"
                placeholder="Teacher name"
              />
            </div>
            <div class="manage-form-field">
              <label class="manage-label">Email</label>
              <input
                v-model="newEmail"
                type="email"
                class="manage-input"
                placeholder="teacher@example.com"
              />
            </div>
            <div class="manage-form-field manage-form-field--gender">
              <label class="manage-label">Gender</label>
              <FormSelect
                v-model="newGender"
                name="new_gender"
                select-class="manage-select"
                :show-error="false"
              >
                <option value="male">Male</option>
                <option value="female">Female</option>
                <option value="lgbtq+">LGBTQ+</option>
              </FormSelect>
            </div>
            <div class="manage-form-field manage-form-field--action">
              <button
                type="button"
                class="manage-btn manage-btn--primary"
                :disabled="!newName.trim() || !newEmail.trim() || isLoading"
                @click="handleCreate"
              >
                {{ isLoading ? 'Adding...' : 'Add Teacher' }}
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Mobile / Tablet Card List -->
      <div
        v-if="isMobile || isTablet"
        class="manage-section"
      >
        <h2 class="manage-section-title">Teachers ({{ store.teachers.length }})</h2>
        <div v-if="store.teachers.length === 0" class="manage-empty">
          No teachers found
        </div>
        <div v-else class="manage-card-list">
          <div
            v-for="teacher in store.teachers"
            :key="teacher.id"
            class="manage-card"
            :class="{ 'manage-card--deactivated': teacher.status === 'deactivated' }"
          >
            <div class="manage-card-header">
              <div>
                <div class="manage-card-name">{{ teacher.name }}</div>
                <div class="manage-card-email">{{ teacher.email }}</div>
              </div>
              <button
                type="button"
                class="manage-toggle"
                :class="{ 'manage-toggle--active': teacher.status === 'active' }"
                :aria-label="`Toggle ${teacher.name} status`"
                @click="handleToggle(teacher.id, teacher.status)"
              >
                <span class="manage-toggle-knob" />
              </button>
            </div>
            <div class="manage-card-row">
              <span class="manage-card-label">Gender</span>
              <select
                class="manage-card-select"
                :value="teacher.gender"
                @change="handleGenderChange(teacher.id, ($event.target as HTMLSelectElement).value)"
              >
                <option value="male">Male</option>
                <option value="female">Female</option>
                <option value="lgbtq+">LGBTQ+</option>
              </select>
            </div>
            <div class="manage-card-row">
              <span class="manage-card-label">Status</span>
              <span
                class="manage-badge"
                :class="teacher.status === 'active' ? 'manage-badge--active' : 'manage-badge--deactivated'"
              >
                {{ teacher.status }}
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- Desktop Table -->
      <div v-else class="manage-section">
        <h2 class="manage-section-title">Teachers ({{ store.teachers.length }})</h2>
        <div v-if="store.teachers.length === 0" class="manage-empty">
          No teachers found
        </div>
        <div v-else class="manage-table-wrap">
          <table class="manage-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Email</th>
                <th>Gender</th>
                <th>Status</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="teacher in store.teachers"
                :key="teacher.id"
                :class="{ 'manage-row--deactivated': teacher.status === 'deactivated' }"
              >
                <td class="manage-cell-name">{{ teacher.name }}</td>
                <td class="manage-cell-email">{{ teacher.email }}</td>
                <td>
                  <select
                    class="manage-table-select"
                    :value="teacher.gender"
                    @change="handleGenderChange(teacher.id, ($event.target as HTMLSelectElement).value)"
                  >
                    <option value="male">Male</option>
                    <option value="female">Female</option>
                    <option value="lgbtq+">LGBTQ+</option>
                  </select>
                </td>
                <td>
                  <span
                    class="manage-badge"
                    :class="teacher.status === 'active' ? 'manage-badge--active' : 'manage-badge--deactivated'"
                  >
                    {{ teacher.status }}
                  </span>
                </td>
                <td>
                  <button
                    type="button"
                    class="manage-toggle"
                    :class="{ 'manage-toggle--active': teacher.status === 'active' }"
                    :aria-label="`Toggle ${teacher.name} status`"
                    @click="handleToggle(teacher.id, teacher.status)"
                  />
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
    margin: 0 0 1rem;
    font-family: 'Instrument Sans', sans-serif;
    font-size: 1.125rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .manage-form {
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

  .manage-form-field--gender {
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

  .manage-select {
    padding: 0.5rem 0.75rem;
    font-size: 0.875rem;
    font-family: Inter, sans-serif;
    color: var(--text-primary);
    background: var(--bg-cream);
    border: 1px solid var(--border-medium);
    border-radius: 0.5rem;
    outline: none;
    cursor: pointer;
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

  .manage-card--deactivated {
    opacity: 0.6;
  }

  .manage-card-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
  }

  .manage-card-name {
    font-family: Inter, sans-serif;
    font-size: 0.9375rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .manage-card-email {
    font-family: Inter, sans-serif;
    font-size: 0.8125rem;
    color: var(--text-secondary);
    margin-top: 0.125rem;
  }

  .manage-card-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .manage-card-label {
    font-family: Inter, sans-serif;
    font-size: 0.75rem;
    font-weight: 600;
    letter-spacing: 0.05em;
    text-transform: uppercase;
    color: var(--text-muted);
  }

  .manage-card-select {
    padding: 0.375rem 0.625rem;
    font-size: 0.8125rem;
    font-family: Inter, sans-serif;
    color: var(--text-primary);
    background: var(--bg-cream);
    border: 1px solid var(--border-medium);
    border-radius: 0.375rem;
    cursor: pointer;
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

  .manage-row--deactivated {
    opacity: 0.55;
  }

  .manage-cell-name {
    font-weight: 500;
  }

  .manage-cell-email {
    color: var(--text-secondary);
  }

  .manage-table-select {
    padding: 0.375rem 0.625rem;
    font-size: 0.8125rem;
    font-family: Inter, sans-serif;
    color: var(--text-primary);
    background: var(--bg-cream);
    border: 1px solid var(--border-medium);
    border-radius: 0.375rem;
    cursor: pointer;
  }

  .manage-badge {
    display: inline-block;
    padding: 0.1875rem 0.5rem;
    font-size: 0.6875rem;
    font-weight: 600;
    border-radius: 9999px;
    text-transform: capitalize;
  }

  .manage-badge--active {
    color: #166534;
    background: #dcfce7;
  }

  .manage-badge--deactivated {
    color: #991b1b;
    background: #fef2f2;
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

  .manage-toggle-knob,
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

  @media (max-width: 767px) {
    .manage-section {
      padding: 1rem;
    }

    .manage-form-row {
      flex-direction: column;
      gap: 0.75rem;
    }

    .manage-form-field--gender {
      flex: 1;
    }

    .manage-form-field--action {
      flex: 1;
    }

    .manage-btn {
      width: 100%;
    }

    .manage-section-title {
      font-size: 1rem;
    }
  }

  @media (min-width: 768px) and (max-width: 1023px) {
    .manage-section {
      padding: 1.25rem;
    }
  }
</style>
