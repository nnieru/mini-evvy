<script setup lang="ts">
import { computed, h, ref } from 'vue'
import { useRoute } from 'vue-router'
import type { ColumnDef } from '@tanstack/vue-table'
import PageHeader from '@/shared/ui/PageHeader.vue'
import Badge from '@/shared/ui/Badge.vue'
import Button from '@/shared/ui/Button.vue'
import DataTable from '@/shared/ui/DataTable.vue'
import Input from '@/shared/ui/Input.vue'
import Select from '@/shared/ui/Select.vue'
import { formatDateTime, titleCase } from '@/shared/lib/format'
import { getErrorMessage } from '@/shared/lib/errors'
import { useToast } from '@/shared/composables/useToast'
import { useDebouncedSearch } from '@/shared/composables/useDebouncedSearch'
import { useAuthStore } from '@/features/auth/stores/auth'
import { useJobsQuery } from '../queries/jobs'
import { exportInvitationEmails, type ListJobsParams } from '../api/jobs'
import type { Job } from '@/shared/api/types'

const route = useRoute()
const auth = useAuthStore()
const eventId = computed(() => route.params.eventId as string)

const page = ref(1)
const pageSize = ref(20)
const typeFilter = ref('')
const statusFilter = ref('')
const exporting = ref(false)
const { showToast } = useToast()

const { searchInput, searchQuery } = useDebouncedSearch(() => {
  page.value = 1
})

const listFilters = computed<ListJobsParams>(() => ({
  page: page.value,
  page_size: pageSize.value,
  q: searchQuery.value || undefined,
  type: typeFilter.value || undefined,
  status: statusFilter.value || undefined,
}))

const { data: jobsPage, isLoading } = useJobsQuery(eventId, listFilters)

const jobRows = computed(() => jobsPage.value?.items ?? [])
const jobTotal = computed(() => jobsPage.value?.total ?? 0)

const typeOptions = [
  { value: '', label: 'All types' },
  { value: 'finalize_seating', label: 'Finalize seating' },
  { value: 'send_invitation', label: 'Send invitation' },
]

const statusOptions = [
  { value: '', label: 'All statuses' },
  { value: 'pending', label: 'Pending' },
  { value: 'in_process', label: 'In process' },
  { value: 'done', label: 'Done' },
  { value: 'failed', label: 'Failed' },
]

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

async function downloadInvitationReport() {
  exporting.value = true
  try {
    await exportInvitationEmails(eventId.value, auth.token!, {
      q: searchQuery.value || undefined,
      status: statusFilter.value || undefined,
    })
    showToast('Invitation report downloaded', 'success')
  } catch (err) {
    showToast(getErrorMessage(err), 'error')
  } finally {
    exporting.value = false
  }
}
</script>

<template>
  <div class="space-y-6">
    <PageHeader title="Jobs" description="Background tasks for finalize seating and invitations." />

    <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <Input
        v-model="searchInput"
        label="Search jobs"
        placeholder="Type or status"
        autocomplete="off"
      />
      <Select
        v-model="typeFilter"
        label="Job type"
        :options="typeOptions"
        @update:model-value="page = 1"
      />
      <Select
        v-model="statusFilter"
        label="Job status"
        :options="statusOptions"
        @update:model-value="page = 1"
      />
    </div>

    <div class="flex flex-wrap items-center gap-3">
      <Button :disabled="exporting" @click="downloadInvitationReport">
        {{ exporting ? 'Exporting…' : 'Export invitation emails' }}
      </Button>
    </div>

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
