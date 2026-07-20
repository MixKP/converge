import { API_BASE } from '../config/api'
export async function fetchApi<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}/${endpoint}`, options)
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: response.statusText }))
    throw new Error(error.error || `Failed: ${response.statusText}`)
  }
  if (response.status === 204) return undefined as T
  return response.json()
}
export async function postApi<T>(endpoint: string, payload: unknown): Promise<T> {
  return fetchApi<T>(endpoint, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
}
// Fetch a list endpoint, coalescing a null/absent body (e.g. a backend nil
// slice serialized as JSON null) to an empty array so callers can safely map/filter.
export async function fetchList<T>(endpoint: string, options?: RequestInit): Promise<T[]> {
  const data = await fetchApi<T[] | null>(endpoint, options)
  return data ?? []
}
export function createApiGetAll<T>(endpoint: string): () => Promise<T[]> {
  return () => fetchList<T>(endpoint)
}
