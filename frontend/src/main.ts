import { createApp } from 'vue'
import { createPinia } from 'pinia'
import './css/style.css'
import App from './App.vue'
import router from './router'
import { useAuthStore } from './stores/authStore'
const pinia = createPinia()
const app = createApp(App)
app.use(pinia)
// Must run before app.use(router): the first beforeEach fires during install,
// and a store that has not rehydrated yet would bounce a signed-in admin to
// /login on every hard refresh.
useAuthStore(pinia).restore()
app.use(router)
app.mount('#app')
