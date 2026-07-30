import { createRouter, createWebHistory } from 'vue-router'
import GuestView from '../views/GuestView.vue'
import SubmitAvailability from '../views/SubmitAvailability.vue'
import BookingView from '../views/BookingView.vue'
import BookingConfirm from '../views/BookingCart.vue'
import TeacherManagement from '../views/TeacherManagement.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'guest',
      component: GuestView,
    },
    {
      path: '/form',
      name: 'submit-availability',
      component: SubmitAvailability,
    },
    {
      path: '/booking',
      name: 'booking',
      component: BookingView,
    },
    {
      path: '/booking/confirm',
      name: 'booking-confirm',
      component: BookingConfirm,
    },
    {
      path: '/manage',
      name: 'teacher-management',
      component: TeacherManagement,
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/',
    },
  ],
})
export default router
