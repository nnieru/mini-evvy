import { apiGet, apiPatch, apiPost } from '@/shared/api/client'
import { ApiError } from '@/shared/lib/errors'
import type { CreateGuestRequest, Guest, UpdateGuestRequest } from '@/shared/api/types'
import type { ErrorEnvelope, SuccessEnvelope } from '@/shared/api/types'

const baseUrl = import.meta.env.VITE_API_BASE_URL ?? '/api'

export type GuestImportResult = {
  created: number
  updated: number
  failed: number
  errors: { row: number; message: string }[]
}

export function listGuests(eventId: string, token: string) {
  return apiGet<Guest[]>(`/events/${eventId}/guests`, token)
}

export function getGuest(guestId: string, token: string) {
  return apiGet<Guest>(`/guests/${guestId}`, token)
}

export function createGuest(eventId: string, body: CreateGuestRequest, token: string) {
  return apiPost<Guest>(`/events/${eventId}/guests`, body, token)
}

export function updateGuest(guestId: string, body: UpdateGuestRequest, token: string) {
  return apiPatch<Guest>(`/guests/${guestId}`, body, token)
}

export async function importGuests(eventId: string, file: File, token: string) {
  const form = new FormData()
  form.append('file', file)

  const response = await fetch(`${baseUrl}/events/${eventId}/guests/import`, {
    method: 'POST',
    headers: {
      Accept: 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: form,
  })

  const payload = (await response.json()) as SuccessEnvelope<GuestImportResult> | ErrorEnvelope
  if (!response.ok) {
    const errorPayload = payload as ErrorEnvelope
    throw new ApiError(
      response.status,
      errorPayload.error?.code ?? 'UNKNOWN_ERROR',
      errorPayload.error?.message ?? response.statusText,
    )
  }
  const envelope = payload as SuccessEnvelope<GuestImportResult>
  return envelope.data
}
