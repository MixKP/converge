import { createApp } from 'vue'
import { createPinia } from 'pinia'
import './css/style.css'
import App from './App.vue'
import router from './router'
import { useAuthStore } from './stores/authStore'
const pinia = createPinia()
const app = createApp(App)
app.use(pinia)
useAuthStore(pinia).restore()
app.use(router)
app.mount('#app')
