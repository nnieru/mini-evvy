<script setup lang="ts">
import { computed, h, ref } from 'vue'
import { useRoute } from 'vue-router'
import type { ColumnDef } from '@tanstack/vue-table'
import PageHeader from '@/shared/ui/PageHeader.vue'
import Badge from '@/shared/ui/Badge.vue'
import DataTable from '@/shared/ui/DataTable.vue'
import Input from '@/shared/ui/Input.vue'
import { formatDateTime, titleCase } from '@/shared/lib/format'
import { useDebouncedSearch } from '@/shared/composables/useDebouncedSearch'
import { useJobsQuery } from '../queries/jobs'
import type { ListJobsParams } from '../api/jobs'
import type { Job } from '@/shared/api/types'

const route = useRoute()
const eventId = computed(() => route.params.eventId as string)

const page = ref(1)
const pageSize = ref(20)
const { searchInput, searchQuery } = useDebouncedSearch(() => {
  page.value = 1
})

const listFilters = computed<ListJobsParams>(() => ({
  page: page.value,
  page_size: pageSize.value,
  q: searchQuery.value || undefined,
}))

const { data: jobsPage, isLoading } = useJobsQuery(eventId, listFilters)

const jobRows = computed(() => jobsPage.value?.items ?? [])
const jobTotal = computed(() => jobsPage.value?.total ?? 0)

const jobColumns = computed<ColumnDef<Job>[]>(() => [
  {
    accessorKey: 'type',
    header: 'Type',
    cell: ({ row }) => titleCase(row.original.type ?? ''),
  },
  {
    accessorKey: 'id',
    header: 'Job ID',
    cell: ({ row }) => row.original.id || '—',
  },
  {
    id: 'updated',
    header: 'Updated',
    cell: ({ row }) => formatDateTime(row.original.updated_at),
  },
  {
    id: 'status',
    header: 'Status',
    cell: ({ row }) => {
      const status = row.original.status ?? ''
      const tone =
        status === 'done' ? 'success' : status === 'failed' ? 'danger' : 'warning'
      return h(Badge, { tone }, () => titleCase(status))
    },
  },
])
</script>

<template>
  <div class="space-y-6">
    <PageHeader title="Jobs" description="Background tasks for finalize seating and invitations." />

    <Input
      v-model="searchInput"
      label="Search jobs"
      placeholder="Type or status"
      autocomplete="off"
    />

    <DataTable
      :columns="jobColumns"
      :rows="jobRows"
      :loading="isLoading"
      :page="page"
      :page-size="pageSize"
      :total="jobTotal"
      empty-message="No jobs yet. Jobs appear after finalize seating or resend invitation."
      @update:page="page = $event"
      @update:page-size="pageSize = $event"
    />
  </div>
</template>
