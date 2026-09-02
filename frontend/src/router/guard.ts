import type { Role } from '../types'

export const LOGIN_PATH = '/login'
export const GUEST_PATH = '/'

export const ADMIN_HOME = '/dashboard'
export const MY_CLASSES_PATH = '/my-classes'

export type RouteAccess = {
  requiresAuth?: boolean
  roles?: readonly Role[]
}

export type AuthState = {
  isAuthenticated: boolean
  role: Role | null
}

export type GuardDecision =
  { type: 'allow' } | { type: 'redirect'; path: string; query?: Record<string, string> }

const ALLOW: GuardDecision = { type: 'allow' }

export function homeFor(role: Role | null): string {
  switch (role) {
    case 'admin':
      return ADMIN_HOME
    case 'student':
    case 'parent':
      return MY_CLASSES_PATH
    default:
      return GUEST_PATH
  }
}

export function resolveRoute(
  target: { path: string; fullPath: string; access: RouteAccess },
  auth: AuthState
): GuardDecision {
  const { path, fullPath, access } = target

  if (path === LOGIN_PATH) {
    return auth.isAuthenticated ? { type: 'redirect', path: homeFor(auth.role) } : ALLOW
  }

  if (!access.requiresAuth) return ALLOW

  if (!auth.isAuthenticated) {
    const query = fullPath === LOGIN_PATH ? undefined : { redirect: fullPath }
    return { type: 'redirect', path: LOGIN_PATH, query }
  }

  if (access.roles && (auth.role === null || !access.roles.includes(auth.role))) {
    return { type: 'redirect', path: homeFor(auth.role) }
  }

  return ALLOW
}

export function safeRedirect(raw: unknown, fallback: string): string {
  if (typeof raw !== 'string' || raw === '') return fallback
  if (!raw.startsWith('/') || raw.startsWith('//')) return fallback
  if (raw === LOGIN_PATH) return fallback
  return raw
}
