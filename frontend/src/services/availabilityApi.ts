import type { AvailabilityPayload } from '../types/calendar'

const API_BASE = 'http://localhost:8080/api'

export const availabilityApi = {
  async submitAvailability(payload: AvailabilityPayload): Promise<void> {
    const response = await fetch(`${API_BASE}/availability`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
    if (!response.ok) {
      const error = await response.json()
      throw new Error(error.error || 'Failed to submit availability')
    }
  },
}
