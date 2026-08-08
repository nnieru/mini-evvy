import { apiGet, apiPost } from '@/shared/api/client'
import type {
  AddMemberRequest,
  CreateOrgRequest,
  Member,
  Organization,
} from '@/shared/api/types'

export function listOrganizations(token: string) {
  return apiGet<Organization[]>('/orgs', token)
}

export function getOrganization(orgId: string, token: string) {
  return apiGet<Organization>(`/orgs/${orgId}`, token)
}

export function createOrganization(body: CreateOrgRequest, token: string) {
  return apiPost<Organization>('/orgs', body, token)
}

export function listMembers(orgId: string, token: string) {
  return apiGet<Member[]>(`/orgs/${orgId}/members`, token)
}

export function addMember(orgId: string, body: AddMemberRequest, token: string) {
  return apiPost<Member>(`/orgs/${orgId}/members`, body, token)
}

export function getMyRole(orgId: string, token: string) {
  return apiGet<{ role: string }>(`/orgs/${orgId}/my-role`, token)
}
