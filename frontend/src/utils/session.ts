import { ref } from 'vue'
import { authSessionSchema } from '../schemas/auth'
import type { AuthSession } from '../types'
import { getValidated, removeItem, setItem } from './storage'

export const AUTH_STORAGE_KEY = 'converge.auth'

const session = ref<AuthSession | null>(null)

const isAuthSession = (value: unknown): value is AuthSession =>
  authSessionSchema.safeParse(value).success

export function getSession(): AuthSession | null {
  return session.value
}

export function getToken(): string | null {
  return session.value?.token ?? null
}

export function restoreSession(): AuthSession | null {
  const stored = getValidated<AuthSession>(AUTH_STORAGE_KEY, isAuthSession)
  session.value = stored
  if (!stored) removeItem(AUTH_STORAGE_KEY)
  return stored
}

export function persistSession(next: AuthSession): void {
  session.value = next
  setItem(AUTH_STORAGE_KEY, next)
}

export function clearSession(): void {
  session.value = null
  removeItem(AUTH_STORAGE_KEY)
}
