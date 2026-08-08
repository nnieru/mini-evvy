<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import PageHeader from '@/shared/ui/PageHeader.vue'
import Card from '@/shared/ui/Card.vue'
import Button from '@/shared/ui/Button.vue'
import Input from '@/shared/ui/Input.vue'
import Select from '@/shared/ui/Select.vue'
import Modal from '@/shared/ui/Modal.vue'
import LoadingSpinner from '@/shared/ui/LoadingSpinner.vue'
import EmptyState from '@/shared/ui/EmptyState.vue'
import Alert from '@/shared/ui/Alert.vue'
import { getErrorMessage } from '@/shared/lib/errors'
import { formatDateTime } from '@/shared/lib/format'
import { useCategoriesQuery } from '@/features/categories/queries/categories'
import { useEventOrgRole } from '@/features/organizations/composables/useEventOrgRole'
import {
  useCreateGuestMutation,
  useGuestsQuery,
  useImportGuestsMutation,
} from '../queries/guests'
import type { GuestImportResult } from '../api/guests'

const route = useRoute()
const eventId = computed(() => route.params.eventId as string)

const showCreate = ref(false)
const error = ref('')
const message = ref('')
const importResult = ref<GuestImportResult | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)

const form = reactive({
  name: '',
  email: '',
  category_id: '',
  ticket_count: '1',
})

const { data: guests, isLoading } = useGuestsQuery(eventId)
const { data: categories } = useCategoriesQuery(eventId)
const { isStaff } = useEventOrgRole(eventId)
const createMutation = useCreateGuestMutation(eventId)
const importMutation = useImportGuestsMutation(eventId)

const categoryOptions = computed(
  () => categories.value?.map((c) => ({ label: c.name ?? 'Category', value: c.id! })) ?? [],
)

async function createGuest() {
  error.value = ''
  try {
    await createMutation.mutateAsync({
      name: form.name,
      email: form.email,
      category_id: form.category_id,
      ticket_count: Number(form.ticket_count),
    })
    showCreate.value = false
    form.name = ''
    form.email = ''
    form.ticket_count = '1'
  } catch (err) {
    error.value = getErrorMessage(err)
  }
}

function openImport() {
  fileInput.value?.click()
}

async function onImportFile(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return

  error.value = ''
  message.value = ''
  importResult.value = null

  try {
    const result = await importMutation.mutateAsync(file)
    importResult.value = result
    message.value = `Import done: ${result.created} created, ${result.updated} updated, ${result.failed} failed`
  } catch (err) {
    error.value = getErrorMessage(err)
  }
}
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <PageHeader title="Guests" description="Invitees and ticket holders for this event." />
      <div class="flex flex-wrap gap-2">
        <a
          v-if="isStaff"
          href="/templates/guests_import.xlsx"
          download
          class="inline-flex items-center justify-center gap-2 rounded-lg border border-border bg-surface-raised px-4 py-2 text-sm font-medium text-ink transition-colors hover:bg-surface"
        >
          Download template
        </a>
        <Button
          v-if="isStaff"
          variant="secondary"
          :loading="importMutation.isPending.value"
          @click="openImport"
        >
          Import guests
        </Button>
        <Button @click="showCreate = true">Add guest</Button>
      </div>
    </div>

    <input
      ref="fileInput"
      type="file"
      accept=".xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
      class="hidden"
      @change="onImportFile"
    />

    <Alert v-if="message" tone="success" :title="message" />
    <Alert v-if="error" tone="error" :title="error" />

    <div
      v-if="importResult?.errors?.length"
      class="rounded-xl border border-warning/20 bg-warning-soft px-4 py-3 text-sm text-warning"
    >
      <p class="font-medium">Row errors</p>
      <ul class="mt-2 list-disc space-y-1 pl-5">
        <li v-for="rowErr in importResult.errors.slice(0, 10)" :key="rowErr.row">
          Row {{ rowErr.row }}: {{ rowErr.message }}
        </li>
      </ul>
    </div>

    <LoadingSpinner v-if="isLoading" />
    <EmptyState v-else-if="!guests?.length" title="No guests yet" description="Add guests before creating bookings." />

    <div v-else class="grid gap-4 sm:grid-cols-2">
      <Card v-for="guest in guests" :key="guest.id">
        <h3 class="font-display text-lg font-semibold text-ink">{{ guest.name }}</h3>
        <p class="text-sm text-ink-muted">{{ guest.email }}</p>
        <p class="mt-3 text-sm">Tickets: {{ guest.ticket_count }}</p>
        <p class="text-sm text-ink-muted">
          Paid: {{ guest.paid_date ? formatDateTime(guest.paid_date) : 'Not yet' }}
        </p>
      </Card>
    </div>

    <Modal :open="showCreate" title="Add guest" @close="showCreate = false">
      <form class="space-y-4" @submit.prevent="createGuest">
        <Alert v-if="error" tone="error" :title="error" />
        <Input v-model="form.name" label="Name" required />
        <Input v-model="form.email" label="Email" type="email" required />
        <Select
          v-model="form.category_id"
          label="Category"
          placeholder="Select category"
          :options="categoryOptions"
          required
        />
        <Input v-model="form.ticket_count" label="Ticket count" type="number" min="1" required />
        <div class="flex justify-end gap-2">
          <Button variant="secondary" type="button" @click="showCreate = false">Cancel</Button>
          <Button type="submit" :loading="createMutation.isPending.value">Create</Button>
        </div>
      </form>
    </Modal>
  </div>
</template>
