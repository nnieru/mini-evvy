import { useQuery } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'
import { getJob, listJobs, type ListJobsParams } from '../api/jobs'
import { useAuthStore } from '@/features/auth/stores/auth'

export const jobKeys = {
  list: (eventId: string, params?: ListJobsParams) => ['jobs', eventId, params ?? null] as const,
  detail: (jobId: string) => ['jobs', 'detail', jobId] as const,
}

export function useJobsQuery(eventId: Ref<string>, filters?: Ref<ListJobsParams>) {
  const auth = useAuthStore()
  return useQuery({
    queryKey: computed(() => jobKeys.list(eventId.value, filters?.value)),
    queryFn: () => listJobs(eventId.value, auth.token!, filters?.value),
    enabled: computed(() => Boolean(auth.token && eventId.value)),
  })
}

export function useJobQuery(jobId: Ref<string>) {
  const auth = useAuthStore()
  return useQuery({
    queryKey: computed(() => jobKeys.detail(jobId.value)),
    queryFn: () => getJob(jobId.value, auth.token!),
    enabled: computed(() => Boolean(auth.token && jobId.value)),
  })
}
