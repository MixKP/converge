import { API_BASE } from '../config/api'
import { getToken } from './session'

export class UnauthorizedError extends Error {}
export class ForbiddenError extends Error {}

let onUnauthorized: (() => void) | null = null

export function setUnauthorizedHandler(handler: () => void): void {
  onUnauthorized = handler
}

function withAuth(options?: RequestInit): RequestInit | undefined {
  const token = getToken()
  if (!token) return options
  return {
    ...options,
    headers: { ...(options?.headers ?? {}), Authorization: `Bearer ${token}` },
  }
}

const jsonRequest = (method: string, payload: unknown): RequestInit => ({
  method,
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(payload),
})

export async function fetchApi<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}/${endpoint}`, withAuth(options))
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: response.statusText }))
    const message = error.error || `Failed: ${response.statusText}`
    if (response.status === 401) {
      onUnauthorized?.()
      throw new UnauthorizedError(message)
    }
    if (response.status === 403) {
      throw new ForbiddenError(message)
    }
    throw new Error(message)
  }
  if (response.status === 204) return undefined as T
  return response.json()
}
export async function postApi<T>(endpoint: string, payload: unknown): Promise<T> {
  return fetchApi<T>(endpoint, jsonRequest('POST', payload))
}
export async function patchApi<T>(endpoint: string, payload: unknown): Promise<T> {
  return fetchApi<T>(endpoint, jsonRequest('PATCH', payload))
}
export async function deleteApi<T>(endpoint: string): Promise<T> {
  return fetchApi<T>(endpoint, { method: 'DELETE' })
}
export async function fetchList<T>(endpoint: string, options?: RequestInit): Promise<T[]> {
  const data = await fetchApi<T[] | null>(endpoint, options)
  return data ?? []
}
export function createApiGetAll<T>(endpoint: string): () => Promise<T[]> {
  return () => fetchList<T>(endpoint)
}
