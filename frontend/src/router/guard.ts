import type { Role } from '../types'

export const LOGIN_PATH = '/login'
export const GUEST_PATH = '/'
export const ADMIN_HOME = '/manage'

// Type aliases, not interfaces: vue-router's RouteMeta carries an index
// signature, and only an alias is structurally assignable to it.
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

/** Where a signed-in user belongs when they land somewhere they may not go. */
export function homeFor(role: Role | null): string {
  return role === 'admin' ? ADMIN_HOME : GUEST_PATH
}

/**
 * Pure navigation decision. Framework-free and side-effect-free so it can be
 * exercised directly; the vue-router hook is a thin adapter over it.
 */
export function resolveRoute(
  target: { path: string; fullPath: string; access: RouteAccess },
  auth: AuthState
): GuardDecision {
  const { path, fullPath, access } = target

  // A signed-in user has no reason to see the login form again.
  if (path === LOGIN_PATH) {
    return auth.isAuthenticated ? { type: 'redirect', path: homeFor(auth.role) } : ALLOW
  }

  if (!access.requiresAuth) return ALLOW

  if (!auth.isAuthenticated) {
    // Preserve the destination so login can return the user to it. Sending them
    // back to /login would loop, so that case falls back to no query.
    const query = fullPath === LOGIN_PATH ? undefined : { redirect: fullPath }
    return { type: 'redirect', path: LOGIN_PATH, query }
  }

  if (access.roles && (auth.role === null || !access.roles.includes(auth.role))) {
    return { type: 'redirect', path: homeFor(auth.role) }
  }

  return ALLOW
}

/**
 * A `redirect` query value is attacker-controllable, so only same-origin
 * absolute paths are honoured. `//evil.com` and `https://evil.com` are rejected.
 */
export function safeRedirect(raw: unknown, fallback: string): string {
  if (typeof raw !== 'string' || raw === '') return fallback
  if (!raw.startsWith('/') || raw.startsWith('//')) return fallback
  if (raw === LOGIN_PATH) return fallback
  return raw
}
