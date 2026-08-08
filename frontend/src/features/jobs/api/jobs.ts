import { apiGet } from '@/shared/api/client'
import type { Job } from '@/shared/api/types'

export function listJobs(eventId: string, token: string) {
  return apiGet<Job[]>(`/events/${eventId}/jobs`, token)
}

export function getJob(jobId: string, token: string) {
  return apiGet<Job>(`/jobs/${jobId}`, token)
}
