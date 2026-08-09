import { apiDelete, apiGet, apiPatch, apiPost } from '@/shared/api/client'
import type {
  Booking,
  CreateBookingBatchRequest,
  CreateBookingRequest,
  JobIdResponse,
  PaginatedBookingList,
  UpdateBookingRequest,
} from '@/shared/api/types'

export type ListBookingsParams = {
  page?: number
  page_size?: number
  payment_status?: 'all' | 'paid' | 'unpaid'
  q?: string
}

export function listBookings(eventId: string, params: ListBookingsParams, token: string) {
  const search = new URLSearchParams()
  if (params.page) search.set('page', String(params.page))
  if (params.page_size) search.set('page_size', String(params.page_size))
  if (params.payment_status) search.set('payment_status', params.payment_status)
  if (params.q) search.set('q', params.q)
  const query = search.toString()
  return apiGet<PaginatedBookingList>(
    `/events/${eventId}/bookings${query ? `?${query}` : ''}`,
    token,
  )
}

export function getBooking(bookingId: string, token: string) {
  return apiGet<Booking>(`/bookings/${bookingId}`, token)
}

export function createBooking(eventId: string, body: CreateBookingRequest, token: string) {
  return apiPost<Booking>(`/events/${eventId}/bookings`, body, token)
}

export function createBookingsBatch(eventId: string, body: CreateBookingBatchRequest, token: string) {
  return apiPost<Booking[]>(`/events/${eventId}/bookings/batch`, body, token)
}

export function updateBooking(bookingId: string, body: UpdateBookingRequest, token: string) {
  return apiPatch<Booking>(`/bookings/${bookingId}`, body, token)
}

export function deleteBooking(bookingId: string, token: string) {
  return apiDelete<void>(`/bookings/${bookingId}`, token)
}

export function resendInvitation(bookingId: string, token: string) {
  return apiPost<JobIdResponse>(`/bookings/${bookingId}/resend-invitation`, {}, token)
}
