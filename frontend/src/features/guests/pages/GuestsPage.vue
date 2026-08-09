<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import type { ColumnDef } from '@tanstack/vue-table'
import PageHeader from '@/shared/ui/PageHeader.vue'
import Button from '@/shared/ui/Button.vue'
import Input from '@/shared/ui/Input.vue'
import Select from '@/shared/ui/Select.vue'
import Modal from '@/shared/ui/Modal.vue'
import DataTable from '@/shared/ui/DataTable.vue'
import Alert from '@/shared/ui/Alert.vue'
import { getErrorMessage } from '@/shared/lib/errors'
import { formatDateTime } from '@/shared/lib/format'
import { useDebouncedSearch } from '@/shared/composables/useDebouncedSearch'
import { useCategoriesQuery } from '@/features/categories/queries/categories'
import { useEventOrgRole } from '@/features/organizations/composables/useEventOrgRole'
import {
  useCreateGuestMutation,
  useGuestsQuery,
  useImportGuestsMutation,
} from '../queries/guests'
import type { GuestImportResult, ListGuestsParams } from '../api/guests'
import type { Guest } from '@/shared/api/types'

const route = useRoute()
const eventId = computed(() => route.params.eventId as string)
const guestImportTemplateUrl = `${import.meta.env.BASE_URL}templates/guests_import.xlsx`

const showCreate = ref(false)
const error = ref('')
const message = ref('')
const importResult = ref<GuestImportResult | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)
const page = ref(1)
const pageSize = ref(20)
const { searchInput, searchQuery } = useDebouncedSearch(() => {
  page.value = 1
})

const form = reactive({
  name: '',
  email: '',
  category_id: '',
  ticket_count: '1',
})

const listFilters = computed<ListGuestsParams>(() => ({
  page: page.value,
  page_size: pageSize.value,
  q: searchQuery.value || undefined,
}))

const { data: guestsPage, isLoading } = useGuestsQuery(eventId, listFilters)
const { data: categories } = useCategoriesQuery(eventId)
const { isStaff } = useEventOrgRole(eventId)
const createMutation = useCreateGuestMutation(eventId)
const importMutation = useImportGuestsMutation(eventId)

const guestRows = computed(() => guestsPage.value?.items ?? [])
const guestTotal = computed(() => guestsPage.value?.total ?? 0)

const categoryOptions = computed(
  () => categories.value?.map((c) => ({ label: c.name ?? 'Category', value: c.id! })) ?? [],
)

const guestColumns = computed<ColumnDef<Guest>[]>(() => [
  {
    accessorKey: 'name',
    header: 'Name',
    cell: ({ row }) => row.original.name || '—',
  },
  {
    accessorKey: 'email',
    header: 'Email',
    cell: ({ row }) => row.original.email || '—',
  },
  {
    accessorKey: 'ticket_count',
    header: 'Tickets',
  },
  {
    id: 'paid',
    header: 'Paid',
    cell: ({ row }) =>
      row.original.paid_date ? formatDateTime(row.original.paid_date) : 'Not yet',
  },
])

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
          :href="guestImportTemplateUrl"
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

    <Input
      v-model="searchInput"
      label="Search guests"
      placeholder="Name or email"
      autocomplete="off"
    />

    <DataTable
      :columns="guestColumns"
      :rows="guestRows"
      :loading="isLoading"
      :page="page"
      :page-size="pageSize"
      :total="guestTotal"
      empty-message="No guests yet. Add guests before creating bookings."
      @update:page="page = $event"
      @update:page-size="pageSize = $event"
    />

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
