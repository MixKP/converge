import { ref, onUnmounted } from 'vue'

/**
 * Composable for managing setTimeout cleanup
 *
 * Prevents memory leaks by tracking and automatically cleaning up
 * timeouts when the component unmounts.
 *
 * @example
 * ```ts
 * const { trackTimeout, clearTimeouts } = useTimeoutCleanup()
 *
 * // Track a timeout that auto-clears on unmount
 * trackTimeout(() => {
 *   successMessage.value = ''
 * }, 3000)
 *
 * // Or clear all tracked timeouts manually
 * clearTimeouts()
 * ```
 */
export function useTimeoutCleanup() {
  const timeoutIds = ref<number[]>([])

  /**
   * Track a timeout and auto-clean it on unmount
   * The timeout ID is tracked and automatically cleared when the component unmounts.
   * After execution, the timeout ID is removed from the tracking array.
   *
   * @param callback - Function to execute after delay
   * @param delay - Delay in milliseconds
   * @returns The timeout ID (useful if you need to clearTimeout manually)
   *
   * @example
   * ```ts
   * // Auto-dismiss success message after 3 seconds
   * trackTimeout(() => {
   *   successMessage.value = ''
   * }, 3000)
   * ```
   */
  const trackTimeout = (callback: () => void, delay: number): number => {
    const id = setTimeout(() => {
      callback()
      // Remove from tracking after execution
      const index = timeoutIds.value.indexOf(id)
      if (index > -1) timeoutIds.value.splice(index, 1)
    }, delay)

    timeoutIds.value.push(id)
    return id
  }

  /**
   * Clear all tracked timeouts immediately
   * Useful for resetting state or preventing pending timeouts from executing
   *
   * @example
   * ```ts
   * // User clicked cancel, prevent any pending timeouts
   * clearTimeouts()
   * ```
   */
  const clearTimeouts = () => {
    timeoutIds.value.forEach((id) => clearTimeout(id))
    timeoutIds.value = []
  }

  // Clean up all timeouts on component unmount
  onUnmounted(() => {
    clearTimeouts()
  })

  return {
    trackTimeout,
    clearTimeouts,
    timeoutIds,
  }
}
