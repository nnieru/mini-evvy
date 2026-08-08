<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import PageHeader from '@/shared/ui/PageHeader.vue'
import Card from '@/shared/ui/Card.vue'
import Button from '@/shared/ui/Button.vue'
import Input from '@/shared/ui/Input.vue'
import Select from '@/shared/ui/Select.vue'
import Badge from '@/shared/ui/Badge.vue'
import LoadingSpinner from '@/shared/ui/LoadingSpinner.vue'
import EmptyState from '@/shared/ui/EmptyState.vue'
import Alert from '@/shared/ui/Alert.vue'
import BarcodeCameraScanner from '../components/BarcodeCameraScanner.vue'
import { getErrorMessage } from '@/shared/lib/errors'
import { formatDateTime, titleCase } from '@/shared/lib/format'
import { useGuestsQuery } from '@/features/guests/queries/guests'
import { useSeatsQuery } from '@/features/seats/queries/seats'
import {
  useAttendanceQuery,
  useCheckInMutation,
  useUndoCheckInMutation,
} from '../queries/attendance'

const route = useRoute()
const eventId = computed(() => route.params.eventId as string)

const mode = ref<'camera' | 'manual' | 'guest'>('camera')
const error = ref('')
const success = ref('')
const scannerPaused = ref(false)

const manualForm = reactive({ barcode: '' })
const guestForm = reactive({ guest_id: '', seat_id: '', message: '' })

const { data: attendance, isLoading } = useAttendanceQuery(eventId)
const { data: guests } = useGuestsQuery(eventId)
const { data: seats } = useSeatsQuery(eventId)
const checkInMutation = useCheckInMutation(eventId)
const undoMutation = useUndoCheckInMutation(eventId)

const guestOptions = computed(
  () => guests.value?.map((g) => ({ label: g.name!, value: g.id! })) ?? [],
)
const seatOptions = computed(
  () => seats.value?.map((s) => ({ label: s.code!, value: s.id! })) ?? [],
)

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
      <LoadingSpinner v-if="isLoading" />
      <EmptyState v-else-if="!attendance?.length" title="No check-ins yet" />
      <div v-else class="space-y-3">
        <div
          v-for="item in attendance"
          :key="item.id"
          class="rounded-xl border border-border px-3 py-3 sm:flex sm:items-center sm:justify-between"
        >
          <div class="text-sm">
            <p class="font-medium text-ink">Guest {{ item.guest_id?.slice(0, 8) }}</p>
            <p class="text-ink-muted">Seat {{ item.seat_id?.slice(0, 8) }}</p>
            <p class="text-ink-muted">{{ formatDateTime(item.created_at) }}</p>
          </div>
          <div class="mt-3 flex items-center gap-2 sm:mt-0">
            <Badge :tone="item.status === 'checked_in' ? 'success' : 'muted'">
              {{ titleCase(item.status) }}
            </Badge>
            <Button
              v-if="item.status === 'checked_in'"
              size="sm"
              variant="secondary"
              @click="undo(item.id!)"
            >
              Undo
            </Button>
          </div>
        </div>
      </div>
    </Card>
  </div>
</template>
