import { ref } from 'vue'

const successMessage = ref('')
const errorMessage = ref('')

let successTimer: ReturnType<typeof setTimeout> | undefined
let errorTimer: ReturnType<typeof setTimeout> | undefined

const showSuccess = (message: string, duration?: number) => {
  clearTimeout(successTimer)
  clearTimeout(errorTimer)
  errorMessage.value = ''
  successMessage.value = message
  if (duration) {
    successTimer = setTimeout(() => {
      successMessage.value = ''
    }, duration)
  }
}

const showError = (error: unknown, message: string, duration?: number) => {
  if (error) console.error(message, error)
  clearTimeout(errorTimer)
  clearTimeout(successTimer)
  successMessage.value = ''
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
