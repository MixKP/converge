import { API_BASE } from '../config/api'
export async function fetchApi<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}/${endpoint}`, options)
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: response.statusText }))
    throw new Error(error.error || `Failed: ${response.statusText}`)
  }
  return response.json()
}
export async function postApi<T>(endpoint: string, payload: unknown): Promise<T> {
  return fetchApi<T>(endpoint, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
}
export function createApiGetAll<T>(endpoint: string): () => Promise<T[]> {
  return () => fetchApi<T[]>(endpoint)
}
