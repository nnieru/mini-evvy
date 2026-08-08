import { apiDelete, apiGet, apiPatch, apiPost } from '@/shared/api/client'
import type {
  Attendance,
  CreateAttendanceRequest,
  UpdateAttendanceRequest,
} from '@/shared/api/types'

export function listAttendance(eventId: string, token: string) {
  return apiGet<Attendance[]>(`/events/${eventId}/attendance`, token)
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
