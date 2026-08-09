import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'
import {
  createSeats,
  listSeats,
  listSeatsAll,
  updateSeat,
  type ListSeatsParams,
} from '../api/seats'
import { useAuthStore } from '@/features/auth/stores/auth'
import type { CreateSeatsRequest, UpdateSeatRequest } from '@/shared/api/types'

export const seatKeys = {
  list: (eventId: string, filters?: ListSeatsParams) =>
    ['seats', eventId, filters ?? null] as const,
  all: (eventId: string, filters?: Pick<ListSeatsParams, 'status' | 'category_id'>) =>
    ['seats', eventId, 'all', filters ?? null] as const,
}

export function useSeatsQuery(eventId: Ref<string>, filters?: Ref<ListSeatsParams | undefined>) {
  const auth = useAuthStore()
  return useQuery({
    queryKey: computed(() => seatKeys.list(eventId.value, filters?.value)),
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

export function useAllSeatsQuery(
  eventId: Ref<string>,
  filters?: Ref<Pick<ListSeatsParams, 'status' | 'category_id'> | undefined>,
) {
  const auth = useAuthStore()
  return useQuery({
    queryKey: computed(() => seatKeys.all(eventId.value, filters?.value)),
    queryFn: () => listSeatsAll(eventId.value, auth.token!, filters?.value),
    enabled: computed(() => Boolean(auth.token && eventId.value)),
  })
}

export function useAvailableSeatsQuery(eventId: Ref<string>, categoryId: Ref<string>) {
  const auth = useAuthStore()
  const filters = computed(() => ({
    status: 'available' as const,
    category_id: categoryId.value,
    page_size: 100,
  }))
  return useQuery({
    queryKey: computed(() => seatKeys.list(eventId.value, filters.value)),
    queryFn: () => listSeats(eventId.value, auth.token!, filters.value),
    enabled: computed(() => Boolean(auth.token && eventId.value && categoryId.value)),
    select: (data) => data.items,
  })
}

export function useCreateSeatsMutation(eventId: Ref<string>) {
  const auth = useAuthStore()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: CreateSeatsRequest) => createSeats(eventId.value, body, auth.token!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['seats', eventId.value] })
    },
  })
}

export function useUpdateSeatMutation(eventId: Ref<string>) {
  const auth = useAuthStore()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ seatId, body }: { seatId: string; body: UpdateSeatRequest }) =>
      updateSeat(seatId, body, auth.token!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['seats', eventId.value] })
    },
  })
}
