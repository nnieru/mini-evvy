import { apiGet, apiPost } from '@/shared/api/client'
import type { AuthResponse, LoginRequest, RegisterRequest, User } from '@/shared/api/types'

export function register(body: RegisterRequest) {
  return apiPost<AuthResponse>('/auth/register', body, null)
}

export function login(body: LoginRequest) {
  return apiPost<AuthResponse>('/auth/login', body, null)
}

export function getMe(token: string) {
  return apiGet<User>('/me', token)
}
