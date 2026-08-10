import { apiGet, apiPatch, apiPost } from '@/shared/api/client'
import { ApiError } from '@/shared/lib/errors'
import { appendListPageParams, type ListPageParams, type PaginatedList } from '@/shared/api/pagination'
import type {
  CreateEventRequest,
  Event,
  JobIdResponse,
  MyEvent,
  UpdateEventRequest,
} from '@/shared/api/types'

const baseUrl = import.meta.env.VITE_API_BASE_URL ?? '/api'

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

export type CategoryCapacityShortfall = {
  category_id: string
  slots_needed: number
  seats_available: number
}

export type SeatingReadiness = {
  unbooked_guests: number
  slots_needed: number
  can_assign_any: boolean
  shortfalls: CategoryCapacityShortfall[]
}

export function getSeatingReadiness(eventId: string, token: string) {
  return apiGet<SeatingReadiness>(`/events/${eventId}/seating-readiness`, token)
}

export type SeatingPreviewRow = {
  id: string
  guest_name: string
  guest_email: string
  seat_code: string
  seat_id: string
  category_name: string
  category_code?: string | null
  section?: string | null
}

export type ListSeatingPreviewParams = ListPageParams

export function getSeatingPreview(
  eventId: string,
  token: string,
  params?: ListSeatingPreviewParams,
) {
  const search = new URLSearchParams()
  appendListPageParams(search, params)
  const query = search.toString()
  return apiGet<PaginatedList<SeatingPreviewRow>>(
    `/events/${eventId}/seating-preview${query ? `?${query}` : ''}`,
    token,
  )
}

export async function exportSeatingPreview(
  eventId: string,
  token: string,
  q?: string,
) {
  const search = new URLSearchParams()
  if (q) search.set('q', q)
  const query = search.toString()

  const response = await fetch(
    `${baseUrl}/events/${eventId}/seating-preview/export${query ? `?${query}` : ''}`,
    {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    },
  )

  if (!response.ok) {
    let message = response.statusText
    try {
      const payload = (await response.json()) as { error?: { message?: string } }
      message = payload.error?.message ?? message
    } catch {
      // ignore non-json error bodies
    }
    throw new ApiError(response.status, 'EXPORT_FAILED', message)
  }

  const blob = await response.blob()
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `seating-preview-${eventId}.xlsx`
  link.click()
  URL.revokeObjectURL(url)
}

export function approveSeating(eventId: string, token: string) {
  return apiPost<{ status: string }>(`/events/${eventId}/seating-approve`, {}, token)
}

export function rejectSeating(eventId: string, token: string) {
  return apiPost<{ status: string }>(`/events/${eventId}/seating-reject`, {}, token)
}

export type ImportEventConfigRequest = {
  source_event_id: string
  include_categories: boolean
  include_seats: boolean
  include_email_template: boolean
}

export type ImportEventConfigResult = {
  categories_created: number
  seats_created: number
  email_template_copied: boolean
}

export function importEventConfig(
  targetEventId: string,
  body: ImportEventConfigRequest,
  token: string,
) {
  return apiPost<ImportEventConfigResult>(
    `/events/${targetEventId}/import-config`,
    body,
    token,
  )
}
