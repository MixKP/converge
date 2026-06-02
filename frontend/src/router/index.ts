import { createRouter, createWebHistory } from 'vue-router'
import SubmitAvailability from '../views/SubmitAvailability.vue'
import BookingView from '../views/BookingView.vue'
import BookingConfirm from '../views/BookingConfirm.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
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
      path: '/',
      redirect: '/booking',
    },
  ],
})
export default router
