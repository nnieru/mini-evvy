import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'
import {
  createGuest,
  getUnbookedGuestCount,
  importGuests,
  listGuests,
  listGuestsAll,
  updateGuest,
  type ListGuestsParams,
} from '../api/guests'
import { useAuthStore } from '@/features/auth/stores/auth'
import { seatingDraftKeys } from '@/features/seating-draft/queries/seatingDraft'
import type { CreateGuestRequest, UpdateGuestRequest } from '@/shared/api/types'

export const guestKeys = {
  list: (eventId: string, params?: ListGuestsParams) => ['guests', eventId, params ?? null] as const,
  all: (eventId: string) => ['guests', eventId, 'all'] as const,
  unbookedCount: (eventId: string) => ['guests', eventId, 'unbooked-count'] as const,
}

function invalidateGuestDerivedQueries(
  queryClient: ReturnType<typeof useQueryClient>,
  eventId: string,
) {
  queryClient.invalidateQueries({ queryKey: ['guests', eventId] })
  queryClient.invalidateQueries({ queryKey: guestKeys.unbookedCount(eventId) })
  queryClient.invalidateQueries({ queryKey: seatingDraftKeys.readiness(eventId) })
}

export function useGuestsQuery(eventId: Ref<string>, filters?: Ref<ListGuestsParams>) {
  const auth = useAuthStore()
  return useQuery({
    queryKey: computed(() => guestKeys.list(eventId.value, filters?.value)),
    queryFn: () => listGuests(eventId.value, auth.token!, filters?.value),
    enabled: computed(() => Boolean(auth.token && eventId.value)),
  })
}

export function useUnbookedGuestCountQuery(eventId: Ref<string>) {
  const auth = useAuthStore()
  return useQuery({
    queryKey: computed(() => guestKeys.unbookedCount(eventId.value)),
    queryFn: () => getUnbookedGuestCount(eventId.value, auth.token!),
    enabled: computed(() => Boolean(auth.token && eventId.value)),
  })
}

export function useAllGuestsQuery(eventId: Ref<string>) {
  const auth = useAuthStore()
  return useQuery({
    queryKey: computed(() => guestKeys.all(eventId.value)),
    queryFn: () => listGuestsAll(eventId.value, auth.token!),
    enabled: computed(() => Boolean(auth.token && eventId.value)),
  })
}

export function useCreateGuestMutation(eventId: Ref<string>) {
  const auth = useAuthStore()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: CreateGuestRequest) => createGuest(eventId.value, body, auth.token!),
    onSuccess: () => {
      invalidateGuestDerivedQueries(queryClient, eventId.value)
    },
  })
}

export function useImportGuestsMutation(eventId: Ref<string>) {
  const auth = useAuthStore()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (file: File) => importGuests(eventId.value, file, auth.token!),
    onSuccess: () => {
      invalidateGuestDerivedQueries(queryClient, eventId.value)
    },
  })
}

export function useUpdateGuestMutation(eventId: Ref<string>) {
  const auth = useAuthStore()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ guestId, body }: { guestId: string; body: UpdateGuestRequest }) =>
      updateGuest(guestId, body, auth.token!),
    onSuccess: () => {
      invalidateGuestDerivedQueries(queryClient, eventId.value)
    },
  })
}
