import { ref } from 'vue'
import { useTimeoutCleanup } from './useTimeoutCleanup'
import { getErrorMessage } from '../utils/errorHandler'

export function useMessages() {
  const successMessage = ref<string>('')
  const errorMessage = ref<string>('')
  const { trackTimeout } = useTimeoutCleanup()

  const showSuccess = (message: string, duration: number = 3000) => {
    successMessage.value = message
    trackTimeout(() => {
      successMessage.value = ''
    }, duration)
  }

  const showError = (error: unknown, fallback: string, duration: number = 5000) => {
    const message = getErrorMessage(error, fallback)
    errorMessage.value = message
    trackTimeout(() => {
      errorMessage.value = ''
    }, duration)
  }

  const clearMessages = () => {
    successMessage.value = ''
    errorMessage.value = ''
  }

  const hasActiveMessage = () => {
    return successMessage.value !== '' || errorMessage.value !== ''
  }

  return {
    successMessage,
    errorMessage,
    showSuccess,
    showError,
    clearMessages,
    hasActiveMessage,
  }
}
