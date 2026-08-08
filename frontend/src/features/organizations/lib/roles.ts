export type OrgRole = 'owner' | 'admin' | 'member'

export function isStaffRole(role: OrgRole | string | null | undefined): boolean {
  return role === 'owner' || role === 'admin'
}

export function hasOrgRole(role: OrgRole | string | null | undefined): boolean {
  return role === 'owner' || role === 'admin' || role === 'member'
}

export function canCreateOrganization(
  orgs: Array<{ my_role?: string | null }> | null | undefined,
): boolean {
  if (!orgs?.length) return true
  return orgs.some((org) => isStaffRole(org.my_role))
}

export function isMemberOnlyHome(
  orgs: Array<{ my_role?: string | null }> | null | undefined,
): boolean {
  return Boolean(orgs?.length) && !canCreateOrganization(orgs)
}
