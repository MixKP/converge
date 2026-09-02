import { createRouter, createWebHistory } from 'vue-router'
import GuestView from '../views/GuestView.vue'
import LoginView from '../views/LoginView.vue'
import { useAuthStore } from '../stores/authStore'
import { setUnauthorizedHandler } from '../utils/api'
import { LOGIN_PATH, MY_CLASSES_PATH, resolveRoute, type RouteAccess } from './guard'
import type { Role } from '../types'

declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
    roles?: readonly Role[]
  }
}

const adminOnly: RouteAccess = { requiresAuth: true, roles: ['admin'] }
const familyOnly: RouteAccess = { requiresAuth: true, roles: ['student', 'parent'] }

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'guest',
      component: GuestView,
    },
    {
      path: LOGIN_PATH,
      name: 'login',
      component: LoginView,
    },
    {
      path: MY_CLASSES_PATH,
      name: 'my-classes',
      component: () => import('../views/MyClassesView.vue'),
      meta: familyOnly,
    },
    {
      path: '/form',
      name: 'submit-availability',
      component: () => import('../views/SubmitAvailability.vue'),
      meta: adminOnly,
    },
    {
      path: '/dashboard',
      name: 'admin-dashboard',
      component: () => import('../views/AdminDashboard.vue'),
      meta: adminOnly,
    },
    {
      path: '/booking',
      name: 'booking',
      component: () => import('../views/BookingView.vue'),
      meta: adminOnly,
    },
    {
      path: '/booking/confirm',
      name: 'booking-confirm',
      component: () => import('../views/BookingCart.vue'),
      meta: adminOnly,
    },
    {
      path: '/manage',
      name: 'teacher-management',
      component: () => import('../views/TeacherManagement.vue'),
      meta: adminOnly,
    },
    {
      path: '/manage/branches',
      name: 'branch-management',
      component: () => import('../views/BranchManagement.vue'),
      meta: adminOnly,
    },
    {
      path: '/manage/commute',
      name: 'commute-management',
      component: () => import('../views/CommuteManagement.vue'),
      meta: adminOnly,
    },
    {
      path: '/manage/schedule',
      name: 'schedule-dashboard',
      component: () => import('../views/ScheduleDashboard.vue'),
      meta: adminOnly,
    },
    {
      path: '/manage/accounts',
      name: 'account-management',
      component: () => import('../views/AccountManagement.vue'),
      meta: adminOnly,
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/',
    },
  ],
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  const decision = resolveRoute(
    { path: to.path, fullPath: to.fullPath, access: to.meta },
    { isAuthenticated: auth.isAuthenticated, role: auth.role }
  )
  if (decision.type === 'allow') return true
  return { path: decision.path, query: decision.query ?? {}, replace: true }
})

setUnauthorizedHandler(() => {
  const auth = useAuthStore()
  if (!auth.isAuthenticated) return
  const from = router.currentRoute.value.fullPath
  auth.logout()
  void router.replace({ path: LOGIN_PATH, query: from === LOGIN_PATH ? {} : { redirect: from } })
})

export default router
