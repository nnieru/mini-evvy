<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { RouterLink } from 'vue-router'
import PageHeader from '@/shared/ui/PageHeader.vue'
import Card from '@/shared/ui/Card.vue'
import Button from '@/shared/ui/Button.vue'
import Input from '@/shared/ui/Input.vue'
import Modal from '@/shared/ui/Modal.vue'
import LoadingSpinner from '@/shared/ui/LoadingSpinner.vue'
import EmptyState from '@/shared/ui/EmptyState.vue'
import Alert from '@/shared/ui/Alert.vue'
import Badge from '@/shared/ui/Badge.vue'
import { getErrorMessage } from '@/shared/lib/errors'
import { formatDate, formatDateTime, titleCase } from '@/shared/lib/format'
import {
  useCreateOrganizationMutation,
  useOrganizationsQuery,
} from '../queries/organizations'
import { useMyEventsQuery } from '@/features/events/queries/events'
import {
  canCreateOrganization,
  isMemberOnlyHome,
} from '../lib/roles'

const showCreate = ref(false)
const error = ref('')
const form = reactive({ name: '' })

const { data: orgs, isLoading: orgsLoading } = useOrganizationsQuery()
const createMutation = useCreateOrganizationMutation()

const isMemberOnly = computed(() => isMemberOnlyHome(orgs.value))
const canCreateOrg = computed(() => canCreateOrganization(orgs.value))

const { data: myEvents, isLoading: eventsLoading } = useMyEventsQuery(isMemberOnly)
const isLoading = computed(() => orgsLoading.value || (isMemberOnly.value && eventsLoading.value))

async function createOrg() {
  error.value = ''
  try {
    await createMutation.mutateAsync({ name: form.name })
    form.name = ''
    showCreate.value = false
  } catch (err) {
    error.value = getErrorMessage(err)
  }
}
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <PageHeader
        :title="isMemberOnly ? 'My events' : 'Organizations'"
        :description="
          isMemberOnly
            ? 'Events you can access across your teams.'
            : 'Manage teams and events across your organizations.'
        "
      />
      <Button v-if="canCreateOrg" @click="showCreate = true">New organization</Button>
    </div>

    <LoadingSpinner v-if="isLoading" />

    <template v-else-if="isMemberOnly">
      <EmptyState
        v-if="!myEvents?.length"
        title="No events yet"
        description="You have not been assigned to any events yet."
      />
      <div v-else class="grid gap-4 sm:grid-cols-2">
        <RouterLink
          v-for="event in myEvents"
          :key="event.id"
          :to="`/events/${event.id}/bookings`"
          class="block"
        >
          <Card class="transition hover:border-accent/40 hover:shadow-sm">
            <div class="flex items-start justify-between gap-3">
              <div>
                <h3 class="font-display text-lg font-semibold text-ink">{{ event.name }}</h3>
                <p class="mt-1 text-sm text-ink-muted">{{ event.organization_name }}</p>
              </div>
              <Badge>{{ titleCase(event.status) }}</Badge>
            </div>
            <p class="mt-3 text-sm text-ink-muted">
              {{ formatDate(event.start_date) }}
              <span v-if="event.end_date"> – {{ formatDate(event.end_date) }}</span>
            </p>
          </Card>
        </RouterLink>
      </div>
    </template>

    <template v-else>
      <EmptyState
        v-if="!orgs?.length"
        title="No organizations yet"
        description="Create your first organization to start planning events."
      >
        <Button @click="showCreate = true">Create organization</Button>
      </EmptyState>

      <div v-else class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        <RouterLink
          v-for="org in orgs"
          :key="org.id"
          :to="`/orgs/${org.id}`"
          class="block"
        >
          <Card class="transition hover:border-accent/40 hover:shadow-sm">
            <h3 class="font-display text-lg font-semibold text-ink">{{ org.name }}</h3>
            <p class="mt-2 text-sm text-ink-muted">Created {{ formatDateTime(org.created_at) }}</p>
          </Card>
        </RouterLink>
      </div>
    </template>

    <Modal :open="showCreate" title="Create organization" @close="showCreate = false">
      <form class="space-y-4" @submit.prevent="createOrg">
        <Alert v-if="error" tone="error" :title="error" />
        <Input v-model="form.name" label="Organization name" required />
        <div class="flex justify-end gap-2">
          <Button variant="secondary" type="button" @click="showCreate = false">Cancel</Button>
          <Button type="submit" :loading="createMutation.isPending.value">Create</Button>
        </div>
      </form>
    </Modal>
  </div>
</template>
