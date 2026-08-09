import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'
import {
  createGuest,
  importGuests,
  listGuests,
  listGuestsAll,
  updateGuest,
  type ListGuestsParams,
} from '../api/guests'
import { useAuthStore } from '@/features/auth/stores/auth'
import type { CreateGuestRequest, UpdateGuestRequest } from '@/shared/api/types'

export const guestKeys = {
  list: (eventId: string, params?: ListGuestsParams) => ['guests', eventId, params ?? null] as const,
  all: (eventId: string) => ['guests', eventId, 'all'] as const,
}

export function useGuestsQuery(eventId: Ref<string>, filters?: Ref<ListGuestsParams>) {
  const auth = useAuthStore()
  return useQuery({
    queryKey: computed(() => guestKeys.list(eventId.value, filters?.value)),
    queryFn: () => listGuests(eventId.value, auth.token!, filters?.value),
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
      queryClient.invalidateQueries({ queryKey: ['guests', eventId.value] })
    },
  })
}

export function useImportGuestsMutation(eventId: Ref<string>) {
  const auth = useAuthStore()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (file: File) => importGuests(eventId.value, file, auth.token!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['guests', eventId.value] })
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
      queryClient.invalidateQueries({ queryKey: ['guests', eventId.value] })
    },
  })
}
