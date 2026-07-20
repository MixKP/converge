import { ref, onUnmounted } from 'vue'

export function useTimeoutCleanup() {
  const timeoutIds = ref<number[]>([])

  const trackTimeout = (callback: () => void, delay: number): number => {
    const id = setTimeout(() => {
      callback()
      const index = timeoutIds.value.indexOf(id)
      if (index > -1) timeoutIds.value.splice(index, 1)
    }, delay)

    timeoutIds.value.push(id)
    return id
  }

  const clearTimeouts = () => {
    timeoutIds.value.forEach((id) => clearTimeout(id))
    timeoutIds.value = []
  }

  onUnmounted(() => {
    clearTimeouts()
  })

  return {
    trackTimeout,
    clearTimeouts,
    timeoutIds,
  }
}
