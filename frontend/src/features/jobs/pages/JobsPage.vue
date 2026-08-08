<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import PageHeader from '@/shared/ui/PageHeader.vue'
import Card from '@/shared/ui/Card.vue'
import Badge from '@/shared/ui/Badge.vue'
import LoadingSpinner from '@/shared/ui/LoadingSpinner.vue'
import EmptyState from '@/shared/ui/EmptyState.vue'
import { formatDateTime, titleCase } from '@/shared/lib/format'
import { useJobsQuery } from '../queries/jobs'

const route = useRoute()
const eventId = computed(() => route.params.eventId as string)

const { data: jobs, isLoading } = useJobsQuery(eventId)
</script>

<template>
  <div class="space-y-6">
    <PageHeader title="Jobs" description="Background tasks for finalize seating and invitations." />

    <LoadingSpinner v-if="isLoading" />
    <EmptyState v-else-if="!jobs?.length" title="No jobs yet" description="Jobs appear after finalize seating or resend invitation." />

    <div v-else class="space-y-3">
      <Card v-for="job in jobs" :key="job.id">
        <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <p class="font-medium text-ink">{{ titleCase(job.type) }}</p>
            <p class="text-sm text-ink-muted">{{ job.id }}</p>
            <p class="text-sm text-ink-muted">Updated {{ formatDateTime(job.updated_at) }}</p>
          </div>
          <Badge
            :tone="job.status === 'done' ? 'success' : job.status === 'failed' ? 'danger' : 'warning'"
          >
            {{ titleCase(job.status) }}
          </Badge>
        </div>
      </Card>
    </div>
  </div>
</template>
