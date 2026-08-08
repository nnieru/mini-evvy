import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'
import {
  createBooking,
  createBookingsBatch,
  deleteBooking,
  listBookings,
  type ListBookingsParams,
  resendInvitation,
  updateBooking,
} from '../api/bookings'
import { useAuthStore } from '@/features/auth/stores/auth'
import { guestKeys } from '@/features/guests/queries/guests'
import type { CreateBookingBatchRequest, CreateBookingRequest, UpdateBookingRequest } from '@/shared/api/types'

export const bookingKeys = {
  list: (eventId: string, params: ListBookingsParams) =>
    ['bookings', eventId, params] as const,
}

export function useBookingsQuery(eventId: Ref<string>, filters: Ref<ListBookingsParams>) {
  const auth = useAuthStore()
  return useQuery({
    queryKey: computed(() => bookingKeys.list(eventId.value, filters.value)),
    queryFn: () => listBookings(eventId.value, filters.value, auth.token!),
    enabled: computed(() => Boolean(auth.token && eventId.value)),
  })
}

export function useCreateBookingMutation(eventId: Ref<string>) {
  const auth = useAuthStore()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: CreateBookingRequest) =>
      createBooking(eventId.value, body, auth.token!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['bookings', eventId.value] })
      queryClient.invalidateQueries({ queryKey: guestKeys.list(eventId.value) })
      queryClient.invalidateQueries({ queryKey: ['seats', eventId.value] })
    },
  })
}

export function useCreateBookingsBatchMutation(eventId: Ref<string>) {
  const auth = useAuthStore()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: CreateBookingBatchRequest) =>
      createBookingsBatch(eventId.value, body, auth.token!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['bookings', eventId.value] })
      queryClient.invalidateQueries({ queryKey: guestKeys.list(eventId.value) })
      queryClient.invalidateQueries({ queryKey: ['seats', eventId.value] })
    },
  })
}

export function useUpdateBookingMutation(eventId: Ref<string>) {
  const auth = useAuthStore()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ bookingId, body }: { bookingId: string; body: UpdateBookingRequest }) =>
      updateBooking(bookingId, body, auth.token!),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ['bookings', eventId.value] }),
  })
}

export function useDeleteBookingMutation(eventId: Ref<string>) {
  const auth = useAuthStore()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (bookingId: string) => deleteBooking(bookingId, auth.token!),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ['bookings', eventId.value] }),
  })
}

export function useResendInvitationMutation() {
  const auth = useAuthStore()
  return useMutation({
    mutationFn: (bookingId: string) => resendInvitation(bookingId, auth.token!),
  })
}
