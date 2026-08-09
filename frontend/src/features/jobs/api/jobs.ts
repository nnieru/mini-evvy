import { apiGet } from '@/shared/api/client'
import { appendListPageParams, type ListPageParams, type PaginatedList } from '@/shared/api/pagination'
import type { Job } from '@/shared/api/types'

export type ListJobsParams = ListPageParams

export function listJobs(eventId: string, token: string, params?: ListJobsParams) {
  const search = new URLSearchParams()
  appendListPageParams(search, params)
  const query = search.toString()
  return apiGet<PaginatedList<Job>>(
    `/events/${eventId}/jobs${query ? `?${query}` : ''}`,
    token,
  )
}

export function getJob(jobId: string, token: string) {
  return apiGet<Job>(`/jobs/${jobId}`, token)
}
