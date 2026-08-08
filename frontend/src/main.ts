import { VueQueryPlugin } from '@tanstack/vue-query'
import { createPinia } from 'pinia'
import { createApp } from 'vue'
import '@fontsource/dm-sans/500.css'
import '@fontsource/dm-sans/600.css'
import '@fontsource/instrument-sans/400.css'
import '@fontsource/instrument-sans/500.css'
import '@fontsource/instrument-sans/600.css'
import App from './App.vue'
import router from './app/router'
import { setUnauthorizedHandler } from './shared/api/client'
import { useAuthStore } from './features/auth/stores/auth'
import './style.css'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)
app.use(VueQueryPlugin, {
  queryClientConfig: {
    defaultOptions: {
      queries: {
        staleTime: 30_000,
        retry: 1,
      },
    },
  },
})

const auth = useAuthStore()
setUnauthorizedHandler(() => {
  auth.clearSession()
  if (router.currentRoute.value.meta.requiresAuth) {
    router.push({ name: 'login' })
  }
})

app.mount('#app')
