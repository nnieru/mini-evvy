import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'
import {
  createAttendance,
  deleteAttendance,
  listAttendance,
  updateAttendance,
} from '../api/attendance'
import { useAuthStore } from '@/features/auth/stores/auth'
import type { CreateAttendanceRequest } from '@/shared/api/types'

export const attendanceKeys = {
  list: (eventId: string) => ['attendance', eventId] as const,
}

export function useAttendanceQuery(eventId: Ref<string>) {
  const auth = useAuthStore()
  return useQuery({
    queryKey: computed(() => attendanceKeys.list(eventId.value)),
    queryFn: () => listAttendance(eventId.value, auth.token!),
    enabled: computed(() => Boolean(auth.token && eventId.value)),
  })
}

export function useCheckInMutation(eventId: Ref<string>) {
  const auth = useAuthStore()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: CreateAttendanceRequest) =>
      createAttendance(eventId.value, body, auth.token!),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: attendanceKeys.list(eventId.value) }),
  })
}

export function useUndoCheckInMutation(eventId: Ref<string>) {
  const auth = useAuthStore()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (attendanceId: string) =>
      updateAttendance(attendanceId, { status: 'not_checked_in' }, auth.token!),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: attendanceKeys.list(eventId.value) }),
  })
}

export function useDeleteAttendanceMutation(eventId: Ref<string>) {
  const auth = useAuthStore()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (attendanceId: string) => deleteAttendance(attendanceId, auth.token!),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: attendanceKeys.list(eventId.value) }),
  })
}
