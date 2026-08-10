import { apiPatch } from '@/shared/api/client'
import {
  approveSeating,
  exportSeatingPreview,
  finalizeSeating,
  getSeatingPreview,
  getSeatingReadiness,
  rejectSeating,
  type ListSeatingPreviewParams,
  type SeatingReadiness,
  type SeatingPreviewRow,
} from '@/features/events/api/events'

export type { ListSeatingPreviewParams, SeatingPreviewRow, SeatingReadiness }

export {
  approveSeating,
  exportSeatingPreview,
  finalizeSeating,
  getSeatingPreview,
  getSeatingReadiness,
  rejectSeating,
}

export function reassignDraftItem(
  eventId: string,
  itemId: string,
  seatId: string,
  token: string,
) {
  return apiPatch<{ status: string }>(
    `/events/${eventId}/seating-draft/items/${itemId}`,
    { seat_id: seatId },
    token,
  )
}
