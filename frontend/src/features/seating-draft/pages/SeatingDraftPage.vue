<script setup lang="ts">
import { computed, h, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import type { ColumnDef } from '@tanstack/vue-table'
import PageHeader from '@/shared/ui/PageHeader.vue'
import Card from '@/shared/ui/Card.vue'
import Button from '@/shared/ui/Button.vue'
import Input from '@/shared/ui/Input.vue'
import Select from '@/shared/ui/Select.vue'
import Badge from '@/shared/ui/Badge.vue'
import Modal from '@/shared/ui/Modal.vue'
import LoadingSpinner from '@/shared/ui/LoadingSpinner.vue'
import Alert from '@/shared/ui/Alert.vue'
import DataTable from '@/shared/ui/DataTable.vue'
import { getErrorMessage } from '@/shared/lib/errors'
import { titleCase } from '@/shared/lib/format'
import { useDebouncedSearch } from '@/shared/composables/useDebouncedSearch'
import { useJobPoller } from '@/shared/composables/useJobPoller'
import { useEventQuery } from '@/features/events/queries/events'
import { useJobsQuery } from '@/features/jobs/queries/jobs'
import { useAllSeatsQuery } from '@/features/seats/queries/seats'
import { useAuthStore } from '@/features/auth/stores/auth'
import {
  useApproveSeatingMutation,
  useFinalizeSeatingMutation,
  useReassignDraftItemMutation,
  useRejectSeatingMutation,
  useSeatingDraftPreviewQuery,
} from '../queries/seatingDraft'
import { exportSeatingPreview, type ListSeatingPreviewParams, type SeatingPreviewRow } from '../api/seatingDraft'

const route = useRoute()
const auth = useAuthStore()
const eventId = computed(() => route.params.eventId as string)

const message = ref('')
const error = ref('')
const previewPage = ref(1)
const previewPageSize = ref(20)
const exportingPreview = ref(false)
const showReassign = ref(false)
const reassignItem = ref<SeatingPreviewRow | null>(null)
const reassignSeatId = ref('')

const { searchInput: previewSearchInput, searchQuery: previewSearchQuery } = useDebouncedSearch(() => {
  previewPage.value = 1
})

const { data: event, refetch: refetchEvent } = useEventQuery(eventId)
const jobListFilters = computed(() => ({ page: 1, page_size: 100 }))
const { data: jobsPage, refetch: refetchJobs } = useJobsQuery(eventId, jobListFilters)
const jobs = computed(() => jobsPage.value?.items ?? [])
const { data: allSeats, refetch: refetchSeats } = useAllSeatsQuery(eventId)

const finalizeMutation = useFinalizeSeatingMutation(eventId)
const approveMutation = useApproveSeatingMutation(eventId)
const rejectMutation = useRejectSeatingMutation(eventId)
const reassignMutation = useReassignDraftItemMutation(eventId)
const { polling, pollJob } = useJobPoller()

const seatingPhase = computed(() => event.value?.seating_phase ?? 'open')
const hasOpenDraft = computed(() => seatingPhase.value === 'preview')
const previewEnabled = computed(() => hasOpenDraft.value)

const previewFilters = computed<ListSeatingPreviewParams>(() => ({
  page: previewPage.value,
  page_size: previewPageSize.value,
  q: previewSearchQuery.value || undefined,
}))

const {
  data: previewPageData,
  isLoading: previewLoading,
  isFetching: previewFetching,
  isError: previewError,
  error: previewFetchError,
  refetch: refetchPreview,
} = useSeatingDraftPreviewQuery(eventId, previewEnabled, previewFilters)

const previewRows = computed(() => previewPageData.value?.items ?? [])
const previewTotal = computed(() => previewPageData.value?.total ?? 0)

const canGenerate = computed(
  () =>
    seatingPhase.value === 'open' ||
    seatingPhase.value === 'approved' ||
    seatingPhase.value === 'preview',
)

const reassignSeatOptions = computed(() => {
  if (!reassignItem.value) return []
  const currentSeat = (allSeats.value ?? []).find((s) => s.id === reassignItem.value?.seat_id)
  const categoryId = currentSeat?.category_id
  if (!categoryId) return []

  return (allSeats.value ?? [])
    .filter((seat) => {
      if (!seat.id || !seat.code) return false
      if (seat.category_id !== categoryId) return false
      return seat.status === 'available' || seat.id === reassignItem.value?.seat_id
    })
    .map((seat) => ({
      label: seat.code!,
      value: seat.id!,
    }))
})

const previewColumns = computed<ColumnDef<SeatingPreviewRow>[]>(() => [
  {
    accessorKey: 'guest_name',
    header: 'Guest',
    cell: ({ row }) => row.original.guest_name || '—',
  },
  {
    accessorKey: 'guest_email',
    header: 'Email',
    cell: ({ row }) => row.original.guest_email || '—',
  },
  {
    accessorKey: 'seat_code',
    header: 'Seat',
    cell: ({ row }) => row.original.seat_code || '—',
  },
  {
    id: 'category',
    header: 'Category',
    cell: ({ row }) => {
      const name = row.original.category_name || '—'
      const code = row.original.category_code
      return code ? `${name} (${code})` : name
    },
  },
  {
    id: 'section',
    header: 'Section',
    cell: ({ row }) => row.original.section || '—',
  },
  {
    id: 'actions',
    header: 'Actions',
    cell: ({ row }) =>
      h(
        Button,
        {
          size: 'sm',
          variant: 'secondary',
          onClick: () => openReassign(row.original),
        },
        () => 'Reassign',
      ),
  },
])

const approveActionsDisabled = computed(
  () => previewLoading.value || previewFetching.value || previewError.value,
)

const finalizeJob = computed(() =>
  jobs.value?.find(
    (job) =>
      job.type === 'finalize_seating' &&
      (job.status === 'pending' || job.status === 'in_process'),
  ),
)

const finalizeInProgress = computed(() => Boolean(finalizeJob.value) || polling.value)

const previewReady = computed(
  () => !previewLoading.value && !previewFetching.value && !previewError.value,
)

const previewEmpty = computed(() => previewReady.value && previewTotal.value === 0)

watch(
  finalizeJob,
  (job) => {
    if (!job?.id || polling.value) return
    void pollJob(job.id, {
      onDone: async () => {
        message.value = 'Seating draft generated — review assignments below'
        await refetchEvent()
        await refetchPreview()
        await refetchJobs()
        await refetchSeats()
      },
      onFailed: async () => {
        error.value = 'Generate seating failed — check that the worker is running'
        await refetchEvent()
        await refetchJobs()
      },
    })
  },
  { immediate: true },
)

watch(seatingPhase, (phase) => {
  if (phase === 'preview') {
    void refetchPreview()
  }
})

function openReassign(row: SeatingPreviewRow) {
  reassignItem.value = row
  reassignSeatId.value = row.seat_id
  showReassign.value = true
}

async function submitReassign() {
  if (!reassignItem.value?.id || !reassignSeatId.value) return
  error.value = ''
  try {
    await reassignMutation.mutateAsync({
      itemId: reassignItem.value.id,
      seatId: reassignSeatId.value,
    })
    showReassign.value = false
    message.value = 'Seat reassigned'
    await refetchPreview()
    await refetchSeats()
  } catch (err) {
    error.value = getErrorMessage(err)
  }
}

async function generateSeating() {
  error.value = ''
  message.value = ''
  try {
    const result = await finalizeMutation.mutateAsync()
    await refetchEvent()
    await pollJob(result.job_id, {
      onDone: async () => {
        message.value = 'Seating draft generated — review assignments below'
        await refetchEvent()
        await refetchPreview()
        await refetchSeats()
      },
      onFailed: async () => {
        error.value = 'Generate seating failed'
        await refetchEvent()
      },
    })
  } catch (err) {
    error.value = getErrorMessage(err)
  }
}

async function approveSeating() {
  error.value = ''
  message.value = ''
  try {
    await approveMutation.mutateAsync()
    message.value = 'Seating approved — invitation emails queued'
    await refetchEvent()
    await refetchPreview()
    await refetchSeats()
  } catch (err) {
    error.value = getErrorMessage(err)
  }
}

async function rejectSeating() {
  error.value = ''
  message.value = ''
  try {
    await rejectMutation.mutateAsync()
    message.value =
      seatingPhase.value === 'preview'
        ? 'Draft rejected — seats unlocked'
        : 'Draft rejected'
    await refetchEvent()
    await refetchSeats()
  } catch (err) {
    error.value = getErrorMessage(err)
  }
}

async function downloadSeatingPreview() {
  if (!auth.token) return
  error.value = ''
  exportingPreview.value = true
  try {
    await exportSeatingPreview(eventId.value, auth.token, previewSearchQuery.value || undefined)
  } catch (err) {
    error.value = getErrorMessage(err)
  } finally {
    exportingPreview.value = false
  }
}
</script>

<template>
  <div class="space-y-6">
    <PageHeader
      title="Seating draft"
      description="Generate seat assignments, review the draft, reassign seats, then approve to send invitations."
    />

    <Alert v-if="message" tone="success" :title="message" />
    <Alert v-if="error" tone="error" :title="error" />

    <Card title="Draft status">
      <p class="mb-4 text-sm text-ink-muted">
        Phase:
        <Badge class="ml-1">{{ titleCase(seatingPhase) }}</Badge>
      </p>

      <div v-if="canGenerate && !hasOpenDraft" class="space-y-3">
        <p class="text-sm text-ink-muted">
          Upload guests first, then generate seating. Seats are held in draft until you approve or
          reject. After approval you can import more guests and generate another wave.
        </p>
        <Alert v-if="finalizeInProgress" tone="info" title="Generating seating…">
          <p class="mt-1 text-ink-muted">
            Keep this page open or check Jobs. Draft table appears when the worker finishes.
          </p>
        </Alert>
        <Button
          :loading="finalizeMutation.isPending.value || finalizeInProgress"
          :disabled="finalizeInProgress"
          @click="generateSeating"
        >
          {{ seatingPhase === 'approved' ? 'Generate next wave' : 'Generate seating' }}
        </Button>
      </div>

      <div v-else-if="hasOpenDraft" class="space-y-3">
        <p class="text-sm text-ink-muted">
          Review assignments below. Approve to create bookings and send invitations, or Reject to
          unlock held seats.
        </p>
        <div class="flex flex-wrap gap-3">
          <Button
            :loading="finalizeMutation.isPending.value || finalizeInProgress"
            :disabled="finalizeInProgress"
            variant="secondary"
            @click="generateSeating"
          >
            Append guests
          </Button>
          <Button
            :loading="approveMutation.isPending.value"
            :disabled="approveActionsDisabled"
            @click="approveSeating"
          >
            Approve seating
          </Button>
          <Button
            variant="secondary"
            :loading="rejectMutation.isPending.value"
            @click="rejectSeating"
          >
            Reject draft
          </Button>
        </div>
      </div>

      <div v-else-if="seatingPhase === 'approved'" class="space-y-3">
        <Alert tone="success" title="Last wave approved — invitations queued or sent." />
        <p class="text-sm text-ink-muted">
          Import more guests, then generate the next wave. Existing bookings are kept.
        </p>
        <Button
          :loading="finalizeMutation.isPending.value || finalizeInProgress"
          :disabled="finalizeInProgress"
          @click="generateSeating"
        >
          Generate next wave
        </Button>
      </div>
    </Card>

    <Card v-if="hasOpenDraft" title="Draft assignments">
      <div class="mb-4 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <Input
          v-model="previewSearchInput"
          label="Search assignments"
          placeholder="Guest name, email, or seat code"
          autocomplete="off"
          class="sm:flex-1"
        />
        <Button
          variant="secondary"
          :loading="exportingPreview"
          @click="downloadSeatingPreview"
        >
          Export Excel
        </Button>
      </div>

      <LoadingSpinner v-if="previewLoading || previewFetching" />
      <Alert
        v-else-if="previewError"
        tone="error"
        :title="getErrorMessage(previewFetchError)"
        class="mb-4"
      >
        <Button class="mt-3" @click="refetchPreview">Retry</Button>
      </Alert>
      <Alert
        v-else-if="previewEmpty"
        tone="warning"
        title="No seat assignments in draft"
        class="mb-4"
      >
        <p class="mt-1 text-ink-muted">
          Guests may be missing or seats unavailable. Click Reject draft to unlock, add guests, and
          try again.
        </p>
      </Alert>
      <DataTable
        v-else
        :columns="previewColumns"
        :rows="previewRows"
        :loading="previewLoading || previewFetching"
        :page="previewPage"
        :page-size="previewPageSize"
        :total="previewTotal"
        empty-message="No assignments yet."
        @update:page="previewPage = $event"
        @update:page-size="previewPageSize = $event"
      />
    </Card>

    <Modal v-model:open="showReassign" title="Reassign seat">
      <form class="space-y-4" @submit.prevent="submitReassign">
        <p v-if="reassignItem" class="text-sm text-ink-muted">
          {{ reassignItem.guest_name }} — current seat {{ reassignItem.seat_code }}
        </p>
        <Select
          v-model="reassignSeatId"
          label="New seat"
          placeholder="Select seat"
          :options="reassignSeatOptions"
        />
        <div class="flex justify-end gap-2">
          <Button type="button" variant="secondary" @click="showReassign = false">Cancel</Button>
          <Button type="submit" :loading="reassignMutation.isPending.value" :disabled="!reassignSeatId">
            Save
          </Button>
        </div>
      </form>
    </Modal>
  </div>
</template>
