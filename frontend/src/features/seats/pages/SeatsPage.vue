<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import PageHeader from '@/shared/ui/PageHeader.vue'
import Button from '@/shared/ui/Button.vue'
import Input from '@/shared/ui/Input.vue'
import Select from '@/shared/ui/Select.vue'
import Badge from '@/shared/ui/Badge.vue'
import Modal from '@/shared/ui/Modal.vue'
import LoadingSpinner from '@/shared/ui/LoadingSpinner.vue'
import EmptyState from '@/shared/ui/EmptyState.vue'
import Alert from '@/shared/ui/Alert.vue'
import { getErrorMessage } from '@/shared/lib/errors'
import { titleCase } from '@/shared/lib/format'
import { useCategoriesQuery } from '@/features/categories/queries/categories'
import { useCreateSeatsMutation, useSeatsQuery } from '../queries/seats'

const route = useRoute()
const eventId = computed(() => route.params.eventId as string)

const showCreate = ref(false)
const error = ref('')
const form = reactive({
  code: '',
  category_id: '',
  section: '',
  row: '',
  col: '',
  status: 'available',
})

const { data: seats, isLoading } = useSeatsQuery(eventId)
const { data: categories } = useCategoriesQuery(eventId)
const createMutation = useCreateSeatsMutation(eventId)

const categoryOptions = computed(
  () => categories.value?.map((c) => ({ label: c.name ?? 'Category', value: c.id! })) ?? [],
)

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

    <LoadingSpinner v-if="isLoading" />
    <EmptyState v-else-if="!seats?.length" title="No seats yet" description="Create categories first, then add seats." />

    <div v-else class="space-y-3">
      <div
        v-for="seat in seats"
        :key="seat.id"
        class="rounded-xl border border-border bg-surface-raised p-4 sm:flex sm:items-center sm:justify-between"
      >
        <div>
          <p class="font-medium text-ink">{{ seat.code }}</p>
          <p class="text-sm text-ink-muted">
            {{ seat.section || 'No section' }}
            <span v-if="seat.row != null"> · Row {{ seat.row }}</span>
            <span v-if="seat.col != null"> · Col {{ seat.col }}</span>
          </p>
        </div>
        <Badge class="mt-2 sm:mt-0">{{ titleCase(seat.status) }}</Badge>
      </div>
    </div>

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
