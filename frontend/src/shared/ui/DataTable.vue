<script setup lang="ts" generic="T">
import { computed } from 'vue'
import {
  FlexRender,
  getCoreRowModel,
  useVueTable,
  type ColumnDef,
} from '@tanstack/vue-table'
import Button from '@/shared/ui/Button.vue'
import LoadingSpinner from '@/shared/ui/LoadingSpinner.vue'

const props = withDefaults(
  defineProps<{
    columns: ColumnDef<T, unknown>[]
    rows: T[]
    loading?: boolean
    page: number
    pageSize: number
    total: number
    emptyMessage?: string
  }>(),
  {
    loading: false,
    emptyMessage: 'No rows to display.',
  },
)

const emit = defineEmits<{
  'update:page': [page: number]
  'update:pageSize': [pageSize: number]
}>()

const pageCount = computed(() => Math.max(1, Math.ceil(props.total / props.pageSize)))

const rangeStart = computed(() => {
  if (props.total === 0) return 0
  return (props.page - 1) * props.pageSize + 1
})

const rangeEnd = computed(() => Math.min(props.page * props.pageSize, props.total))

const table = useVueTable({
  get data() {
    return props.rows
  },
  get columns() {
    return props.columns
  },
  getCoreRowModel: getCoreRowModel(),
  manualPagination: true,
  get pageCount() {
    return pageCount.value
  },
})

function goPrev() {
  if (props.page > 1) emit('update:page', props.page - 1)
}

function goNext() {
  if (props.page < pageCount.value) emit('update:page', props.page + 1)
}

function onPageSizeChange(event: Event) {
  const value = Number((event.target as HTMLSelectElement).value)
  emit('update:pageSize', value)
  emit('update:page', 1)
}
</script>

<template>
  <div class="space-y-3">
    <LoadingSpinner v-if="loading" />
    <p v-else-if="!rows.length" class="text-sm text-ink-muted">{{ emptyMessage }}</p>
    <div v-else class="overflow-x-auto rounded-xl border border-border">
      <table class="w-full min-w-[640px] text-left text-sm">
        <thead>
          <tr
            v-for="headerGroup in table.getHeaderGroups()"
            :key="headerGroup.id"
            class="border-b border-border bg-surface text-ink-muted"
          >
            <th
              v-for="header in headerGroup.headers"
              :key="header.id"
              class="px-3 py-2 font-medium"
            >
              <FlexRender
                v-if="!header.isPlaceholder"
                :render="header.column.columnDef.header"
                :props="header.getContext()"
              />
            </th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="row in table.getRowModel().rows"
            :key="row.id"
            class="border-b border-border/60 last:border-b-0"
          >
            <td
              v-for="cell in row.getVisibleCells()"
              :key="cell.id"
              class="px-3 py-3 align-top"
            >
              <FlexRender :render="cell.column.columnDef.cell" :props="cell.getContext()" />
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div
      v-if="total > 0"
      class="flex flex-col gap-3 text-sm text-ink-muted sm:flex-row sm:items-center sm:justify-between"
    >
      <p>
        Showing {{ rangeStart }}–{{ rangeEnd }} of {{ total }}
      </p>
      <div class="flex flex-wrap items-center gap-2">
        <label class="flex items-center gap-2">
          <span>Rows</span>
          <select
            class="rounded-lg border border-border bg-surface px-2 py-1 text-ink"
            :value="pageSize"
            @change="onPageSizeChange"
          >
            <option :value="10">10</option>
            <option :value="20">20</option>
            <option :value="50">50</option>
          </select>
        </label>
        <Button size="sm" variant="secondary" :disabled="page <= 1" @click="goPrev">
          Previous
        </Button>
        <span>Page {{ page }} / {{ pageCount }}</span>
        <Button size="sm" variant="secondary" :disabled="page >= pageCount" @click="goNext">
          Next
        </Button>
      </div>
    </div>
  </div>
</template>
