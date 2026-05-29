import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { addDatePrototypes } from 'vue-cal'
import 'vue-cal/style.css'
import './style.css'
import App from './App.vue'
import router from './router'
const pinia = createPinia()
const app = createApp(App)
app.use(pinia)
app.use(router)
app.mount('#app')
