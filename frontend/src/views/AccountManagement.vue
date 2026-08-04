<script setup lang="ts">
  import { ref, computed, onMounted } from 'vue'
  import { useScreenSize } from '../composables/useScreenSize'
  import { useNotification } from '../composables/useNotification'
  import PageLayout from '../components/PageLayout.vue'
  import ManagementNav from '../components/ManagementNav.vue'
  import { userApi } from '../services/userApi'
  import { authApi } from '../services/authApi'
  import { registerRequestSchema } from '../schemas/auth'
  import type { AuthUser, ParentWithStudents, Role } from '../types'

  const { isMobile, isTablet } = useScreenSize()
  const { showSuccess, showError } = useNotification()

  const isCompact = computed(() => isMobile.value || isTablet.value)

  type Tab = 'students' | 'parents'
  const activeTab = ref<Tab>('students')

  const students = ref<AuthUser[]>([])
  const parents = ref<ParentWithStudents[]>([])
  const isLoading = ref(false)

  const expandedParentId = ref<number | null>(null)
  const linkDraft = ref<Record<number, string>>({})
  const busyParentId = ref<number | null>(null)

  const isModalOpen = ref(false)
  const isSubmitting = ref(false)
  const formError = ref('')
  const form = ref({
    name: '',
    password: '',
    role: 'student' as Role,
    studentIds: [] as number[],
  })

  const loadAccounts = async () => {
    isLoading.value = true
    try {
      // Fetched together: the create-parent flow needs the student list, and the
      // parents table needs it to label links, so one round trip serves both.
      const [nextStudents, nextParents] = await Promise.all([
        userApi.listStudents(),
        userApi.listParents(),
      ])
      students.value = nextStudents
      parents.value = nextParents
    } catch (err) {
      showError(err, 'Failed to load accounts')
    } finally {
      isLoading.value = false
    }
  }

  onMounted(loadAccounts)

  const openModal = () => {
    form.value = { name: '', password: '', role: 'student', studentIds: [] }
    formError.value = ''
    isModalOpen.value = true
  }

  const closeModal = () => {
    if (isSubmitting.value) return
    isModalOpen.value = false
  }

  const toggleFormStudent = (id: number) => {
    const current = form.value.studentIds
    form.value.studentIds = current.includes(id)
      ? current.filter((s) => s !== id)
      : [...current, id]
  }

  // The backend rejects a parent with no links, so block the request client-side
  // rather than surfacing a 400 the admin cannot act on.
  const canSubmit = computed(() => {
    if (isSubmitting.value) return false
    if (form.value.name.trim() === '' || form.value.password === '') return false
    return form.value.role !== 'parent' || form.value.studentIds.length > 0
  })

  const handleCreate = async () => {
    formError.value = ''
    const parsed = registerRequestSchema.safeParse({
      name: form.value.name,
      password: form.value.password,
      role: form.value.role,
      student_ids: form.value.role === 'parent' ? form.value.studentIds : undefined,
    })
    if (!parsed.success) {
      formError.value = parsed.error.issues[0]?.message ?? 'Please check the form'
      return
    }

    isSubmitting.value = true
    try {
      const created = await authApi.register(parsed.data)
      showSuccess(`Account "${created.name}" created as ${created.role}`)
      isModalOpen.value = false
      await loadAccounts()
    } catch (err) {
      // A duplicate username comes back as 409; keep it inline on the form so the
      // admin can correct the name without losing what they typed.
      formError.value = err instanceof Error ? err.message : 'Failed to create account'
    } finally {
      isSubmitting.value = false
    }
  }

  const toggleParent = (parentId: number) => {
    expandedParentId.value = expandedParentId.value === parentId ? null : parentId
  }

  const unlinkedStudentsFor = (parent: ParentWithStudents) => {
    const linked = new Set(parent.students.map((s) => s.id))
    return students.value.filter((s) => !linked.has(s.id))
  }

  const handleLink = async (parent: ParentWithStudents) => {
    const raw = String(linkDraft.value[parent.id] ?? '').trim()
    if (raw === '') return
    const studentId = Number(raw)
    if (!Number.isInteger(studentId) || studentId <= 0) return

    busyParentId.value = parent.id
    try {
      await userApi.linkStudent(parent.id, studentId)
      const student = students.value.find((s) => s.id === studentId)
      if (student) parent.students.push(student)
      linkDraft.value[parent.id] = ''
      showSuccess(`Linked ${student?.name ?? 'student'} to ${parent.name}`)
    } catch (err) {
      showError(err, 'Failed to link student')
    } finally {
      busyParentId.value = null
    }
  }

  const handleUnlink = async (parent: ParentWithStudents, student: AuthUser) => {
    // The backend requires a parent to keep at least one student, so removing the
    // last link would orphan the account.
    if (parent.students.length <= 1) {
      showError(null, `${parent.name} must stay linked to at least one student`)
      return
    }

    busyParentId.value = parent.id
    try {
      await userApi.unlinkStudent(parent.id, student.id)
      parent.students = parent.students.filter((s) => s.id !== student.id)
      showSuccess(`Unlinked ${student.name} from ${parent.name}`)
    } catch (err) {
      showError(err, 'Failed to unlink student')
    } finally {
      busyParentId.value = null
    }
  }
</script>

<template>
  <PageLayout title="Manage Accounts" :show-cart="false">
    <div class="manage-root">
      <ManagementNav />
      <div class="manage-section">
        <div class="manage-section-head">
          <div>
            <h2 class="manage-section-title">Accounts</h2>
            <p class="manage-section-subtitle">
              Accounts are created here only. There is no self-registration.
            </p>
          </div>
          <button type="button" class="manage-btn manage-btn--primary" @click="openModal">
            Create account
          </button>
        </div>

        <div class="account-tabs" role="tablist">
          <button
            type="button"
            role="tab"
            class="account-tab"
            :class="{ 'account-tab--active': activeTab === 'students' }"
            :aria-selected="activeTab === 'students'"
            @click="activeTab = 'students'"
          >
            Students ({{ students.length }})
          </button>
          <button
            type="button"
            role="tab"
            class="account-tab"
            :class="{ 'account-tab--active': activeTab === 'parents' }"
            :aria-selected="activeTab === 'parents'"
            @click="activeTab = 'parents'"
          >
            Parents ({{ parents.length }})
          </button>
        </div>

        <div v-if="isLoading" class="manage-empty">Loading accounts...</div>

        <!-- Students -->
        <template v-else-if="activeTab === 'students'">
          <div v-if="students.length === 0" class="manage-empty">No student accounts yet</div>
          <div v-else-if="isCompact" class="manage-card-list">
            <div v-for="student in students" :key="student.id" class="manage-card">
              <div class="manage-card-header">
                <div class="manage-card-name">{{ student.name }}</div>
                <span class="manage-badge manage-badge--role">{{ student.role }}</span>
              </div>
            </div>
          </div>
          <div v-else class="manage-table-wrap">
            <table class="manage-table">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Role</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="student in students" :key="student.id">
                  <td class="manage-cell-name">{{ student.name }}</td>
                  <td>
                    <span class="manage-badge manage-badge--role">{{ student.role }}</span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </template>

        <!-- Parents -->
        <template v-else>
          <div v-if="parents.length === 0" class="manage-empty">No parent accounts yet</div>
          <div v-else class="manage-card-list">
            <div v-for="parent in parents" :key="parent.id" class="manage-card">
              <div class="manage-card-header">
                <div class="manage-card-name">{{ parent.name }}</div>
                <button
                  type="button"
                  class="manage-btn manage-btn--ghost"
                  @click="toggleParent(parent.id)"
                >
                  {{ expandedParentId === parent.id ? 'Hide' : 'Manage' }}
                  ({{ parent.students.length }})
                </button>
              </div>

              <div v-if="expandedParentId === parent.id" class="account-links">
                <div v-if="parent.students.length === 0" class="account-link-empty">
                  No students linked
                </div>
                <div v-for="student in parent.students" :key="student.id" class="account-link-row">
                  <span class="account-link-name">{{ student.name }}</span>
                  <button
                    type="button"
                    class="manage-btn manage-btn--danger"
                    :disabled="busyParentId === parent.id"
                    @click="handleUnlink(parent, student)"
                  >
                    Unlink
                  </button>
                </div>

                <div class="account-link-add">
                  <select
                    v-model="linkDraft[parent.id]"
                    class="account-select"
                    :aria-label="`Link a student to ${parent.name}`"
                  >
                    <option value="">Select a student</option>
                    <option
                      v-for="student in unlinkedStudentsFor(parent)"
                      :key="student.id"
                      :value="String(student.id)"
                    >
                      {{ student.name }}
                    </option>
                  </select>
                  <button
                    type="button"
                    class="manage-btn manage-btn--primary"
                    :disabled="busyParentId === parent.id || !linkDraft[parent.id]"
                    @click="handleLink(parent)"
                  >
                    Link
                  </button>
                </div>
              </div>
            </div>
          </div>
        </template>
      </div>
    </div>

    <!-- Create account modal -->
    <div v-if="isModalOpen" class="account-modal-backdrop" @click.self="closeModal">
      <div class="account-modal" role="dialog" aria-modal="true" aria-label="Create account">
        <h3 class="account-modal-title">Create account</h3>

        <form class="account-form" novalidate @submit.prevent="handleCreate">
          <div class="account-field">
            <label class="manage-card-label" for="new-name">Username</label>
            <input id="new-name" v-model="form.name" type="text" class="account-input" />
          </div>

          <div class="account-field">
            <label class="manage-card-label" for="new-password">Password</label>
            <input
              id="new-password"
              v-model="form.password"
              type="password"
              autocomplete="new-password"
              class="account-input"
            />
          </div>

          <div class="account-field">
            <label class="manage-card-label" for="new-role">Role</label>
            <select id="new-role" v-model="form.role" class="account-input">
              <option value="student">Student</option>
              <option value="parent">Parent</option>
              <option value="admin">Admin</option>
            </select>
          </div>

          <div v-if="form.role === 'parent'" class="account-field">
            <label class="manage-card-label">Linked students (at least one)</label>
            <div v-if="students.length === 0" class="account-link-empty">
              Create a student account first
            </div>
            <div v-else class="account-checklist">
              <label v-for="student in students" :key="student.id" class="account-check">
                <input
                  type="checkbox"
                  :checked="form.studentIds.includes(student.id)"
                  @change="toggleFormStudent(student.id)"
                />
                <span>{{ student.name }}</span>
              </label>
            </div>
          </div>

          <p v-if="formError" class="account-error" role="alert">{{ formError }}</p>

          <div class="account-modal-actions">
            <button
              type="button"
              class="manage-btn manage-btn--ghost"
              :disabled="isSubmitting"
              @click="closeModal"
            >
              Cancel
            </button>
            <button type="submit" class="manage-btn manage-btn--primary" :disabled="!canSubmit">
              {{ isSubmitting ? 'Creating...' : 'Create' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </PageLayout>
</template>

<style scoped>
  .manage-root {
    width: 100%;
    height: 100%;
  }

  .manage-section {
    padding: 1.5rem;
    max-width: 64rem;
    margin: 0 auto;
  }

  .manage-section-head {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 1rem;
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

  .account-tabs {
    display: flex;
    gap: 0.25rem;
    margin-bottom: 1rem;
  }

  .account-tab {
    padding: 0.375rem 1rem;
    font-family: Inter, sans-serif;
    font-size: 0.8125rem;
    font-weight: 500;
    color: var(--text-secondary);
    background: transparent;
    border: 1px solid var(--border-subtle);
    border-radius: 9999px;
    cursor: pointer;
    transition: all 0.15s;
  }

  .account-tab--active {
    color: #fff;
    background: var(--primary-indigo);
    border-color: var(--primary-indigo);
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

  .manage-card-label {
    font-family: Inter, sans-serif;
    font-size: 0.75rem;
    font-weight: 600;
    letter-spacing: 0.05em;
    text-transform: uppercase;
    color: var(--text-muted);
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

  .manage-btn--ghost {
    color: var(--text-secondary);
    background: transparent;
    border: 1px solid var(--border-subtle);
  }

  .manage-btn--ghost:hover:not(:disabled) {
    background: var(--bg-subtle);
  }

  .manage-btn--danger {
    color: #b91c1c;
    background: transparent;
    border: 1px solid #fecaca;
    padding: 0.375rem 0.875rem;
    font-size: 0.8125rem;
  }

  .manage-btn--danger:hover:not(:disabled) {
    background: #fef2f2;
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

  .manage-badge--role {
    color: var(--text-secondary);
    background: var(--bg-subtle);
    border: 1px solid var(--border-subtle);
  }

  .account-links {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    padding-top: 0.75rem;
    border-top: 1px solid var(--border-subtle);
  }

  .account-link-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 0.5rem;
  }

  .account-link-name {
    font-family: Inter, sans-serif;
    font-size: 0.875rem;
    color: var(--text-primary);
  }

  .account-link-empty {
    font-family: Inter, sans-serif;
    font-size: 0.8125rem;
    color: var(--text-muted);
  }

  .account-link-add {
    display: flex;
    gap: 0.5rem;
    align-items: center;
    margin-top: 0.25rem;
  }

  .account-select,
  .account-input {
    flex: 1;
    padding: 0.5rem 0.75rem;
    font-size: 0.875rem;
    font-family: Inter, sans-serif;
    color: var(--text-primary);
    background: var(--bg-cream);
    border: 1px solid var(--border-medium);
    border-radius: 0.375rem;
    outline: none;
    transition: border-color 0.15s;
  }

  .account-select:focus,
  .account-input:focus {
    border-color: var(--accent-sage);
    box-shadow: 0 0 0 3px rgba(157, 180, 160, 0.15);
  }

  .account-modal-backdrop {
    position: fixed;
    inset: 0;
    z-index: 50;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 1.5rem;
    background: rgba(15, 23, 42, 0.4);
  }

  .account-modal {
    width: 100%;
    max-width: 26rem;
    max-height: 90vh;
    overflow-y: auto;
    padding: 1.5rem;
    background: var(--bg-card);
    border-radius: 0.75rem;
  }

  .account-modal-title {
    margin: 0 0 1rem;
    font-family: 'Instrument Sans', sans-serif;
    font-size: 1.0625rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .account-form {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .account-field {
    display: flex;
    flex-direction: column;
    gap: 0.375rem;
  }

  .account-checklist {
    display: flex;
    flex-direction: column;
    gap: 0.375rem;
    max-height: 10rem;
    overflow-y: auto;
    padding: 0.5rem;
    border: 1px solid var(--border-subtle);
    border-radius: 0.375rem;
  }

  .account-check {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-family: Inter, sans-serif;
    font-size: 0.875rem;
    color: var(--text-primary);
    cursor: pointer;
  }

  .account-error {
    margin: 0;
    font-family: Inter, sans-serif;
    font-size: 0.8125rem;
    color: #b91c1c;
  }

  .account-modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
  }

  @media (max-width: 767px) {
    .manage-section {
      padding: 1rem;
    }

    .manage-section-head {
      flex-direction: column;
      align-items: stretch;
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
