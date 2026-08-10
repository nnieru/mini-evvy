import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'
import {
  approveSeating,
  finalizeSeating,
  getSeatingPreview,
  getSeatingReadiness,
  reassignDraftItem,
  rejectSeating,
  type ListSeatingPreviewParams,
} from '../api/seatingDraft'
import { useAuthStore } from '@/features/auth/stores/auth'
import { eventKeys } from '@/features/events/queries/events'
import { guestKeys } from '@/features/guests/queries/guests'

export const seatingDraftKeys = {
  preview: (eventId: string, params?: ListSeatingPreviewParams) =>
    ['seating-draft', 'preview', eventId, params ?? null] as const,
  readiness: (eventId: string) => ['seating-draft', 'readiness', eventId] as const,
}

export function useSeatingReadinessQuery(eventId: Ref<string>, enabled: Ref<boolean>) {
  const auth = useAuthStore()
  return useQuery({
    queryKey: computed(() => seatingDraftKeys.readiness(eventId.value)),
    queryFn: () => getSeatingReadiness(eventId.value, auth.token!),
    enabled: computed(() => Boolean(auth.token && eventId.value && enabled.value)),
  })
}

export function useFinalizeSeatingMutation(eventId: Ref<string>) {
  const auth = useAuthStore()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => finalizeSeating(eventId.value, auth.token!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: eventKeys.detail(eventId.value) })
      queryClient.invalidateQueries({ queryKey: seatingDraftKeys.readiness(eventId.value) })
    },
  })
}

export function useSeatingDraftPreviewQuery(
  eventId: Ref<string>,
  enabled: Ref<boolean>,
  filters?: Ref<ListSeatingPreviewParams>,
) {
  const auth = useAuthStore()
  return useQuery({
    queryKey: computed(() => seatingDraftKeys.preview(eventId.value, filters?.value)),
    queryFn: () => getSeatingPreview(eventId.value, auth.token!, filters?.value),
    enabled: computed(() => Boolean(auth.token && eventId.value && enabled.value)),
    retry: false,
  })
}

export function useApproveSeatingMutation(eventId: Ref<string>) {
  const auth = useAuthStore()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => approveSeating(eventId.value, auth.token!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: eventKeys.detail(eventId.value) })
      queryClient.removeQueries({ queryKey: seatingDraftKeys.preview(eventId.value) })
      queryClient.invalidateQueries({ queryKey: ['bookings', eventId.value] })
      queryClient.invalidateQueries({ queryKey: ['seats', eventId.value] })
      queryClient.invalidateQueries({ queryKey: ['guests', eventId.value] })
      queryClient.invalidateQueries({ queryKey: seatingDraftKeys.readiness(eventId.value) })
    },
  })
}

export function useRejectSeatingMutation(eventId: Ref<string>) {
  const auth = useAuthStore()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => rejectSeating(eventId.value, auth.token!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: eventKeys.detail(eventId.value) })
      queryClient.removeQueries({ queryKey: seatingDraftKeys.preview(eventId.value) })
      queryClient.invalidateQueries({ queryKey: guestKeys.list(eventId.value) })
      queryClient.invalidateQueries({ queryKey: guestKeys.all(eventId.value) })
      queryClient.invalidateQueries({ queryKey: guestKeys.unbookedCount(eventId.value) })
      queryClient.invalidateQueries({ queryKey: seatingDraftKeys.readiness(eventId.value) })
      queryClient.invalidateQueries({ queryKey: ['bookings', eventId.value] })
      queryClient.invalidateQueries({ queryKey: ['seats', eventId.value] })
    },
  })
}

export function useReassignDraftItemMutation(eventId: Ref<string>) {
  const auth = useAuthStore()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ itemId, seatId }: { itemId: string; seatId: string }) =>
      reassignDraftItem(eventId.value, itemId, seatId, auth.token!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: seatingDraftKeys.preview(eventId.value) })
      queryClient.invalidateQueries({ queryKey: ['seats', eventId.value] })
    },
  })
}
