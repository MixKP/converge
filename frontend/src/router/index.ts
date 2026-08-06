import { createRouter, createWebHistory } from 'vue-router'
import GuestView from '../views/GuestView.vue'
import LoginView from '../views/LoginView.vue'
import SubmitAvailability from '../views/SubmitAvailability.vue'
import BookingView from '../views/BookingView.vue'
import BookingConfirm from '../views/BookingCart.vue'
import TeacherManagement from '../views/TeacherManagement.vue'
import BranchManagement from '../views/BranchManagement.vue'
import CommuteManagement from '../views/CommuteManagement.vue'
import AccountManagement from '../views/AccountManagement.vue'
import MyClassesView from '../views/MyClassesView.vue'
import { useAuthStore } from '../stores/authStore'
import { setUnauthorizedHandler } from '../utils/api'
import { LOGIN_PATH, MY_CLASSES_PATH, resolveRoute, type RouteAccess } from './guard'
import type { Role } from '../types'

// guard.ts stays framework-free, so the vue-router contract is declared here.
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
      component: MyClassesView,
      meta: familyOnly,
    },
    {
      path: '/form',
      name: 'submit-availability',
      component: SubmitAvailability,
      meta: adminOnly,
    },
    {
      path: '/booking',
      name: 'booking',
      component: BookingView,
      meta: adminOnly,
    },
    {
      path: '/booking/confirm',
      name: 'booking-confirm',
      component: BookingConfirm,
      meta: adminOnly,
    },
    {
      path: '/manage',
      name: 'teacher-management',
      component: TeacherManagement,
      meta: adminOnly,
    },
    {
      path: '/manage/branches',
      name: 'branch-management',
      component: BranchManagement,
      meta: adminOnly,
    },
    {
      path: '/manage/commute',
      name: 'commute-management',
      component: CommuteManagement,
      meta: adminOnly,
    },
    {
      path: '/manage/accounts',
      name: 'account-management',
      component: AccountManagement,
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

// A token can expire mid-session, so the 401 on the next request is the
// authoritative signal to end it rather than a client-side expiry timer.
setUnauthorizedHandler(() => {
  const auth = useAuthStore()
  if (!auth.isAuthenticated) return
  const from = router.currentRoute.value.fullPath
  auth.logout()
  void router.replace({ path: LOGIN_PATH, query: from === LOGIN_PATH ? {} : { redirect: from } })
})

export default router
