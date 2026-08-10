import { apiGet } from '@/shared/api/client'
import { ApiError } from '@/shared/lib/errors'
import { appendListPageParams, type ListPageParams, type PaginatedList } from '@/shared/api/pagination'
import type { Job } from '@/shared/api/types'

const baseUrl = import.meta.env.VITE_API_BASE_URL ?? '/api'

export type ListJobsParams = ListPageParams & {
  type?: string
  status?: string
}

export type ExportInvitationEmailsParams = {
  q?: string
  status?: string
}

export function listJobs(eventId: string, token: string, params?: ListJobsParams) {
  const search = new URLSearchParams()
  appendListPageParams(search, params)
  if (params?.type) search.set('type', params.type)
  if (params?.status) search.set('status', params.status)
  const query = search.toString()
  return apiGet<PaginatedList<Job>>(
    `/events/${eventId}/jobs${query ? `?${query}` : ''}`,
    token,
  )
}

export function getJob(jobId: string, token: string) {
  return apiGet<Job>(`/jobs/${jobId}`, token)
}

export async function exportInvitationEmails(
  eventId: string,
  token: string,
  params?: ExportInvitationEmailsParams,
) {
  const search = new URLSearchParams()
  if (params?.q) search.set('q', params.q)
  if (params?.status) search.set('status', params.status)
  const query = search.toString()

  const response = await fetch(
    `${baseUrl}/events/${eventId}/jobs/invitation-emails/export${query ? `?${query}` : ''}`,
    {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    },
  )

  if (!response.ok) {
    let message = response.statusText
    try {
      const payload = (await response.json()) as { error?: { message?: string } }
      message = payload.error?.message ?? message
    } catch {
      // ignore non-json error bodies
    }
    throw new ApiError(response.status, 'EXPORT_FAILED', message)
  }

  const blob = await response.blob()
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `invitation-emails-${eventId}.xlsx`
  link.click()
  URL.revokeObjectURL(url)
}
