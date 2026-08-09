<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import PageHeader from '@/shared/ui/PageHeader.vue'
import Card from '@/shared/ui/Card.vue'
import Button from '@/shared/ui/Button.vue'
import Select from '@/shared/ui/Select.vue'
import Badge from '@/shared/ui/Badge.vue'
import LoadingSpinner from '@/shared/ui/LoadingSpinner.vue'
import Alert from '@/shared/ui/Alert.vue'
import { getErrorMessage } from '@/shared/lib/errors'
import { formatDate, titleCase } from '@/shared/lib/format'
import { useJobPoller } from '@/shared/composables/useJobPoller'
import { useEventOrgRole } from '@/features/organizations/composables/useEventOrgRole'
import {
  useApproveSeatingMutation,
  useEventQuery,
  useFinalizeSeatingMutation,
  useImportEventConfigMutation,
  useMyEventsQuery,
  useRejectSeatingMutation,
  useSeatingPreviewQuery,
  useUpdateEventMutation,
} from '../queries/events'
import { useJobsQuery } from '@/features/jobs/queries/jobs'

const route = useRoute()
const eventId = computed(() => route.params.eventId as string)

const status = ref('')
const message = ref('')
const error = ref('')

const { data: event, isLoading, refetch: refetchEvent } = useEventQuery(eventId)
const { isStaff } = useEventOrgRole(eventId)
const { data: jobs, refetch: refetchJobs } = useJobsQuery(eventId)
const updateMutation = useUpdateEventMutation(eventId)
const finalizeMutation = useFinalizeSeatingMutation(eventId)
const approveMutation = useApproveSeatingMutation(eventId)
const rejectMutation = useRejectSeatingMutation(eventId)
const importMutation = useImportEventConfigMutation(eventId)
const { data: myEvents } = useMyEventsQuery(isStaff)
const { polling, currentJob, pollJob } = useJobPoller()

const sourceEventId = ref('')
const includeCategories = ref(true)
const includeSeats = ref(true)
const includeEmailTemplate = ref(true)

const sourceEventOptions = computed(() =>
  (myEvents.value ?? [])
    .filter((item) => item.id !== eventId.value)
    .map((item) => ({
      label: `${item.name} (${item.organization_name})`,
      value: item.id!,
    })),
)

const seatingPhase = computed(() => event.value?.seating_phase ?? 'open')
const showPreview = computed(
  () => seatingPhase.value === 'preview' || seatingPhase.value === 'approved',
)
const previewEnabled = computed(() => showPreview.value)

const {
  data: previewRows,
  isLoading: previewLoading,
  isFetching: previewFetching,
  isError: previewError,
  error: previewFetchError,
  refetch: refetchPreview,
} = useSeatingPreviewQuery(eventId, previewEnabled)

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

const previewEmpty = computed(() => previewReady.value && !(previewRows.value?.length ?? 0))

watch(
  finalizeJob,
  (job) => {
    if (!job?.id || polling.value) return
    void pollJob(job.id, {
      onDone: async () => {
        message.value = 'Finalize seating completed — review assignments below'
        await refetchEvent()
        await refetchPreview()
        await refetchJobs()
      },
      onFailed: async () => {
        error.value = 'Finalize seating failed — check that the worker is running'
        await refetchEvent()
        await refetchJobs()
      },
    })
  },
  { immediate: true },
)

watch(seatingPhase, (phase) => {
  if (phase === 'preview' || phase === 'approved') {
    void refetchPreview()
  }
})

async function saveStatus() {
  if (!status.value) return
  error.value = ''
  try {
    await updateMutation.mutateAsync({ status: status.value as 'draft' | 'active' | 'cancelled' | 'completed' })
    message.value = 'Event status updated'
  } catch (err) {
    error.value = getErrorMessage(err)
  }
}

async function finalizeSeating() {
  error.value = ''
  message.value = ''
  try {
    const result = await finalizeMutation.mutateAsync()
    await refetchEvent()
    await pollJob(result.job_id, {
      onDone: async () => {
        message.value = 'Finalize seating completed — review assignments below'
        await refetchEvent()
        await refetchPreview()
      },
      onFailed: async () => {
        error.value = 'Finalize seating failed'
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
  } catch (err) {
    error.value = getErrorMessage(err)
  }
}

async function rejectSeating() {
  error.value = ''
  message.value = ''
  try {
    await rejectMutation.mutateAsync()
    message.value = 'Seating rejected — you can edit guests and re-finalize'
    await refetchEvent()
  } catch (err) {
    error.value = getErrorMessage(err)
  }
}

async function importConfig() {
  if (!sourceEventId.value) return
  error.value = ''
  message.value = ''
  try {
    const result = await importMutation.mutateAsync({
      source_event_id: sourceEventId.value,
      include_categories: includeCategories.value,
      include_seats: includeSeats.value,
      include_email_template: includeEmailTemplate.value,
    })
    const parts: string[] = []
    if (result.categories_created) {
      parts.push(`${result.categories_created} categories`)
    }
    if (result.seats_created) {
      parts.push(`${result.seats_created} seats`)
    }
    if (result.email_template_copied) {
      parts.push('email template')
    }
    message.value = parts.length
      ? `Imported ${parts.join(', ')}`
      : 'Import completed'
  } catch (err) {
    error.value = getErrorMessage(err)
  }
}
</script>

<template>
  <div class="space-y-6">
    <PageHeader
      :title="event?.name ?? 'Event'"
      description="Overview, status, finalize seating, and recent jobs."
    />

    <LoadingSpinner v-if="isLoading" />

    <div v-else class="grid gap-6 lg:grid-cols-[1.2fr_1fr]">
      <Card title="Event details">
        <Alert v-if="message" tone="success" :title="message" class="mb-4" />
        <Alert v-if="error" tone="error" :title="error" class="mb-4" />
        <dl class="grid gap-3 text-sm sm:grid-cols-2">
          <div>
            <dt class="text-ink-muted">Status</dt>
            <dd><Badge>{{ titleCase(event?.status) }}</Badge></dd>
          </div>
          <div>
            <dt class="text-ink-muted">Seating phase</dt>
            <dd><Badge>{{ titleCase(seatingPhase) }}</Badge></dd>
          </div>
          <div>
            <dt class="text-ink-muted">Dates</dt>
            <dd class="font-medium">
              {{ formatDate(event?.start_date) }}
              <span v-if="event?.end_date"> – {{ formatDate(event?.end_date) }}</span>
            </dd>
          </div>
          <div class="sm:col-span-2">
            <dt class="text-ink-muted">Description</dt>
            <dd class="font-medium">{{ event?.description || '—' }}</dd>
          </div>
        </dl>

        <div v-if="isStaff" class="mt-6 grid gap-3 sm:grid-cols-[1fr_auto] sm:items-end">
          <Select
            v-model="status"
            label="Update status"
            placeholder="Select status"
            :options="[
              { label: 'Draft', value: 'draft' },
              { label: 'Active', value: 'active' },
              { label: 'Cancelled', value: 'cancelled' },
              { label: 'Completed', value: 'completed' },
            ]"
          />
          <Button :loading="updateMutation.isPending.value" @click="saveStatus">Save</Button>
        </div>

        <p v-if="currentJob" class="mt-3 text-sm text-ink-muted">
          Job {{ currentJob.id }}: {{ titleCase(currentJob.status) }}
        </p>
      </Card>

      <Card v-if="isStaff" title="Recent jobs">
        <div v-if="jobs?.length" class="space-y-3">
          <div
            v-for="job in jobs.slice(0, 5)"
            :key="job.id"
            class="rounded-xl border border-border px-3 py-3 text-sm"
          >
            <p class="font-medium">{{ titleCase(job.type) }}</p>
            <p class="text-ink-muted">{{ titleCase(job.status) }}</p>
          </div>
        </div>
        <p v-else class="text-sm text-ink-muted">No jobs yet.</p>
      </Card>
    </div>

    <Card v-if="isStaff" title="Import config">
      <p class="mb-4 text-sm text-ink-muted">
        Copy seat categories, seat layout, and/or invitation email from another event you manage.
        Does not copy guests or bookings. Target must have no categories or seats when importing
        layout.
      </p>
      <div class="grid gap-4 lg:grid-cols-2">
        <Select
          v-model="sourceEventId"
          label="Source event"
          placeholder="Select source event"
          :options="sourceEventOptions"
        />
        <div class="space-y-3 text-sm text-ink">
          <label class="flex items-center gap-2">
            <input v-model="includeCategories" type="checkbox" class="rounded border-border" />
            Seat categories
          </label>
          <label class="flex items-center gap-2">
            <input
              v-model="includeSeats"
              type="checkbox"
              class="rounded border-border"
              :disabled="!includeCategories"
            />
            Seat layout
          </label>
          <label class="flex items-center gap-2">
            <input
              v-model="includeEmailTemplate"
              type="checkbox"
              class="rounded border-border"
            />
            Invitation email template
          </label>
        </div>
      </div>
      <div class="mt-4">
        <Button
          :loading="importMutation.isPending.value"
          :disabled="!sourceEventId || (!includeCategories && !includeSeats && !includeEmailTemplate)"
          @click="importConfig"
        >
          Import config
        </Button>
      </div>
    </Card>

    <Card v-if="isStaff" title="Seat assignment">
      <p class="mb-4 text-sm text-ink-muted">
        Phase:
        <Badge class="ml-1">{{ titleCase(seatingPhase) }}</Badge>
      </p>

      <div v-if="seatingPhase === 'open'" class="space-y-3">
        <p class="text-sm text-ink-muted">
          Assign seats to guests automatically. Requires guests, available seats, and the background
          worker running.
        </p>
        <Alert v-if="finalizeInProgress" tone="info" title="Assigning seats…">
          <p class="mt-1 text-ink-muted">
            Keep this page open or check Jobs. Seating preview appears when the worker finishes.
          </p>
        </Alert>
        <Button
          :loading="finalizeMutation.isPending.value || finalizeInProgress"
          :disabled="finalizeInProgress"
          @click="finalizeSeating"
        >
          Finalize seating
        </Button>
      </div>

      <div v-else-if="seatingPhase === 'preview'" class="space-y-3">
        <p class="text-sm text-ink-muted">
          Review assignments below. Approve to send invitations, or Reject to go back and finalize again.
        </p>
        <div class="flex flex-wrap gap-3">
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
            Reject
          </Button>
        </div>
      </div>

      <div v-else-if="seatingPhase === 'approved'" class="space-y-3">
        <Alert tone="success" title="Seating approved — invitations queued or sent." />
        <p class="text-sm text-ink-muted">
          Need to assign seats again? Re-open seating to return to the open phase and run Finalize.
        </p>
        <Button
          variant="secondary"
          :loading="rejectMutation.isPending.value"
          @click="rejectSeating"
        >
          Re-open seating
        </Button>
      </div>
    </Card>

    <Card v-if="isStaff && showPreview" title="Seating preview">
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
        v-else-if="previewEmpty && seatingPhase === 'preview'"
        tone="warning"
        title="No seat assignments"
        class="mb-4"
      >
        <p class="mt-1 text-ink-muted">
          Guests may be missing, seats unavailable, or finalize did not run. Click Reject to return
          to open and try again. Ensure the background worker is running
          (<code class="text-xs">go run ./cmd/worker</code>).
        </p>
      </Alert>
      <div v-else-if="previewRows?.length" class="overflow-x-auto">
        <table class="w-full text-left text-sm">
          <thead>
            <tr class="border-b border-border text-ink-muted">
              <th class="px-3 py-2 font-medium">Guest</th>
              <th class="px-3 py-2 font-medium">Email</th>
              <th class="px-3 py-2 font-medium">Seat</th>
              <th class="px-3 py-2 font-medium">Category</th>
              <th class="px-3 py-2 font-medium">Section</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="row in previewRows"
              :key="row.booking_id"
              class="border-b border-border/60"
            >
              <td class="px-3 py-2 font-medium">{{ row.guest_name }}</td>
              <td class="px-3 py-2">{{ row.guest_email }}</td>
              <td class="px-3 py-2">{{ row.seat_code }}</td>
              <td class="px-3 py-2">
                {{ row.category_name }}
                <span v-if="row.category_code" class="text-ink-muted">({{ row.category_code }})</span>
              </td>
              <td class="px-3 py-2">{{ row.section || '—' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <p v-else class="text-sm text-ink-muted">No assignments yet.</p>
    </Card>
  </div>
</template>
