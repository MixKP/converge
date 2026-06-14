import { ref } from 'vue'
import { useTimeoutCleanup } from './useTimeoutCleanup'
import { getErrorMessage } from '../utils/errorHandler'

/**
 * Composable for managing success and error messages with auto-dismiss
 *
 * Provides reactive message state and methods to show messages that
 * automatically dismiss after a specified duration.
 *
 * @example
 * ```ts
 * const { successMessage, errorMessage, showSuccess, showError, clearMessages } = useMessages()
 *
 * // Show success message (auto-dismisses after 3 seconds)
 * showSuccess('Operation completed successfully!')
 *
 * // Show error message (auto-dismisses after 5 seconds)
 * showError(new Error('Something went wrong'), 'Operation failed')
 *
 * // Custom duration
 * showSuccess('Saved!', 2000)
 *
 * // Clear all messages immediately
 * clearMessages()
 * ```
 */
export function useMessages() {
  const successMessage = ref<string>('')
  const errorMessage = ref<string>('')
  const { trackTimeout } = useTimeoutCleanup()

  /**
   * Show a success message that auto-dismisses
   *
   * @param message - The message to display
   * @param duration - How long to show the message in milliseconds (default: 3000ms)
   *
   * @example
   * ```ts
   * showSuccess('Item added to cart!')
   * showSuccess('Changes saved', 5000) // Custom 5 second duration
   * ```
   */
  const showSuccess = (message: string, duration: number = 3000) => {
    successMessage.value = message
    trackTimeout(() => {
      successMessage.value = ''
    }, duration)
  }

  /**
   * Show an error message that auto-dismisses
   * Safely extracts error message from Error objects or uses fallback
   *
   * @param error - The error (Error instance, string, or unknown)
   * @param fallback - Default message if error is not an Error instance
   * @param duration - How long to show the message in milliseconds (default: 5000ms)
   *
   * @example
   * ```ts
   * try {
   *   await apiCall()
   * } catch (error) {
   *   showError(error, 'Failed to complete operation')
   * }
   *
   * // Network errors might need longer display time
   * showError(networkError, 'Connection failed', 10000)
   * ```
   */
  const showError = (error: unknown, fallback: string, duration: number = 5000) => {
    const message = getErrorMessage(error, fallback)
    errorMessage.value = message
    trackTimeout(() => {
      errorMessage.value = ''
    }, duration)
  }

  /**
   * Clear all messages immediately
   * Useful for manual dismissal or state reset
   *
   * @example
   * ```ts
   * function handleCancel() {
   *   clearMessages()
   *   // Reset other state...
   * }
   * ```
   */
  const clearMessages = () => {
    successMessage.value = ''
    errorMessage.value = ''
  }

  /**
   * Check if any message is currently displayed
   * @returns True if either success or error message is active
   */
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
