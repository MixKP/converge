import { ref } from 'vue'

// Singleton toast state shared across the whole app. Kept as module-level refs
// (not a Pinia store) so any store, composable, or view can push a toast and
// every page renders from the same source.
const successMessage = ref('')
const errorMessage = ref('')

// Track the pending auto-clear timers so a newer message cancels the older
// timer instead of letting a stale one wipe it early.
let successTimer: ReturnType<typeof setTimeout> | undefined
let errorTimer: ReturnType<typeof setTimeout> | undefined

const showSuccess = (message: string, duration?: number) => {
  clearTimeout(successTimer)
  successMessage.value = message
  if (duration) {
    successTimer = setTimeout(() => {
      successMessage.value = ''
    }, duration)
  }
}

// `error` is optional: pass the caught exception to log it, or null for an
// expected user-facing condition (e.g. a conflict) that shouldn't hit console.
const showError = (error: unknown, message: string, duration?: number) => {
  if (error) console.error(message, error)
  clearTimeout(errorTimer)
  errorMessage.value = message
  if (duration) {
    errorTimer = setTimeout(() => {
      errorMessage.value = ''
    }, duration)
  }
}

const clearMessages = () => {
  clearTimeout(successTimer)
  clearTimeout(errorTimer)
  successMessage.value = ''
  errorMessage.value = ''
}

export function useNotification() {
  return {
    successMessage,
    errorMessage,
    showSuccess,
    showError,
    clearMessages,
  }
}
