import { apiGet, apiPatch, apiPost } from '@/shared/api/client'
import type {
  CreateSeatsRequest,
  Seat,
  UpdateSeatRequest,
} from '@/shared/api/types'

export type ListSeatsParams = {
  status?: string
  category_id?: string
}

export function listSeats(eventId: string, token: string, params?: ListSeatsParams) {
  const search = new URLSearchParams()
  if (params?.status) search.set('status', params.status)
  if (params?.category_id) search.set('category_id', params.category_id)
  const query = search.toString()
  return apiGet<Seat[]>(`/events/${eventId}/seats${query ? `?${query}` : ''}`, token)
}

export function getSeat(seatId: string, token: string) {
  return apiGet<Seat>(`/seats/${seatId}`, token)
}

export function createSeats(eventId: string, body: CreateSeatsRequest, token: string) {
  return apiPost<Seat[]>(`/events/${eventId}/seats`, body, token)
}

export function updateSeat(seatId: string, body: UpdateSeatRequest, token: string) {
  return apiPatch<Seat>(`/seats/${seatId}`, body, token)
}
