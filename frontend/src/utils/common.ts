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
