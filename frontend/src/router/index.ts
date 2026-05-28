import { createRouter, createWebHistory } from 'vue-router'
import SubmitAvailability from '../views/SubmitAvailability.vue'
import BookingView from '../views/BookingView.vue'

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
      path: '/',
      redirect: '/form',
    },
  ],
})

export default router
