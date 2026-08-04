// Runnable check for the pure navigation logic in src/router/guard.ts.
// Run: npm run check:auth   (node --experimental-strip-types, no test framework)
import { homeFor, resolveRoute, safeRedirect, LOGIN_PATH } from '../src/router/guard.ts'

let failures = 0

function expect(label: string, actual: unknown, expected: unknown) {
  const a = JSON.stringify(actual)
  const e = JSON.stringify(expected)
  if (a === e) {
    console.log(`  ok   ${label}`)
  } else {
    failures++
    console.error(`  FAIL ${label}\n         expected ${e}\n         actual   ${a}`)
  }
}

const adminOnly = { requiresAuth: true, roles: ['admin'] } as const
const admin = { isAuthenticated: true, role: 'admin' } as const
const student = { isAuthenticated: true, role: 'student' } as const
const anon = { isAuthenticated: false, role: null }

const target = (path: string, access = {}) => ({ path, fullPath: path, access })

console.log('resolveRoute — happy path')
expect(
  'admin reaches an admin route',
  resolveRoute(target('/manage/accounts', adminOnly), admin),
  { type: 'allow' },
)
expect('anyone reaches a public route', resolveRoute(target('/'), anon), { type: 'allow' })
expect('anonymous reaches the login page', resolveRoute(target(LOGIN_PATH), anon), {
  type: 'allow',
})

console.log('resolveRoute — expired or absent session')
// The critical failure mode: a token that expired mid-session leaves the store
// unauthenticated, and the next navigation must divert to login while keeping
// the destination so the user is returned to it after signing in.
expect(
  'expired session on an admin route redirects to login and preserves the destination',
  resolveRoute(
    { path: '/manage/accounts', fullPath: '/manage/accounts?tab=parents', access: adminOnly },
    anon,
  ),
  { type: 'redirect', path: LOGIN_PATH, query: { redirect: '/manage/accounts?tab=parents' } },
)

console.log('resolveRoute — wrong role')
expect(
  'student on an admin route is sent home, not to login',
  resolveRoute(target('/manage', adminOnly), student),
  { type: 'redirect', path: '/' },
)
expect(
  'signed-in admin is bounced off the login page',
  resolveRoute(target(LOGIN_PATH), admin),
  { type: 'redirect', path: '/manage' },
)

console.log('homeFor')
expect('admin home', homeFor('admin'), '/manage')
expect('student home', homeFor('student'), '/')
expect('unknown role home', homeFor(null), '/')

console.log('safeRedirect — untrusted query value')
expect('same-origin path is kept', safeRedirect('/manage/accounts', '/'), '/manage/accounts')
expect('protocol-relative host is rejected', safeRedirect('//evil.com', '/'), '/')
expect('absolute URL is rejected', safeRedirect('https://evil.com', '/'), '/')
expect('login self-redirect is rejected', safeRedirect(LOGIN_PATH, '/manage'), '/manage')
expect('non-string falls back', safeRedirect(undefined, '/'), '/')

if (failures > 0) {
  console.error(`\n${failures} check(s) failed`)
  process.exit(1)
}
console.log('\nAll auth guard checks passed')
