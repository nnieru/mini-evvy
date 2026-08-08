import { computed, type Ref } from 'vue'
import { useEventQuery } from '@/features/events/queries/events'
import { useMyRoleQuery } from '../queries/organizations'
import { isStaffRole } from '../lib/roles'

export function useEventOrgRole(eventId: Ref<string>) {
  const { data: event } = useEventQuery(eventId)
  const orgId = computed(() => event.value?.organization_id ?? '')
  const { data: myRole, isLoading } = useMyRoleQuery(orgId)

  const role = computed(() => myRole.value?.role ?? null)
  const isStaff = computed(() => isStaffRole(role.value))

  return { event, orgId, role, isStaff, isLoading }
}
