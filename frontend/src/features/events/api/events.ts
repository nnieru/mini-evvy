import { apiGet, apiPatch, apiPost } from '@/shared/api/client'
import type {
  CreateEventRequest,
  Event,
  JobIdResponse,
  MyEvent,
  UpdateEventRequest,
} from '@/shared/api/types'

export function listEvents(orgId: string, token: string) {
  return apiGet<Event[]>(`/orgs/${orgId}/events`, token)
}

export function listMyEvents(token: string) {
  return apiGet<MyEvent[]>('/me/events', token)
}

export function getEvent(eventId: string, token: string) {
  return apiGet<Event>(`/events/${eventId}`, token)
}

export function createEvent(orgId: string, body: CreateEventRequest, token: string) {
  return apiPost<Event>(`/orgs/${orgId}/events`, body, token)
}

export function updateEvent(eventId: string, body: UpdateEventRequest, token: string) {
  return apiPatch<Event>(`/events/${eventId}`, body, token)
}

export function finalizeSeating(eventId: string, token: string) {
  return apiPost<JobIdResponse>(`/events/${eventId}/finalize-seating`, {}, token)
}

export type SeatingPreviewRow = {
  booking_id: string
  guest_name: string
  guest_email: string
  seat_code: string
  category_name: string
  category_code?: string | null
  section?: string | null
}

export function getSeatingPreview(eventId: string, token: string) {
  return apiGet<SeatingPreviewRow[]>(`/events/${eventId}/seating-preview`, token)
}

export function approveSeating(eventId: string, token: string) {
  return apiPost<{ status: string }>(`/events/${eventId}/seating-approve`, {}, token)
}

export function rejectSeating(eventId: string, token: string) {
  return apiPost<{ status: string }>(`/events/${eventId}/seating-reject`, {}, token)
}
