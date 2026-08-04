<script setup lang="ts">
  import { ref, computed } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import { useAuthStore } from '../stores/authStore'
  import { loginRequestSchema } from '../schemas/auth'
  import { getErrorMessage } from '../utils/errorHandler'
  import { homeFor, safeRedirect } from '../router/guard'

  const route = useRoute()
  const router = useRouter()
  const auth = useAuthStore()

  const name = ref('')
  const password = ref('')
  const fieldError = ref('')
  const formError = ref('')
  const isSubmitting = ref(false)

  const canSubmit = computed(
    () => !isSubmitting.value && name.value.trim() !== '' && password.value !== ''
  )

  const handleSubmit = async () => {
    fieldError.value = ''
    formError.value = ''

    const parsed = loginRequestSchema.safeParse({ name: name.value, password: password.value })
    if (!parsed.success) {
      fieldError.value = parsed.error.issues[0]?.message ?? 'Please check your details'
      return
    }

    isSubmitting.value = true
    try {
      const session = await auth.login(parsed.data.name, parsed.data.password)
      const fallback = homeFor(session.user.role)
      await router.replace(safeRedirect(route.query.redirect, fallback))
    } catch (err) {
      // The backend deliberately returns one generic message for both an unknown
      // username and a wrong password. Surface it verbatim rather than guessing.
      formError.value = getErrorMessage(err, 'Unable to sign in. Please try again.')
      password.value = ''
    } finally {
      isSubmitting.value = false
    }
  }
</script>

<template>
  <div class="login-page">
    <div class="login-card">
      <div class="login-header">
        <div class="login-indicator"></div>
        <h1 class="login-title">Converge</h1>
      </div>
      <p class="login-subtitle">Sign in to continue</p>

      <form class="login-form" novalidate @submit.prevent="handleSubmit">
        <div class="login-field">
          <label class="login-label" for="login-name">Username</label>
          <input
            id="login-name"
            v-model="name"
            type="text"
            autocomplete="username"
            class="login-input"
            :disabled="isSubmitting"
          />
        </div>

        <div class="login-field">
          <label class="login-label" for="login-password">Password</label>
          <input
            id="login-password"
            v-model="password"
            type="password"
            autocomplete="current-password"
            class="login-input"
            :disabled="isSubmitting"
          />
        </div>

        <p v-if="fieldError" class="login-error" role="alert">{{ fieldError }}</p>
        <p v-if="formError" class="login-error" role="alert">{{ formError }}</p>

        <button type="submit" class="login-btn" :disabled="!canSubmit">
          {{ isSubmitting ? 'Signing in...' : 'Sign in' }}
        </button>
      </form>

      <p class="login-note">
        Accounts are created by an administrator. Contact your admin if you need access.
      </p>
    </div>
  </div>
</template>

<style scoped>
  .login-page {
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 1.5rem;
    background: #fafafa;
  }

  .login-card {
    width: 100%;
    max-width: 24rem;
    padding: 2rem;
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: 0.75rem;
  }

  .login-header {
    display: flex;
    align-items: center;
    gap: 0.625rem;
  }

  .login-indicator {
    width: 0.5rem;
    height: 1.5rem;
    border-radius: 9999px;
    background: var(--primary-indigo);
  }

  .login-title {
    margin: 0;
    font-family: 'Instrument Sans', sans-serif;
    font-size: 1.375rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .login-subtitle {
    margin: 0.5rem 0 1.5rem;
    font-family: Inter, sans-serif;
    font-size: 0.8125rem;
    color: var(--text-secondary);
  }

  .login-form {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .login-field {
    display: flex;
    flex-direction: column;
    gap: 0.375rem;
  }

  .login-label {
    font-family: Inter, sans-serif;
    font-size: 0.75rem;
    font-weight: 600;
    letter-spacing: 0.05em;
    text-transform: uppercase;
    color: var(--text-muted);
  }

  .login-input {
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

  .login-input:focus {
    border-color: var(--accent-sage);
    box-shadow: 0 0 0 3px rgba(157, 180, 160, 0.15);
  }

  .login-input:disabled {
    opacity: 0.6;
  }

  .login-error {
    margin: 0;
    font-family: Inter, sans-serif;
    font-size: 0.8125rem;
    color: #b91c1c;
  }

  .login-btn {
    margin-top: 0.25rem;
    padding: 0.625rem 1.25rem;
    font-size: 0.875rem;
    font-family: Inter, sans-serif;
    font-weight: 500;
    color: #fff;
    background: var(--primary-indigo);
    border: none;
    border-radius: 0.5rem;
    cursor: pointer;
    transition: all 0.15s;
  }

  .login-btn:hover:not(:disabled) {
    opacity: 0.9;
  }

  .login-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .login-note {
    margin: 1.5rem 0 0;
    font-family: Inter, sans-serif;
    font-size: 0.75rem;
    color: var(--text-muted);
    text-align: center;
  }

  @media (max-width: 767px) {
    .login-page {
      background: #ffffff;
      padding: 1rem;
    }

    .login-card {
      border: none;
      padding: 1.5rem 0;
    }
  }
</style>
