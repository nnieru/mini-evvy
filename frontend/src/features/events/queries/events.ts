import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'
import {
  approveSeating,
  createEvent,
  finalizeSeating,
  getEvent,
  getSeatingPreview,
  listEvents,
  listMyEvents,
  rejectSeating,
  updateEvent,
} from '../api/events'
import { useAuthStore } from '@/features/auth/stores/auth'
import type { CreateEventRequest, UpdateEventRequest } from '@/shared/api/types'

export const eventKeys = {
  list: (orgId: string) => ['events', orgId] as const,
  mine: ['events', 'mine'] as const,
  detail: (eventId: string) => ['events', 'detail', eventId] as const,
  seatingPreview: (eventId: string) => ['events', 'seating-preview', eventId] as const,
}

export function useEventsQuery(orgId: Ref<string>) {
  const auth = useAuthStore()
  return useQuery({
    queryKey: computed(() => eventKeys.list(orgId.value)),
    queryFn: () => listEvents(orgId.value, auth.token!),
    enabled: computed(() => Boolean(auth.token && orgId.value)),
  })
}

export function useMyEventsQuery(enabled: Ref<boolean>) {
  const auth = useAuthStore()
  return useQuery({
    queryKey: eventKeys.mine,
    queryFn: () => listMyEvents(auth.token!),
    enabled: computed(() => Boolean(auth.token && enabled.value)),
  })
}

export function useEventQuery(eventId: Ref<string>) {
  const auth = useAuthStore()
  return useQuery({
    queryKey: computed(() => eventKeys.detail(eventId.value)),
    queryFn: () => getEvent(eventId.value, auth.token!),
    enabled: computed(() => Boolean(auth.token && eventId.value)),
  })
}

export function useCreateEventMutation(orgId: Ref<string>) {
  const auth = useAuthStore()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: CreateEventRequest) => createEvent(orgId.value, body, auth.token!),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: eventKeys.list(orgId.value) }),
  })
}

export function useUpdateEventMutation(eventId: Ref<string>) {
  const auth = useAuthStore()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: UpdateEventRequest) => updateEvent(eventId.value, body, auth.token!),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: eventKeys.detail(eventId.value) }),
  })
}

export function useFinalizeSeatingMutation(eventId: Ref<string>) {
  const auth = useAuthStore()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => finalizeSeating(eventId.value, auth.token!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: eventKeys.detail(eventId.value) })
    },
  })
}

export function useSeatingPreviewQuery(eventId: Ref<string>, enabled: Ref<boolean>) {
  const auth = useAuthStore()
  return useQuery({
    queryKey: computed(() => eventKeys.seatingPreview(eventId.value)),
    queryFn: () => getSeatingPreview(eventId.value, auth.token!),
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
      queryClient.invalidateQueries({ queryKey: eventKeys.seatingPreview(eventId.value) })
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
      queryClient.invalidateQueries({ queryKey: eventKeys.seatingPreview(eventId.value) })
    },
  })
}
