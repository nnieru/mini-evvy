<script setup lang="ts">
import { computed, h, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import type { ColumnDef } from '@tanstack/vue-table'
import PageHeader from '@/shared/ui/PageHeader.vue'
import Button from '@/shared/ui/Button.vue'
import Input from '@/shared/ui/Input.vue'
import Select from '@/shared/ui/Select.vue'
import Badge from '@/shared/ui/Badge.vue'
import Modal from '@/shared/ui/Modal.vue'
import DataTable from '@/shared/ui/DataTable.vue'
import Alert from '@/shared/ui/Alert.vue'
import { getErrorMessage } from '@/shared/lib/errors'
import { titleCase } from '@/shared/lib/format'
import { useDebouncedSearch } from '@/shared/composables/useDebouncedSearch'
import { useCategoriesQuery } from '@/features/categories/queries/categories'
import { useCreateSeatsMutation, useSeatsQuery } from '../queries/seats'
import type { ListSeatsParams } from '../api/seats'
import type { Seat } from '@/shared/api/types'

const route = useRoute()
const eventId = computed(() => route.params.eventId as string)

const showCreate = ref(false)
const error = ref('')
const page = ref(1)
const pageSize = ref(20)
const statusFilter = ref('')
const categoryFilter = ref('')
const { searchInput, searchQuery } = useDebouncedSearch(() => {
  page.value = 1
})

const form = reactive({
  code: '',
  category_id: '',
  section: '',
  row: '',
  col: '',
  status: 'available',
})

const listFilters = computed<ListSeatsParams>(() => ({
  page: page.value,
  page_size: pageSize.value,
  q: searchQuery.value || undefined,
  status: statusFilter.value || undefined,
  category_id: categoryFilter.value || undefined,
}))

const { data: seatsPage, isLoading } = useSeatsQuery(eventId, listFilters)
const { data: categories } = useCategoriesQuery(eventId)
const createMutation = useCreateSeatsMutation(eventId)

const seatRows = computed(() => seatsPage.value?.items ?? [])
const seatTotal = computed(() => seatsPage.value?.total ?? 0)

const categoryNameById = computed(() => {
  const map = new Map<string, string>()
  for (const category of categories.value ?? []) {
    if (category.id) map.set(category.id, category.name ?? 'Category')
  }
  return map
})

const categoryOptions = computed(
  () => categories.value?.map((c) => ({ label: c.name ?? 'Category', value: c.id! })) ?? [],
)

const seatColumns = computed<ColumnDef<Seat>[]>(() => [
  {
    accessorKey: 'code',
    header: 'Code',
    cell: ({ row }) => row.original.code || '—',
  },
  {
    id: 'location',
    header: 'Location',
    cell: ({ row }) => {
      const parts = [row.original.section || 'No section']
      if (row.original.row != null) parts.push(`Row ${row.original.row}`)
      if (row.original.col != null) parts.push(`Col ${row.original.col}`)
      return parts.join(' · ')
    },
  },
  {
    id: 'category',
    header: 'Category',
    cell: ({ row }) =>
      row.original.category_id
        ? categoryNameById.value.get(row.original.category_id) ?? '—'
        : '—',
  },
  {
    id: 'status',
    header: 'Status',
    cell: ({ row }) =>
      h(Badge, {}, () => titleCase(row.original.status ?? '')),
  },
])

async function createSeat() {
  error.value = ''
  try {
    await createMutation.mutateAsync({
      seats: [
        {
          code: form.code,
          category_id: form.category_id,
          section: form.section || null,
          row: form.row ? Number(form.row) : null,
          col: form.col ? Number(form.col) : null,
          status: form.status as 'available' | 'reserved' | 'occupied' | 'blocked',
        },
      ],
    })
    showCreate.value = false
    form.code = ''
    form.section = ''
    form.row = ''
    form.col = ''
  } catch (err) {
    error.value = getErrorMessage(err)
  }
}
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <PageHeader title="Seats" description="Manage seat inventory and availability." />
      <Button @click="showCreate = true">Add seat</Button>
    </div>

    <div class="grid gap-3 sm:grid-cols-3">
      <Input
        v-model="searchInput"
        label="Search seats"
        placeholder="Code or section"
        autocomplete="off"
      />
      <Select
        v-model="statusFilter"
        label="Status"
        placeholder="All statuses"
        :options="[
          { label: 'Available', value: 'available' },
          { label: 'Reserved', value: 'reserved' },
          { label: 'Occupied', value: 'occupied' },
          { label: 'Blocked', value: 'blocked' },
        ]"
        @update:model-value="page = 1"
      />
      <Select
        v-model="categoryFilter"
        label="Category"
        placeholder="All categories"
        :options="categoryOptions"
        @update:model-value="page = 1"
      />
    </div>

    <DataTable
      :columns="seatColumns"
      :rows="seatRows"
      :loading="isLoading"
      :page="page"
      :page-size="pageSize"
      :total="seatTotal"
      empty-message="No seats yet. Create categories first, then add seats."
      @update:page="page = $event"
      @update:page-size="pageSize = $event"
    />

    <Modal :open="showCreate" title="Add seat" @close="showCreate = false">
      <form class="space-y-4" @submit.prevent="createSeat">
        <Alert v-if="error" tone="error" :title="error" />
        <Input v-model="form.code" label="Seat code" required />
        <Select
          v-model="form.category_id"
          label="Category"
          placeholder="Select category"
          :options="categoryOptions"
          required
        />
        <Input v-model="form.section" label="Section" />
        <div class="grid gap-3 sm:grid-cols-2">
          <Input v-model="form.row" label="Row" type="number" />
          <Input v-model="form.col" label="Column" type="number" />
        </div>
        <Select
          v-model="form.status"
          label="Status"
          :options="[
            { label: 'Available', value: 'available' },
            { label: 'Reserved', value: 'reserved' },
            { label: 'Occupied', value: 'occupied' },
            { label: 'Blocked', value: 'blocked' },
          ]"
        />
        <div class="flex justify-end gap-2">
          <Button variant="secondary" type="button" @click="showCreate = false">Cancel</Button>
          <Button type="submit" :loading="createMutation.isPending.value">Create</Button>
        </div>
      </form>
    </Modal>
  </div>
</template>
