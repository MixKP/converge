// Error handling utilities

/**
 * Extract error message from unknown error type
 * Provides type-safe error message extraction with fallback
 *
 * @param error - The error to extract message from (can be Error, string, or unknown)
 * @param fallback - Default message if error is not an Error instance
 * @returns The error message or fallback string
 *
 * @example
 * ```ts
 * try {
 *   await someOperation()
 * } catch (error) {
 *   const message = getErrorMessage(error, 'Operation failed')
 *   console.error(message) // Safely logs the error message
 * }
 * ```
 */
export function getErrorMessage(error: unknown, fallback: string): string {
  return error instanceof Error ? error.message : fallback
}

/**
 * Check if error is a network error
 * Detects if the error appears to be network-related
 *
 * @param error - The error to check
 * @returns True if the error appears to be network-related
 *
 * @example
 * ```ts
 * catch (error) {
 *   if (isNetworkError(error)) {
 *     showNetworkErrorPrompt()
 *   } else {
 *     showGenericError()
 *   }
 * }
 * ```
 */
export function isNetworkError(error: unknown): boolean {
  if (error instanceof Error) {
    const message = error.message.toLowerCase()
    return (
      message.includes('fetch') ||
      message.includes('network') ||
      message.includes('timeout') ||
      message.includes('connection')
    )
  }
  return false
}
