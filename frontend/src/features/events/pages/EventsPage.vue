<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import PageHeader from '@/shared/ui/PageHeader.vue'
import Card from '@/shared/ui/Card.vue'
import Button from '@/shared/ui/Button.vue'
import Input from '@/shared/ui/Input.vue'
import Modal from '@/shared/ui/Modal.vue'
import Badge from '@/shared/ui/Badge.vue'
import LoadingSpinner from '@/shared/ui/LoadingSpinner.vue'
import EmptyState from '@/shared/ui/EmptyState.vue'
import Alert from '@/shared/ui/Alert.vue'
import { getErrorMessage } from '@/shared/lib/errors'
import { formatDate, titleCase } from '@/shared/lib/format'
import { useCreateEventMutation, useEventsQuery } from '../queries/events'
import { useMyRoleQuery, useOrganizationQuery } from '@/features/organizations/queries/organizations'
import { isStaffRole } from '@/features/organizations/lib/roles'

const route = useRoute()
const orgId = computed(() => route.params.orgId as string)

const showCreate = ref(false)
const error = ref('')
const form = reactive({
  name: '',
  description: '',
  start_date: '',
  end_date: '',
})

const { data: org } = useOrganizationQuery(orgId)
const { data: events, isLoading } = useEventsQuery(orgId)
const { data: myRole } = useMyRoleQuery(orgId)
const isStaff = computed(() => isStaffRole(myRole.value?.role))
const createMutation = useCreateEventMutation(orgId)

function eventPath(id: string) {
  return isStaff.value ? `/events/${id}` : `/events/${id}/bookings`
}

async function createEvent() {
  error.value = ''
  try {
    await createMutation.mutateAsync({
      name: form.name,
      description: form.description || null,
      start_date: form.start_date,
      end_date: form.end_date || null,
    })
    showCreate.value = false
    form.name = ''
    form.description = ''
    form.start_date = ''
    form.end_date = ''
  } catch (err) {
    error.value = getErrorMessage(err)
  }
}
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <PageHeader
        :title="`${org?.name ?? 'Organization'} events`"
        description="Create and manage events for this organization."
      />
      <Button v-if="isStaff" @click="showCreate = true">New event</Button>
    </div>

    <LoadingSpinner v-if="isLoading" />
    <EmptyState
      v-else-if="!events?.length"
      title="No events yet"
      description="Create your first event to set up seating and guests."
    >
      <Button v-if="isStaff" @click="showCreate = true">Create event</Button>
    </EmptyState>

    <div v-else class="grid gap-4 sm:grid-cols-2">
      <RouterLink
        v-for="event in events"
        :key="event.id"
        :to="eventPath(event.id!)"
        class="block"
      >
        <Card class="transition hover:border-accent/40 hover:shadow-sm">
          <div class="flex items-start justify-between gap-3">
            <div>
              <h3 class="font-display text-lg font-semibold text-ink">{{ event.name }}</h3>
              <p class="mt-1 text-sm text-ink-muted">
                {{ formatDate(event.start_date) }}
                <span v-if="event.end_date"> – {{ formatDate(event.end_date) }}</span>
              </p>
            </div>
            <Badge>{{ titleCase(event.status) }}</Badge>
          </div>
        </Card>
      </RouterLink>
    </div>

    <Modal :open="showCreate" title="Create event" @close="showCreate = false">
      <form class="space-y-4" @submit.prevent="createEvent">
        <Alert v-if="error" tone="error" :title="error" />
        <Input v-model="form.name" label="Event name" required />
        <Input v-model="form.description" label="Description" />
        <Input v-model="form.start_date" label="Start date" type="date" required />
        <Input v-model="form.end_date" label="End date" type="date" />
        <div class="flex justify-end gap-2">
          <Button variant="secondary" type="button" @click="showCreate = false">Cancel</Button>
          <Button type="submit" :loading="createMutation.isPending.value">Create</Button>
        </div>
      </form>
    </Modal>
  </div>
</template>
