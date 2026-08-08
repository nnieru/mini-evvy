import { useMutation } from '@tanstack/vue-query'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import type { LoginRequest, RegisterRequest } from '@/shared/api/types'

export function useLoginMutation() {
  const auth = useAuthStore()
  const router = useRouter()

  return useMutation({
    mutationFn: (payload: LoginRequest) => auth.login(payload),
    onSuccess: () => router.push('/orgs'),
  })
}

export function useRegisterMutation() {
  const auth = useAuthStore()
  const router = useRouter()

  return useMutation({
    mutationFn: (payload: RegisterRequest) => auth.register(payload),
    onSuccess: () => router.push('/orgs'),
  })
}
