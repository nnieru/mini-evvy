import { onUnmounted, ref } from 'vue'
import { useQueryClient } from '@tanstack/vue-query'
import { getJob } from '@/features/jobs/api/jobs'
import { useAuthStore } from '@/features/auth/stores/auth'
import type { Job } from '@/shared/api/types'

export function useJobPoller() {
  const auth = useAuthStore()
  const queryClient = useQueryClient()
  const polling = ref(false)
  const currentJob = ref<Job | null>(null)
  const error = ref<string | null>(null)

  let timer: ReturnType<typeof setInterval> | null = null

  function stop() {
    if (timer) {
      clearInterval(timer)
      timer = null
    }
    polling.value = false
  }

  async function pollJob(
    jobId: string,
    options?: { onDone?: (job: Job) => void; onFailed?: (job: Job) => void },
  ) {
    stop()
    polling.value = true
    error.value = null
    currentJob.value = null

    const check = async () => {
      try {
        const job = await getJob(jobId, auth.token!)
        currentJob.value = job
        if (job.status === 'done') {
          stop()
          await queryClient.invalidateQueries()
          options?.onDone?.(job)
        } else if (job.status === 'failed') {
          stop()
          options?.onFailed?.(job)
        }
      } catch (err) {
        stop()
        error.value = err instanceof Error ? err.message : 'Failed to poll job'
      }
    }

    await check()
    if (polling.value) {
      timer = setInterval(check, 2000)
    }
  }

  onUnmounted(stop)

  return { polling, currentJob, error, pollJob, stop }
}
