import { apiDelete, apiGet, apiPatch, apiPost } from '@/shared/api/client'
import { appendListPageParams, type ListPageParams, type PaginatedList } from '@/shared/api/pagination'
import type {
  Attendance,
  CreateAttendanceRequest,
  UpdateAttendanceRequest,
} from '@/shared/api/types'

export type ListAttendanceParams = ListPageParams

export function listAttendance(eventId: string, token: string, params?: ListAttendanceParams) {
  const search = new URLSearchParams()
  appendListPageParams(search, params)
  const query = search.toString()
  return apiGet<PaginatedList<Attendance>>(
    `/events/${eventId}/attendance${query ? `?${query}` : ''}`,
    token,
  )
}

export function getAttendance(attendanceId: string, token: string) {
  return apiGet<Attendance>(`/attendance/${attendanceId}`, token)
}

export function createAttendance(eventId: string, body: CreateAttendanceRequest, token: string) {
  return apiPost<Attendance>(`/events/${eventId}/attendance`, body, token)
}

export function updateAttendance(
  attendanceId: string,
  body: UpdateAttendanceRequest,
  token: string,
) {
  return apiPatch<Attendance>(`/attendance/${attendanceId}`, body, token)
}

export function deleteAttendance(attendanceId: string, token: string) {
  return apiDelete<void>(`/attendance/${attendanceId}`, token)
}
