import { ref } from 'vue'
import { commuteApi } from '../services/commuteApi'

const commuteMinutes = ref<number | null>(null)
let inFlight: Promise<void> | null = null

export function useCommute() {
  const loadCommuteMinutes = (): Promise<void> => {
    if (commuteMinutes.value !== null) return Promise.resolve()
    if (inFlight) return inFlight

    inFlight = commuteApi
      .get()
      .then((res) => {
        commuteMinutes.value = res.commute_time
      })
      .catch(() => {
        commuteMinutes.value = 0
      })
      .finally(() => {
        inFlight = null
      })

    return inFlight
  }

  return { commuteMinutes, loadCommuteMinutes }
}
