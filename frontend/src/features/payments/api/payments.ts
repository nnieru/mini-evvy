import { apiGet, apiPost } from '@/shared/api/client'
import type { CreatePaymentRequest, Payment } from '@/shared/api/types'

export function listPayments(bookingId: string, token: string) {
  return apiGet<Payment[]>(`/bookings/${bookingId}/payments`, token)
}

export function createPayment(bookingId: string, body: CreatePaymentRequest, token: string) {
  return apiPost<Payment>(`/bookings/${bookingId}/payments`, body, token)
}
