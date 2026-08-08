import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { getMe, login as loginApi, register as registerApi } from '../api/auth'
import type { LoginRequest, RegisterRequest, User } from '@/shared/api/types'

const TOKEN_KEY = 'mini-evvy-token'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem(TOKEN_KEY))
  const user = ref<User | null>(null)
  const initialized = ref(false)

  const isAuthenticated = computed(() => Boolean(token.value))

  function setSession(nextToken: string, nextUser: User) {
    token.value = nextToken
    user.value = nextUser
    localStorage.setItem(TOKEN_KEY, nextToken)
  }

  function clearSession() {
    token.value = null
    user.value = null
    localStorage.removeItem(TOKEN_KEY)
  }

  async function bootstrap() {
    if (!token.value) {
      initialized.value = true
      return
    }
    try {
      user.value = await getMe(token.value)
    } catch {
      clearSession()
    } finally {
      initialized.value = true
    }
  }

  async function login(payload: LoginRequest) {
    const data = await loginApi(payload)
    setSession(data.token!, data.user!)
  }

  async function register(payload: RegisterRequest) {
    const data = await registerApi(payload)
    setSession(data.token!, data.user!)
  }

  function logout() {
    clearSession()
  }

  return {
    token,
    user,
    initialized,
    isAuthenticated,
    bootstrap,
    login,
    register,
    logout,
    clearSession,
  }
})
