<script setup lang="ts">
import { computed, h, ref, watch } from 'vue'
import { useQueryClient } from '@tanstack/vue-query'
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
import { ApiError, getErrorMessage } from '@/shared/lib/errors'
import { titleCase } from '@/shared/lib/format'
import { useDebouncedSearch } from '@/shared/composables/useDebouncedSearch'
import { useJobPoller } from '@/shared/composables/useJobPoller'
import { useEventQuery } from '@/features/events/queries/events'
import { useJobsQuery } from '@/features/jobs/queries/jobs'
import { getSeat } from '@/features/seats/api/seats'
import { useAvailableSeatsQuery, useAllSeatsQuery } from '@/features/seats/queries/seats'
import { useGuestsQuery, useUnbookedGuestCountQuery, guestKeys } from '@/features/guests/queries/guests'
import { useCategoriesQuery } from '@/features/categories/queries/categories'
import { useAuthStore } from '@/features/auth/stores/auth'
import {
  useApproveSeatingMutation,
  useFinalizeSeatingMutation,
  useReassignDraftItemMutation,
  useRejectSeatingMutation,
  useSeatingDraftPreviewQuery,
  useSeatingReadinessQuery,
  seatingDraftKeys,
} from '../queries/seatingDraft'
import { exportSeatingPreview, type ListSeatingPreviewParams, type SeatingPreviewRow } from '../api/seatingDraft'

function isNoOpenDraftError(err: unknown): boolean {
  if (err instanceof ApiError) return err.code === 'NO_OPEN_DRAFT'
  if (err && typeof err === 'object' && 'code' in err) {
    return (err as { code?: string }).code === 'NO_OPEN_DRAFT'
  }
  return false
}

const route = useRoute()
const queryClient = useQueryClient()
const auth = useAuthStore()
const eventId = computed(() => route.params.eventId as string)

const message = ref('')
const error = ref('')

const noSeatsAssignedMessage =
  'No seats were assigned. Guests may already be seated, or there are no available seats in their category.'

function invalidateGuestCounts() {
  void queryClient.invalidateQueries({ queryKey: guestKeys.unbookedCount(eventId.value) })
  void queryClient.invalidateQueries({ queryKey: ['guests', eventId.value] })
  void queryClient.invalidateQueries({ queryKey: seatingDraftKeys.readiness(eventId.value) })
}
const previewPage = ref(1)
const previewPageSize = ref(20)
const exportingPreview = ref(false)
const showReassign = ref(false)
const reassignItem = ref<SeatingPreviewRow | null>(null)
const reassignSeatId = ref('')
const reassignCategoryId = ref('')

const { searchInput: previewSearchInput, searchQuery: previewSearchQuery } = useDebouncedSearch(() => {
  previewPage.value = 1
})

const { data: event, isLoading: eventLoading, refetch: refetchEvent } = useEventQuery(eventId)
const jobListFilters = computed(() => ({ page: 1, page_size: 100 }))
const { data: jobsPage, refetch: refetchJobs } = useJobsQuery(eventId, jobListFilters)
const jobs = computed(() => jobsPage.value?.items ?? [])
const { data: availableSeats } = useAvailableSeatsQuery(eventId, reassignCategoryId)

const guestSummaryFilters = computed(() => ({ page: 1, page_size: 1 }))
const { data: guestsSummary } = useGuestsQuery(eventId, guestSummaryFilters)
const guestTotal = computed(() => guestsSummary.value?.total ?? 0)
const { data: unbookedGuestCount } = useUnbookedGuestCountQuery(eventId)
const unbookedTotal = computed(() => unbookedGuestCount.value?.total ?? 0)
const { data: categories } = useCategoriesQuery(eventId)

const availableSeatFilters = computed(() => ({ status: 'available' as const }))
const { data: availableSeatsAll } = useAllSeatsQuery(eventId, availableSeatFilters)
const availableSeatTotal = computed(() => availableSeatsAll.value?.length ?? 0)

const finalizeMutation = useFinalizeSeatingMutation(eventId)
const approveMutation = useApproveSeatingMutation(eventId)
const rejectMutation = useRejectSeatingMutation(eventId)
const reassignMutation = useReassignDraftItemMutation(eventId)
const { polling, pollJob } = useJobPoller()

const seatingPhase = computed(() => event.value?.seating_phase ?? 'open')

const finalizeJob = computed(() =>
  jobs.value?.find(
    (job) =>
      job.type === 'finalize_seating' &&
      (job.status === 'pending' || job.status === 'in_process'),
  ),
)

const finalizeInProgress = computed(() => Boolean(finalizeJob.value) || polling.value)

// After reject/approve, hide draft UI until generate completes.
const forceHideDraft = ref(false)
const previewEnabled = computed(
  () => seatingPhase.value === 'preview' && !forceHideDraft.value && !finalizeInProgress.value,
)

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
  isSuccess: previewSuccess,
} = useSeatingDraftPreviewQuery(eventId, previewEnabled, previewFilters)

// Phase/draft desync: backend heals phase on NO_OPEN_DRAFT — refresh event, hide stale draft UI.
watch(previewFetchError, (err) => {
  if (!isNoOpenDraftError(err) || finalizeInProgress.value) return
  forceHideDraft.value = true
  if (message.value.startsWith('Seating draft generated')) {
    message.value = ''
    error.value =
      seatingPhase.value === 'approved'
        ? noSeatsAssignedMessage
        : 'No open draft found (phase was out of sync). Click Generate seating again.'
  }
  void refetchEvent()
})

watch(seatingPhase, (phase, prev) => {
  if (phase === 'preview' && prev && prev !== 'preview') {
    forceHideDraft.value = false
  }
})

const previewRows = computed(() => previewPageData.value?.items ?? [])
const previewTotal = computed(() => previewPageData.value?.total ?? 0)

const previewReady = computed(
  () =>
    previewEnabled.value &&
    !previewLoading.value &&
    !previewFetching.value &&
    previewSuccess.value &&
    !previewError.value,
)

// Only treat as open draft when assignments exist — empty preview stays on generate UI.
const hasOpenDraft = computed(
  () =>
    seatingPhase.value === 'preview' &&
    !forceHideDraft.value &&
    previewReady.value &&
    previewTotal.value > 0,
)

const emptyDraftPhase = computed(
  () =>
    seatingPhase.value === 'preview' &&
    !forceHideDraft.value &&
    !finalizeInProgress.value &&
    previewReady.value &&
    previewTotal.value === 0,
)

const draftProbeLoading = computed(
  () =>
    seatingPhase.value === 'preview' &&
    !forceHideDraft.value &&
    !finalizeInProgress.value &&
    (previewLoading.value || previewFetching.value),
)

const showGenerate = computed(() => !hasOpenDraft.value)

const readinessEnabled = computed(() => showGenerate.value && !eventLoading.value)
const { data: seatingReadiness } = useSeatingReadinessQuery(eventId, readinessEnabled)

const capacityBlocked = computed(
  () =>
    (seatingReadiness.value?.slots_needed ?? 0) > 0 &&
    seatingReadiness.value?.can_assign_any === false,
)

const categoryNameById = computed(() => {
  const map = new Map<string, string>()
  for (const category of categories.value ?? []) {
    if (category.id) map.set(category.id, category.name ?? category.id)
  }
  return map
})

const capacityShortfallLines = computed(() =>
  (seatingReadiness.value?.shortfalls ?? []).map((item) => {
    const name = categoryNameById.value.get(item.category_id) ?? item.category_id
    return `${name}: need ${item.slots_needed}, ${item.seats_available} available`
  }),
)

const generateDisabled = computed(
  () =>
    (unbookedTotal.value === 0 && !finalizeInProgress.value && !emptyDraftPhase.value) ||
    (capacityBlocked.value && !finalizeInProgress.value && !emptyDraftPhase.value),
)

const generateGuidance = computed(() => {
  if (seatingPhase.value === 'approved') {
    return 'Prior wave approved. Upload more guests, then generate next wave. Only guests still needing seats are assigned.'
  }
  return 'Upload guests first, then generate seating. Assigns uploaded guests to available seats as a draft (held). Does not create bookings until Approve. Seats already booked manually stay taken on the map.'
})

const reassignSeatOptions = computed(() => {
  if (!reassignItem.value) return []

  return (availableSeats.value ?? [])
    .filter((seat) => {
      if (!seat.id || !seat.code) return false
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
  () => previewLoading.value || previewFetching.value || previewError.value || previewTotal.value === 0,
)

const finishedPollJobIds = ref(new Set<string>())

function markPollFinished(jobId: string) {
  finishedPollJobIds.value = new Set([...finishedPollJobIds.value, jobId])
}

async function afterGenerateDone() {
  forceHideDraft.value = false
  const eventResult = await refetchEvent()
  await refetchJobs()
  const phase = eventResult.data?.seating_phase ?? seatingPhase.value
  const preview = await refetchPreview()

  const noAssignments =
    isNoOpenDraftError(preview.error) ||
    (!preview.isError && (preview.data?.total ?? 0) === 0)

  if (phase === 'approved' && noAssignments) {
    message.value = ''
    error.value = noSeatsAssignedMessage
    forceHideDraft.value = true
    invalidateGuestCounts()
    return
  }

  if (isNoOpenDraftError(preview.error)) {
    message.value = ''
    error.value =
      'Generate finished but no open draft found. Keep worker running, then click Generate seating again.'
    forceHideDraft.value = true
    await refetchEvent()
    return
  }
  if (preview.isError) {
    message.value = ''
    error.value = getErrorMessage(preview.error)
    return
  }
  if ((preview.data?.total ?? 0) === 0) {
    message.value = ''
    error.value =
      'No seats could be assigned. Check that guests use the right category, seats are available (not manually booked), and the worker is running. Reject the draft to reset phase, then try again.'
    return
  }
  message.value = 'Seating draft generated — review assignments below'
}

watch(
  () => finalizeJob.value?.id,
  (jobId) => {
    if (!jobId || polling.value) return
    if (finishedPollJobIds.value.has(jobId)) return
    void pollJob(jobId, {
      onDone: async (job) => {
        if (job.id) markPollFinished(job.id)
        await afterGenerateDone()
      },
      onFailed: async () => {
        error.value =
          'Generate seating failed. Common causes: no available seats left (manual bookings or wrong category), or worker not running.'
        await refetchEvent()
        await refetchJobs()
        invalidateGuestCounts()
      },
    })
  },
  { immediate: true },
)

async function openReassign(row: SeatingPreviewRow) {
  reassignItem.value = row
  reassignSeatId.value = row.seat_id
  reassignCategoryId.value = ''
  showReassign.value = true
  try {
    const seat = await getSeat(row.seat_id, auth.token!)
    reassignCategoryId.value = seat.category_id ?? ''
  } catch (err) {
    error.value = getErrorMessage(err)
    showReassign.value = false
  }
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
  } catch (err) {
    error.value = getErrorMessage(err)
  }
}

async function generateSeating() {
  error.value = ''
  message.value = ''
  try {
    const result = await finalizeMutation.mutateAsync()
    forceHideDraft.value = false
    await pollJob(result.job_id, {
      onDone: async (job) => {
        if (job.id) markPollFinished(job.id)
        await afterGenerateDone()
      },
      onFailed: async () => {
        error.value =
          'Generate seating failed. Common causes: no available seats left (manual bookings or wrong category), or worker not running.'
        await refetchEvent()
        await refetchJobs()
        invalidateGuestCounts()
      },
    })
    await refetchEvent()
    await refetchJobs()
  } catch (err) {
    error.value = getErrorMessage(err)
  }
}

async function approveSeating() {
  error.value = ''
  message.value = ''
  try {
    await approveMutation.mutateAsync()
    forceHideDraft.value = true
    message.value = 'Seating approved — invitation emails queued'
    await refetchEvent()
  } catch (err) {
    error.value = getErrorMessage(err)
  }
}

async function rejectSeating() {
  const confirmed = window.confirm(
    'Reject this draft? Held seats will unlock and unbooked guests will be removed. Manual bookings on the Bookings page are not affected.',
  )
  if (!confirmed) return

  error.value = ''
  message.value = ''
  try {
    await rejectMutation.mutateAsync()
    forceHideDraft.value = true
    message.value =
      'Draft rejected — held seats unlocked. Unbooked guests removed. Manual bookings are unchanged.'
    await refetchEvent()
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

const generateLabel = computed(() =>
  seatingPhase.value === 'approved' ? 'Generate next wave' : 'Generate seating',
)
</script>

<template>
  <div class="space-y-6">
    <PageHeader
      title="Seating draft"
      description="Bulk seating from uploaded guests: generate a draft, review assignments, then approve to create bookings and send invitations. Manual bookings use the Bookings page."
    />

    <Alert v-if="message" tone="success" :title="message" />
    <Alert v-if="error" tone="error" :title="error" />

    <LoadingSpinner v-if="eventLoading" />

    <Card v-else title="Draft status">
      <p class="mb-4 text-sm text-ink-muted">
        Phase:
        <Badge class="ml-1">{{ titleCase(seatingPhase) }}</Badge>
      </p>

      <div v-if="showGenerate" class="space-y-3">
        <LoadingSpinner v-if="draftProbeLoading" />
        <template v-else>
          <p class="text-sm text-ink-muted">
            {{ generateGuidance }}
          </p>
          <p class="text-sm text-ink-muted">
            {{ unbookedTotal }} need seating · {{ guestTotal }} on file ·
            {{ availableSeatTotal }} seat(s) available
          </p>
          <p v-if="generateDisabled && unbookedTotal === 0" class="text-sm text-ink-muted">
            No guests need seats.
          </p>
          <Alert
            v-if="capacityBlocked"
            tone="warning"
            title="No available seats in required categories"
            class="mb-2"
          >
            <p class="mt-1 text-ink-muted">
              Wave 1 (or manual bookings) may have filled these categories. Free seats on the
              Bookings page, add seats, or move guests to another category before generating.
            </p>
            <ul v-if="capacityShortfallLines.length" class="mt-2 list-disc space-y-1 pl-5 text-ink-muted">
              <li v-for="line in capacityShortfallLines" :key="line">{{ line }}</li>
            </ul>
          </Alert>
          <Alert
            v-if="emptyDraftPhase"
            tone="warning"
            title="Last generate assigned no seats"
            class="mb-2"
          >
            <p class="mt-1 text-ink-muted">
              Guests may use a category with no free seats, or seats may already be manually booked.
              Reject the draft to reset, then fix categories or free seats on the Bookings page.
            </p>
            <div class="mt-3 flex flex-wrap gap-2">
              <Button variant="secondary" :loading="rejectMutation.isPending.value" @click="rejectSeating">
                Reject draft
              </Button>
              <Button
                :loading="finalizeMutation.isPending.value || finalizeInProgress"
                :disabled="finalizeInProgress"
                @click="generateSeating"
              >
                Try generate again
              </Button>
            </div>
          </Alert>
          <Alert v-if="finalizeInProgress" tone="info" title="Generating seating…">
            <p class="mt-1 text-ink-muted">
              Keep this page open or check Jobs. Draft table appears when the worker finishes.
            </p>
          </Alert>
          <Button
            v-if="!emptyDraftPhase"
            :loading="finalizeMutation.isPending.value || finalizeInProgress"
            :disabled="finalizeInProgress || generateDisabled"
            @click="generateSeating"
          >
            {{ generateLabel }}
          </Button>
        </template>
      </div>

      <div v-else class="space-y-3">
        <p class="text-sm text-ink-muted">
          Review draft assignments below. Approve creates bookings and sends invitations. Reject
          cancels this draft only — held seats unlock; manual bookings are not affected.
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
        v-else-if="previewError && !isNoOpenDraftError(previewFetchError)"
        tone="error"
        :title="getErrorMessage(previewFetchError)"
        class="mb-4"
      >
        <Button class="mt-3" @click="refetchPreview">Retry</Button>
      </Alert>
      <DataTable
        v-else-if="previewReady"
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
