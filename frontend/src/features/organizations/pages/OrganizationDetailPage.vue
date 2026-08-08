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
import Alert from '@/shared/ui/Alert.vue'
import { getErrorMessage } from '@/shared/lib/errors'
import { formatDateTime } from '@/shared/lib/format'
import {
  useAddMemberMutation,
  useMembersQuery,
  useMyRoleQuery,
  useOrganizationQuery,
} from '../queries/organizations'
import { isStaffRole } from '../lib/roles'

const route = useRoute()
const orgId = computed(() => route.params.orgId as string)

const showAddMember = ref(false)
const error = ref('')
const form = reactive({ email: '', role: 'member' })

const { data: org, isLoading: orgLoading } = useOrganizationQuery(orgId)
const { data: members, isLoading: membersLoading } = useMembersQuery(orgId)
const { data: myRole } = useMyRoleQuery(orgId)
const isStaff = computed(() => isStaffRole(myRole.value?.role))
const addMemberMutation = useAddMemberMutation(orgId)

async function addMember() {
  error.value = ''
  try {
    await addMemberMutation.mutateAsync({
      email: form.email,
      role: form.role as 'admin' | 'member',
    })
    form.email = ''
    showAddMember.value = false
  } catch (err) {
    error.value = getErrorMessage(err)
  }
}
</script>

<template>
  <div class="space-y-6">
    <PageHeader
      :title="org?.name ?? 'Organization'"
      description="View organization details and manage members."
    />

    <LoadingSpinner v-if="orgLoading" />

    <div v-else class="grid gap-6 lg:grid-cols-[1.2fr_1fr]">
      <Card title="Details">
        <dl class="grid gap-3 text-sm sm:grid-cols-2">
          <div>
            <dt class="text-ink-muted">Status</dt>
            <dd class="font-medium capitalize">{{ org?.status }}</dd>
          </div>
          <div>
            <dt class="text-ink-muted">Created</dt>
            <dd class="font-medium">{{ formatDateTime(org?.created_at) }}</dd>
          </div>
        </dl>
      </Card>

      <Card title="Members" description="Invite admins and members to collaborate.">
        <div v-if="isStaff" class="mb-4 flex justify-end">
          <Button size="sm" @click="showAddMember = true">Add member</Button>
        </div>
        <LoadingSpinner v-if="membersLoading" />
        <div v-else class="space-y-3">
          <div
            v-for="member in members"
            :key="member.user_role_id"
            class="rounded-xl border border-border px-3 py-3"
          >
            <p class="font-medium text-ink">{{ member.name }}</p>
            <p class="text-sm text-ink-muted">{{ member.email }}</p>
            <p class="mt-1 text-xs uppercase tracking-wide text-accent">{{ member.role_name }}</p>
          </div>
        </div>
      </Card>
    </div>

    <Modal :open="showAddMember" title="Add member" @close="showAddMember = false">
      <form class="space-y-4" @submit.prevent="addMember">
        <Alert v-if="error" tone="error" :title="error" />
        <Input v-model="form.email" label="Email" type="email" required />
        <Select
          v-model="form.role"
          label="Role"
          :options="[
            { label: 'Admin', value: 'admin' },
            { label: 'Member', value: 'member' },
          ]"
        />
        <div class="flex justify-end gap-2">
          <Button variant="secondary" type="button" @click="showAddMember = false">Cancel</Button>
          <Button type="submit" :loading="addMemberMutation.isPending.value">Add</Button>
        </div>
      </form>
    </Modal>
  </div>
</template>
