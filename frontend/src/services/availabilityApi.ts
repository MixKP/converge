import type { AvailabilityPayload } from '../types/calendar'
import { API_BASE } from '../config/api'

export const availabilityApi = {
  async submitAvailability(payload: AvailabilityPayload): Promise<void> {
    const response = await fetch(`${API_BASE}/availability`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
    if (!response.ok) {
      let message = 'Failed to submit availability'
      try {
        const err = await response.json()
        message = err.error || err.message || message
      } catch {
        const text = await response.text()
        if (text.trim()) message = text
      }
      throw new Error(`${message} (${response.status} ${response.statusText})`)
    }
  },
}
