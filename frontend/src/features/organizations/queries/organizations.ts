import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'
import {
  addMember,
  createOrganization,
  getMyRole,
  getOrganization,
  listMembers,
  listOrganizations,
} from '../api/organizations'
import { useAuthStore } from '@/features/auth/stores/auth'
import type { AddMemberRequest, CreateOrgRequest } from '@/shared/api/types'
import type { OrgRole } from '../lib/roles'

export const orgKeys = {
  all: ['orgs'] as const,
  detail: (orgId: string) => ['orgs', orgId] as const,
  members: (orgId: string) => ['orgs', orgId, 'members'] as const,
  myRole: (orgId: string) => ['orgs', orgId, 'my-role'] as const,
}

export function useOrganizationsQuery() {
  const auth = useAuthStore()
  return useQuery({
    queryKey: orgKeys.all,
    queryFn: () => listOrganizations(auth.token!),
    enabled: computed(() => Boolean(auth.token)),
  })
}

export function useOrganizationQuery(orgId: Ref<string>) {
  const auth = useAuthStore()
  return useQuery({
    queryKey: computed(() => orgKeys.detail(orgId.value)),
    queryFn: () => getOrganization(orgId.value, auth.token!),
    enabled: computed(() => Boolean(auth.token && orgId.value)),
  })
}

export function useMembersQuery(orgId: Ref<string>) {
  const auth = useAuthStore()
  return useQuery({
    queryKey: computed(() => orgKeys.members(orgId.value)),
    queryFn: () => listMembers(orgId.value, auth.token!),
    enabled: computed(() => Boolean(auth.token && orgId.value)),
  })
}

export function useMyRoleQuery(orgId: Ref<string>) {
  const auth = useAuthStore()
  return useQuery({
    queryKey: computed(() => orgKeys.myRole(orgId.value)),
    queryFn: async () => {
      const result = await getMyRole(orgId.value, auth.token!)
      return { role: result.role as OrgRole }
    },
    enabled: computed(() => Boolean(auth.token && orgId.value)),
  })
}

export function useCreateOrganizationMutation() {
  const auth = useAuthStore()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: CreateOrgRequest) => createOrganization(body, auth.token!),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: orgKeys.all }),
  })
}

export function useAddMemberMutation(orgId: Ref<string>) {
  const auth = useAuthStore()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: AddMemberRequest) => addMember(orgId.value, body, auth.token!),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: orgKeys.members(orgId.value) }),
  })
}
