<script setup lang="ts">
import { computed, h, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import type { ColumnDef } from '@tanstack/vue-table'
import PageHeader from '@/shared/ui/PageHeader.vue'
import Card from '@/shared/ui/Card.vue'
import Button from '@/shared/ui/Button.vue'
import Input from '@/shared/ui/Input.vue'
import Select from '@/shared/ui/Select.vue'
import Badge from '@/shared/ui/Badge.vue'
import DataTable from '@/shared/ui/DataTable.vue'
import Alert from '@/shared/ui/Alert.vue'
import BarcodeCameraScanner from '../components/BarcodeCameraScanner.vue'
import { getErrorMessage } from '@/shared/lib/errors'
import { formatDateTime, titleCase } from '@/shared/lib/format'
import { useDebouncedSearch } from '@/shared/composables/useDebouncedSearch'
import { useAllGuestsQuery } from '@/features/guests/queries/guests'
import { useAllSeatsQuery } from '@/features/seats/queries/seats'
import {
  useAttendanceQuery,
  useCheckInMutation,
  useUndoCheckInMutation,
} from '../queries/attendance'
import type { ListAttendanceParams } from '../api/attendance'
import type { Attendance } from '@/shared/api/types'

const route = useRoute()
const eventId = computed(() => route.params.eventId as string)

const mode = ref<'camera' | 'manual' | 'guest'>('camera')
const error = ref('')
const success = ref('')
const scannerPaused = ref(false)
const page = ref(1)
const pageSize = ref(20)
const { searchInput, searchQuery } = useDebouncedSearch(() => {
  page.value = 1
})

const manualForm = reactive({ barcode: '' })
const guestForm = reactive({ guest_id: '', seat_id: '', message: '' })

const listFilters = computed<ListAttendanceParams>(() => ({
  page: page.value,
  page_size: pageSize.value,
  q: searchQuery.value || undefined,
}))

const { data: attendancePage, isLoading } = useAttendanceQuery(eventId, listFilters)
const { data: guests } = useAllGuestsQuery(eventId)
const { data: seats } = useAllSeatsQuery(eventId)
const checkInMutation = useCheckInMutation(eventId)
const undoMutation = useUndoCheckInMutation(eventId)

const attendanceRows = computed(() => attendancePage.value?.items ?? [])
const attendanceTotal = computed(() => attendancePage.value?.total ?? 0)

const guestOptions = computed(
  () => guests.value?.map((g) => ({ label: g.name!, value: g.id! })) ?? [],
)
const seatOptions = computed(
  () => seats.value?.map((s) => ({ label: s.code!, value: s.id! })) ?? [],
)

const attendanceColumns = computed<ColumnDef<Attendance>[]>(() => [
  {
    id: 'guest',
    header: 'Guest',
    cell: ({ row }) => row.original.guest_id?.slice(0, 8) ?? '—',
  },
  {
    id: 'seat',
    header: 'Seat',
    cell: ({ row }) => row.original.seat_id?.slice(0, 8) ?? '—',
  },
  {
    id: 'message',
    header: 'Message',
    cell: ({ row }) => row.original.message || '—',
  },
  {
    id: 'checked_at',
    header: 'Checked in',
    cell: ({ row }) => formatDateTime(row.original.created_at),
  },
  {
    id: 'status',
    header: 'Status',
    cell: ({ row }) =>
      h(
        Badge,
        { tone: row.original.status === 'checked_in' ? 'success' : 'muted' },
        () => titleCase(row.original.status ?? ''),
      ),
  },
  {
    id: 'actions',
    header: '',
    cell: ({ row }) =>
      row.original.status === 'checked_in' && row.original.id
        ? h(
            Button,
            {
              size: 'sm',
              variant: 'secondary',
              onClick: () => undo(row.original.id!),
            },
            () => 'Undo',
          )
        : null,
  },
])

async function checkIn(body: { barcode?: string; guest_id?: string; seat_id?: string; message?: string | null }) {
  error.value = ''
  success.value = ''
  scannerPaused.value = true
  try {
    await checkInMutation.mutateAsync(body)
    success.value = 'Guest checked in successfully'
    manualForm.barcode = ''
    guestForm.message = ''
  } catch (err) {
    error.value = getErrorMessage(err)
  } finally {
    setTimeout(() => {
      scannerPaused.value = false
    }, 1500)
  }
}

async function onScan(barcode: string) {
  await checkIn({ barcode })
}

async function manualCheckIn() {
  if (!manualForm.barcode.trim()) return
  await checkIn({ barcode: manualForm.barcode.trim() })
}

async function guestCheckIn() {
  await checkIn({
    guest_id: guestForm.guest_id,
    seat_id: guestForm.seat_id,
    message: guestForm.message || null,
  })
}

async function undo(attendanceId: string) {
  error.value = ''
  try {
    await undoMutation.mutateAsync(attendanceId)
  } catch (err) {
    error.value = getErrorMessage(err)
  }
}
</script>

<template>
  <div class="space-y-6">
    <PageHeader
      title="Check-in"
      description="Scan tickets with your phone or laptop camera, or enter a barcode manually."
    />

    <Alert v-if="success" tone="success" :title="success" />
    <Alert v-if="error" tone="error" :title="error" />

    <div class="flex flex-wrap gap-2">
      <Button :variant="mode === 'camera' ? 'primary' : 'secondary'" @click="mode = 'camera'">
        Camera
      </Button>
      <Button :variant="mode === 'manual' ? 'primary' : 'secondary'" @click="mode = 'manual'">
        Manual barcode
      </Button>
      <Button :variant="mode === 'guest' ? 'primary' : 'secondary'" @click="mode = 'guest'">
        Guest + seat
      </Button>
    </div>

    <Card v-if="mode === 'camera'" title="Camera scanner">
      <BarcodeCameraScanner :paused="scannerPaused" @scan="onScan" />
    </Card>

    <Card v-else-if="mode === 'manual'" title="Manual barcode">
      <form class="space-y-4" @submit.prevent="manualCheckIn">
        <Input
          v-model="manualForm.barcode"
          label="Barcode"
          placeholder="EVVY-..."
          autocomplete="off"
        />
        <Button type="submit" :loading="checkInMutation.isPending.value">Check in</Button>
      </form>
    </Card>

    <Card v-else title="Guest and seat">
      <form class="space-y-4" @submit.prevent="guestCheckIn">
        <Select
          v-model="guestForm.guest_id"
          label="Guest"
          placeholder="Select guest"
          :options="guestOptions"
          required
        />
        <Select
          v-model="guestForm.seat_id"
          label="Seat"
          placeholder="Select seat"
          :options="seatOptions"
          required
        />
        <Input v-model="guestForm.message" label="Message" />
        <Button type="submit" :loading="checkInMutation.isPending.value">Check in</Button>
      </form>
    </Card>

    <Card title="Attendance log">
      <div class="mb-4">
        <Input
          v-model="searchInput"
          label="Search log"
          placeholder="Guest, seat, barcode, or message"
          autocomplete="off"
        />
      </div>

      <DataTable
        :columns="attendanceColumns"
        :rows="attendanceRows"
        :loading="isLoading"
        :page="page"
        :page-size="pageSize"
        :total="attendanceTotal"
        empty-message="No check-ins yet."
        @update:page="page = $event"
        @update:page-size="pageSize = $event"
      />
    </Card>
  </div>
</template>
