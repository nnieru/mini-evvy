import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'
import { createPayment, listPayments } from '../api/payments'
import { useAuthStore } from '@/features/auth/stores/auth'
import type { CreatePaymentRequest } from '@/shared/api/types'

export const paymentKeys = {
  list: (bookingId: string) => ['payments', bookingId] as const,
}

export function usePaymentsQuery(bookingId: Ref<string>) {
  const auth = useAuthStore()
  return useQuery({
    queryKey: computed(() => paymentKeys.list(bookingId.value)),
    queryFn: () => listPayments(bookingId.value, auth.token!),
    enabled: computed(() => Boolean(auth.token && bookingId.value)),
  })
}

export function useCreatePaymentMutation(bookingId: Ref<string>, eventId: Ref<string>) {
  const auth = useAuthStore()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: CreatePaymentRequest) =>
      createPayment(bookingId.value, body, auth.token!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: paymentKeys.list(bookingId.value) })
      queryClient.invalidateQueries({ queryKey: ['bookings', eventId.value] })
    },
  })
}
