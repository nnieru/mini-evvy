import { apiPatch } from '@/shared/api/client'
import {
  approveSeating,
  exportSeatingPreview,
  finalizeSeating,
  getSeatingPreview,
  rejectSeating,
  type ListSeatingPreviewParams,
  type SeatingPreviewRow,
} from '@/features/events/api/events'

export type { ListSeatingPreviewParams, SeatingPreviewRow }

export {
  approveSeating,
  exportSeatingPreview,
  finalizeSeating,
  getSeatingPreview,
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
