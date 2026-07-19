import { ref } from 'vue'

// Singleton toast state shared across the whole app. Kept as module-level refs
// (not a Pinia store) so any store, composable, or view can push a toast and
// every page renders from the same source.
const successMessage = ref('')
const errorMessage = ref('')

const showSuccess = (message: string, duration?: number) => {
  successMessage.value = message
  if (duration) {
    setTimeout(() => {
      successMessage.value = ''
    }, duration)
  }
}

const showError = (error: unknown, message: string) => {
  console.error(message, error)
  errorMessage.value = message
}

const clearMessages = () => {
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
