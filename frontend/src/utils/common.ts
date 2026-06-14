// Common utility functions

/**
 * Debounce a function call
 * Delays the function execution until after wait milliseconds have elapsed
 * since the last time the debounced function was invoked
 *
 * @param func - The function to debounce
 * @param wait - The delay in milliseconds
 * @returns A debounced version of the function
 *
 * @example
 * ```ts
 * const debouncedSearch = debounce((query: string) => {
 *   performSearch(query)
 * }, 300)
 *
 * // Will only execute once, 300ms after the last call
 * debouncedSearch('hello')
 * debouncedSearch('hello world')
 * debouncedSearch('hello world!')
 * ```
 */
export function debounce<T extends (...args: any[]) => any>(func: T, wait: number): T {
  let timeoutId: number | null = null
  return ((...args: Parameters<T>) => {
    if (timeoutId !== null) {
      clearTimeout(timeoutId)
    }
    timeoutId = setTimeout(() => {
      func(...args)
      timeoutId = null
    }, wait)
  }) as T
}
