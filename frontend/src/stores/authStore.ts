import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { authApi } from '../services/authApi'
import type { AuthSession, AuthUser, Role } from '../types'
import { clearSession, getSession, persistSession, restoreSession } from '../utils/session'
import { removeItem } from '../utils/storage'

export const useAuthStore = defineStore('auth', () => {
  // Mirrors the module-level session so components get reactivity while
  // utils/api.ts can still read the token without importing this store.
  const session = ref<AuthSession | null>(getSession())

  const sync = (next: AuthSession | null) => {
    session.value = next
  }

  const user = computed<AuthUser | null>(() => session.value?.user ?? null)
  const role = computed<Role | null>(() => session.value?.user.role ?? null)
  const isAuthenticated = computed(() => session.value !== null)
  const isAdmin = computed(() => role.value === 'admin')

  /** Rehydrate from localStorage. Must run before the router is installed. */
  const restore = () => {
    sync(restoreSession())
  }

  const login = async (name: string, password: string) => {
    const next = await authApi.login(name, password)
    persistSession(next)
    sync(next)
    return next
  }

  const logout = () => {
    clearSession()
    sync(null)
    // The cart is per-operator state; leaving it behind would hand the next
    // person to sign in on this machine a half-built booking.
    removeItem('bookingCart')
  }

  return { session, user, role, isAuthenticated, isAdmin, restore, login, logout }
})
