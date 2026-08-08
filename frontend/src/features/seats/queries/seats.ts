import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'
import { createSeats, listSeats, updateSeat, type ListSeatsParams } from '../api/seats'
import { useAuthStore } from '@/features/auth/stores/auth'
import type { CreateSeatsRequest, UpdateSeatRequest } from '@/shared/api/types'

export const seatKeys = {
  list: (eventId: string, filters?: ListSeatsParams) =>
    ['seats', eventId, filters?.status ?? null, filters?.category_id ?? null] as const,
}

export function useSeatsQuery(eventId: Ref<string>, filters?: Ref<ListSeatsParams | undefined>) {
  const auth = useAuthStore()
  return useQuery({
    queryKey: computed(() =>
      seatKeys.list(eventId.value, filters?.value),
    ),
    queryFn: () => listSeats(eventId.value, auth.token!, filters?.value),
    enabled: computed(() => {
      if (!auth.token || !eventId.value) return false
      if (!filters) return true
      if (filters.value === undefined) return false
      if (filters.value.category_id === '') return false
      return true
    }),
  })
}

export function useAvailableSeatsQuery(eventId: Ref<string>, categoryId: Ref<string>) {
  const auth = useAuthStore()
  return useQuery({
    queryKey: computed(() => seatKeys.list(eventId.value, {
      status: 'available',
      category_id: categoryId.value,
    })),
    queryFn: () =>
      listSeats(eventId.value, auth.token!, {
        status: 'available',
        category_id: categoryId.value,
      }),
    enabled: computed(() => Boolean(auth.token && eventId.value && categoryId.value)),
  })
}

export function useCreateSeatsMutation(eventId: Ref<string>) {
  const auth = useAuthStore()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: CreateSeatsRequest) => createSeats(eventId.value, body, auth.token!),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: seatKeys.list(eventId.value) }),
  })
}

export function useUpdateSeatMutation(eventId: Ref<string>) {
  const auth = useAuthStore()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ seatId, body }: { seatId: string; body: UpdateSeatRequest }) =>
      updateSeat(seatId, body, auth.token!),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: seatKeys.list(eventId.value) }),
  })
}
