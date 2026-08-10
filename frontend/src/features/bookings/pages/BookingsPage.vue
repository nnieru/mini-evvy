<script setup lang="ts">
import { computed, h, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import type { ColumnDef } from '@tanstack/vue-table'
import PageHeader from '@/shared/ui/PageHeader.vue'
import Button from '@/shared/ui/Button.vue'
import Input from '@/shared/ui/Input.vue'
import Select from '@/shared/ui/Select.vue'
import Badge from '@/shared/ui/Badge.vue'
import Modal from '@/shared/ui/Modal.vue'
import LoadingSpinner from '@/shared/ui/LoadingSpinner.vue'
import EmptyState from '@/shared/ui/EmptyState.vue'
import Alert from '@/shared/ui/Alert.vue'
import DataTable from '@/shared/ui/DataTable.vue'
import { getErrorMessage } from '@/shared/lib/errors'
import { useDebouncedSearch } from '@/shared/composables/useDebouncedSearch'
import { useJobPoller } from '@/shared/composables/useJobPoller'
import { useCategoriesQuery } from '@/features/categories/queries/categories'
import { useAllSeatsQuery } from '@/features/seats/queries/seats'
import SeatGrid from '../components/SeatGrid.vue'
import { buildCategoryColorMap } from '../lib/categoryColors'
import {
  useBookingsQuery,
  useCreateBookingsBatchMutation,
  useDeleteBookingMutation,
  useResendInvitationMutation,
  useUpdateBookingMutation,
} from '../queries/bookings'
import type { ListBookingsParams } from '../api/bookings'
import { useCreatePaymentMutation } from '@/features/payments/queries/payments'
import { useEventQuery } from '@/features/events/queries/events'
import type { BookingListItem, Seat } from '@/shared/api/types'

const route = useRoute()
const eventId = computed(() => route.params.eventId as string)

const showCreate = ref(false)
const showPayment = ref(false)
const selectedBookingId = ref('')
const error = ref('')
const message = ref('')
const selectedSeatIds = ref<string[]>([])
const differentGuests = ref(false)

const form = reactive({
  sharedName: '',
  sharedEmail: '',
  notes: '',
})

const guestBySeatId = reactive<Record<string, { name: string; email: string }>>({})

const paymentForm = reactive({
  amount: '',
  currency: 'IDR',
  status: 'success',
  method: '',
})

type PaymentFilter = 'all' | 'paid' | 'unpaid'

const paymentFilter = ref<PaymentFilter>('all')
const page = ref(1)
const pageSize = ref(20)
const { searchInput, searchQuery } = useDebouncedSearch(() => {
  page.value = 1
})

const listFilters = computed<ListBookingsParams>(() => ({
  page: page.value,
  page_size: pageSize.value,
  payment_status: paymentFilter.value,
  q: searchQuery.value || undefined,
}))

const { data: bookingsPage, isLoading: bookingsLoading } = useBookingsQuery(eventId, listFilters)
const { data: event } = useEventQuery(eventId)
const { data: categories, isLoading: categoriesLoading } = useCategoriesQuery(eventId)
const { data: allSeats, isLoading: seatsLoading } = useAllSeatsQuery(eventId)

const batchMutation = useCreateBookingsBatchMutation(eventId)
const updateMutation = useUpdateBookingMutation(eventId)
const deleteMutation = useDeleteBookingMutation(eventId)
const resendMutation = useResendInvitationMutation()
const paymentMutation = useCreatePaymentMutation(
  computed(() => selectedBookingId.value),
  eventId,
)
const { pollJob } = useJobPoller()

const categoryColorById = computed(() => {
  const ids = categories.value?.map((c) => c.id!).filter(Boolean) ?? []
  return buildCategoryColorMap(ids)
})

const legendCategories = computed(() => {
  const seatCounts = new Map<string, number>()
  for (const seat of allSeats.value ?? []) {
    if (seat.category_id) {
      seatCounts.set(seat.category_id, (seatCounts.get(seat.category_id) ?? 0) + 1)
    }
  }
  return (categories.value ?? [])
    .filter((c) => c.id && (seatCounts.get(c.id) ?? 0) > 0)
    .map((c) => ({
      id: c.id!,
      name: c.name ?? 'Category',
      code: c.code,
      color: categoryColorById.value[c.id!],
    }))
})

const seatingLocked = computed(() => event.value?.seating_phase === 'preview')
const seatingApproved = computed(() => event.value?.seating_phase === 'approved')

const seatById = computed(() => {
  const map = new Map<string, Seat>()
  for (const seat of allSeats.value ?? []) {
    if (seat.id) map.set(seat.id, seat)
  }
  return map
})

const selectedSeats = computed(() =>
  selectedSeatIds.value
    .map((id) => seatById.value.get(id))
    .filter((seat): seat is Seat => Boolean(seat)),
)

watch(paymentFilter, () => {
  page.value = 1
})

const bookingRows = computed(() => bookingsPage.value?.items ?? [])
const bookingTotal = computed(() => bookingsPage.value?.total ?? 0)

const emptyBookingMessage = computed(() => {
  if (paymentFilter.value === 'paid') return 'No paid bookings.'
  if (paymentFilter.value === 'unpaid') return 'No unpaid bookings.'
  return 'No active bookings yet. Select seats above or finalize seating.'
})

const paymentFilterOptions: { label: string; value: PaymentFilter }[] = [
  { label: 'All', value: 'all' },
  { label: 'Paid', value: 'paid' },
  { label: 'Unpaid', value: 'unpaid' },
]

const bookingColumns = computed<ColumnDef<BookingListItem>[]>(() => [
  {
    accessorKey: 'seat_code',
    header: 'Seat',
    cell: ({ row }) => row.original.seat_code || '—',
  },
  {
    id: 'guest',
    header: 'Guest',
    cell: ({ row }) =>
      h('div', { class: 'space-y-0.5' }, [
        h('p', { class: 'font-medium text-ink' }, row.original.guest_name || '—'),
        h('p', { class: 'text-ink-muted' }, row.original.guest_email || '—'),
      ]),
  },
  {
    id: 'payment',
    header: 'Payment',
    cell: ({ row }) => {
      const paid = row.original.payment_label === 'paid'
      return h('div', { class: 'space-y-1' }, [
        h(Badge, { tone: paid ? 'success' : 'warning' }, () => (paid ? 'Paid' : 'Unpaid')),
        row.original.status === 'pending'
          ? h('p', { class: 'text-xs text-ink-muted' }, 'Pending invite')
          : null,
      ])
    },
  },
  {
    accessorKey: 'barcode',
    header: 'Barcode',
    cell: ({ row }) => row.original.barcode || '—',
  },
  {
    id: 'actions',
    header: 'Actions',
    cell: ({ row }) => {
      const id = row.original.id!
      const paid = row.original.payment_label === 'paid'
      const actions = [
        h(
          Button,
          {
            size: 'sm',
            variant: 'secondary',
            onClick: () => updateStatus(id, paid ? 'not_paid' : 'paid'),
          },
          () => (paid ? 'Mark unpaid' : 'Mark paid'),
        ),
      ]
      if (!paid) {
        actions.push(
          h(
            Button,
            {
              size: 'sm',
              variant: 'secondary',
              onClick: () => openPayment(id),
            },
            () => 'Add payment',
          ),
        )
      }
      if (paid && seatingApproved.value) {
        actions.push(
          h(
            Button,
            {
              size: 'sm',
              variant: 'secondary',
              onClick: () => resend(id),
            },
            () => 'Resend invite',
          ),
        )
      }
      actions.push(
        h(
          Button,
          {
            size: 'sm',
            variant: 'danger',
            onClick: () => removeBooking(id),
          },
          () => 'Delete',
        ),
      )
      return h('div', { class: 'flex flex-wrap gap-2' }, actions)
    },
  },
])

watch(differentGuests, (enabled) => {
  if (!enabled) return
  for (const seat of selectedSeats.value) {
    if (!seat.id) continue
    if (!guestBySeatId[seat.id]) {
      guestBySeatId[seat.id] = {
        name: form.sharedName,
        email: form.sharedEmail,
      }
    }
  }
})

function toggleSeat(seatId: string) {
  if (seatingLocked.value) return
  const seat = seatById.value.get(seatId)
  if (!seat || seat.status !== 'available') return

  const idx = selectedSeatIds.value.indexOf(seatId)
  if (idx >= 0) {
    selectedSeatIds.value = selectedSeatIds.value.filter((id) => id !== seatId)
    delete guestBySeatId[seatId]
    return
  }

  const firstSelected = selectedSeats.value[0]
  if (firstSelected && firstSelected.category_id !== seat.category_id) {
    clearSelection()
  }

  selectedSeatIds.value = [...selectedSeatIds.value, seatId]
    guestBySeatId[seatId] = {
      name: form.sharedName,
      email: form.sharedEmail,
    }
}

function clearSelection() {
  selectedSeatIds.value = []
  for (const key of Object.keys(guestBySeatId)) {
    delete guestBySeatId[key]
  }
}

function resetCreateForm() {
  form.sharedName = ''
  form.sharedEmail = ''
  form.notes = ''
  differentGuests.value = false
  clearSelection()
}

function openCreate() {
  if (!selectedSeatIds.value.length) return
  showCreate.value = true
}

async function createBookings() {
  error.value = ''
  if (!selectedSeats.value.length) {
    error.value = 'Select at least one seat'
    return
  }

  const items = selectedSeats.value.map((seat) => {
    const guest = differentGuests.value
      ? guestBySeatId[seat.id!]
      : { name: form.sharedName, email: form.sharedEmail }
    return {
      seat_id: seat.id!,
      name: guest?.name ?? '',
      email: guest?.email ?? '',
    }
  })

  if (items.some((item) => !item.name || !item.email)) {
    error.value = 'Name and email are required for each seat'
    return
  }

  try {
    await batchMutation.mutateAsync({
      notes: form.notes || null,
      items,
    })
    showCreate.value = false
    resetCreateForm()
    message.value = `Created ${items.length} booking(s)`
  } catch (err) {
    error.value = getErrorMessage(err)
  }
}

async function updateStatus(bookingId: string, status: string) {
  error.value = ''
  try {
    await updateMutation.mutateAsync({
      bookingId,
      body: { status: status as 'pending' | 'not_paid' | 'paid' | 'cancelled' },
    })
  } catch (err) {
    error.value = getErrorMessage(err)
  }
}

async function removeBooking(bookingId: string) {
  error.value = ''
  try {
    await deleteMutation.mutateAsync(bookingId)
  } catch (err) {
    error.value = getErrorMessage(err)
  }
}

async function resend(bookingId: string) {
  error.value = ''
  message.value = ''
  try {
    const result = await resendMutation.mutateAsync(bookingId)
    await pollJob(result.job_id, {
      onDone: () => {
        message.value = 'Invitation resent'
      },
      onFailed: () => {
        error.value = 'Resend invitation failed'
      },
    })
  } catch (err) {
    error.value = getErrorMessage(err)
  }
}

function openPayment(bookingId: string) {
  selectedBookingId.value = bookingId
  showPayment.value = true
}

async function submitPayment() {
  error.value = ''
  try {
    await paymentMutation.mutateAsync({
      amount: paymentForm.amount,
      currency: paymentForm.currency,
      status: paymentForm.status as 'pending' | 'success' | 'failed' | 'refunded',
      method: paymentForm.method || null,
    })
    showPayment.value = false
    paymentForm.amount = ''
    paymentForm.method = ''
  } catch (err) {
    error.value = getErrorMessage(err)
  }
}

</script>

<template>
  <div class="space-y-6">
    <PageHeader
      title="Bookings"
      description="Manual seating: select seats on the map and enter guest details to create bookings immediately. For bulk auto-assign from uploaded guests, use Seating draft."
    />

    <Alert v-if="message" tone="success" :title="message" />
    <Alert v-if="error" tone="error" :title="error" />
    <Alert v-if="seatingLocked" tone="warning" title="Seating draft open">
      A seating draft is in review. Approve or reject on the Seating draft page before creating new bookings.
    </Alert>

    <section class="space-y-4 rounded-xl border border-border bg-surface-raised p-4">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div class="flex-1 space-y-3">
          <h2 class="font-display text-lg font-semibold text-ink">Seat map</h2>
          <p class="text-sm text-ink-muted">
            {{ selectedSeatIds.length }} seat(s) selected
            <span v-if="selectedSeats.length"> · {{ legendCategories.find((c) => c.id === selectedSeats[0]?.category_id)?.name }}</span>
          </p>
        </div>
        <div class="flex flex-wrap gap-2">
          <Button variant="secondary" :disabled="!selectedSeatIds.length || seatingLocked" @click="clearSelection">
            Clear
          </Button>
          <Button :disabled="!selectedSeatIds.length || seatingLocked" @click="openCreate">
            Continue
          </Button>
        </div>
      </div>

      <LoadingSpinner v-if="categoriesLoading || seatsLoading" />
      <EmptyState
        v-else-if="!categories?.length"
        title="No categories yet"
        description="Add categories on the Categories page first."
      />
      <EmptyState
        v-else-if="!allSeats?.length"
        title="No seats yet"
        description="Add seats on the Seats page first."
      />
      <template v-else>
        <div class="flex flex-wrap gap-3">
          <div
            v-for="category in legendCategories"
            :key="category.id"
            class="inline-flex items-center gap-2 rounded-lg border border-border bg-surface px-3 py-1.5 text-sm"
          >
            <span
              class="h-3.5 w-3.5 shrink-0 rounded-sm border"
              :style="{ backgroundColor: category.color, borderColor: category.color }"
            />
            <span class="font-medium text-ink">{{ category.name }}</span>
            <span v-if="category.code" class="text-ink-muted">({{ category.code }})</span>
          </div>
        </div>
        <SeatGrid
          :seats="allSeats"
          :selected-ids="selectedSeatIds"
          :category-color-by-id="categoryColorById"
          :disabled="seatingLocked"
          @toggle="toggleSeat"
        />
      </template>
    </section>

    <section class="space-y-4">
      <div class="space-y-2">
        <h2 class="font-display text-lg font-semibold text-ink">Bookings</h2>
        <p class="text-sm text-ink-muted">
          Active bookings only — cancelled are hidden.
          <span class="text-ink">
            <strong>Paid</strong> = ticket settled.
            <strong>Unpaid</strong> = not paid yet (includes pending invites).
          </span>
        </p>
      </div>

      <div class="flex flex-wrap gap-2">
        <Button
          v-for="option in paymentFilterOptions"
          :key="option.value"
          size="sm"
          :variant="paymentFilter === option.value ? 'primary' : 'secondary'"
          @click="paymentFilter = option.value"
        >
          {{ option.label }}
        </Button>
      </div>

      <Input
        v-model="searchInput"
        label="Search bookings"
        placeholder="Guest, seat, or barcode"
        autocomplete="off"
      />

      <DataTable
        :columns="bookingColumns"
        :rows="bookingRows"
        :loading="bookingsLoading"
        :page="page"
        :page-size="pageSize"
        :total="bookingTotal"
        :empty-message="emptyBookingMessage"
        @update:page="page = $event"
        @update:page-size="pageSize = $event"
      />
    </section>

    <Modal :open="showCreate" title="Guest details" @close="showCreate = false">
      <form class="space-y-4" @submit.prevent="createBookings">
        <p class="text-sm text-ink-muted">
          Booking {{ selectedSeatIds.length }} seat(s):
          {{ selectedSeats.map((s) => s.code).join(', ') }}
        </p>

        <label class="flex items-center gap-2 text-sm text-ink">
          <input v-model="differentGuests" type="checkbox" class="rounded border-border" />
          Different guest per seat
        </label>

        <template v-if="!differentGuests">
          <Input v-model="form.sharedName" label="Guest name" required />
          <Input v-model="form.sharedEmail" label="Guest email" type="email" required />
        </template>

        <template v-else>
          <div
            v-for="seat in selectedSeats"
            :key="seat.id"
            class="space-y-2 rounded-lg border border-border p-3"
          >
            <p class="text-sm font-medium text-ink">Seat {{ seat.code }}</p>
            <Input v-model="guestBySeatId[seat.id!].name" label="Name" required />
            <Input v-model="guestBySeatId[seat.id!].email" label="Email" type="email" required />
          </div>
        </template>

        <Input v-model="form.notes" label="Notes (optional)" />

        <div class="flex justify-end gap-2">
          <Button variant="secondary" type="button" @click="showCreate = false">Cancel</Button>
          <Button type="submit" :loading="batchMutation.isPending.value">
            Create {{ selectedSeatIds.length }} booking(s)
          </Button>
        </div>
      </form>
    </Modal>

    <Modal :open="showPayment" title="Record payment" @close="showPayment = false">
      <form class="space-y-4" @submit.prevent="submitPayment">
        <Input v-model="paymentForm.amount" label="Amount" placeholder="150000.00" required />
        <Input v-model="paymentForm.currency" label="Currency" required />
        <Select
          v-model="paymentForm.status"
          label="Status"
          :options="[
            { label: 'Pending', value: 'pending' },
            { label: 'Success', value: 'success' },
            { label: 'Failed', value: 'failed' },
            { label: 'Refunded', value: 'refunded' },
          ]"
        />
        <Input v-model="paymentForm.method" label="Method" />
        <p class="text-xs text-ink-muted">
          Success payment marks booking paid when eligible.
        </p>
        <div class="flex justify-end gap-2">
          <Button variant="secondary" type="button" @click="showPayment = false">Cancel</Button>
          <Button type="submit" :loading="paymentMutation.isPending.value">Save payment</Button>
        </div>
      </form>
    </Modal>
  </div>
</template>
