<script setup lang="ts">
import { computed, ref } from 'vue'
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
import { useEventOrgRole } from '@/features/organizations/composables/useEventOrgRole'
import {
  useEventQuery,
  useImportEventConfigMutation,
  useMyEventsQuery,
  useUpdateEventMutation,
} from '../queries/events'
import { useJobsQuery } from '@/features/jobs/queries/jobs'

const route = useRoute()
const eventId = computed(() => route.params.eventId as string)

const status = ref('')
const message = ref('')
const error = ref('')

const { data: event, isLoading } = useEventQuery(eventId)
const { isStaff } = useEventOrgRole(eventId)
const jobListFilters = computed(() => ({ page: 1, page_size: 100 }))
const { data: jobsPage } = useJobsQuery(eventId, jobListFilters)
const jobs = computed(() => jobsPage.value?.items ?? [])
const updateMutation = useUpdateEventMutation(eventId)
const importMutation = useImportEventConfigMutation(eventId)
const { data: myEvents } = useMyEventsQuery(isStaff)

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

async function saveStatus() {
  if (!status.value) return
  error.value = ''
  try {
    await updateMutation.mutateAsync({
      status: status.value as 'draft' | 'active' | 'cancelled' | 'completed',
    })
    message.value = 'Event status updated'
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
    message.value = parts.length ? `Imported ${parts.join(', ')}` : 'Import completed'
  } catch (err) {
    error.value = getErrorMessage(err)
  }
}
</script>

<template>
  <div class="space-y-6">
    <PageHeader
      :title="event?.name ?? 'Event'"
      description="Overview, status, and import config. Manage seating drafts on the Seating draft page."
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
  </div>
</template>
