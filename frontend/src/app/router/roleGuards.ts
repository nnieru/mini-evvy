import { getEvent } from '@/features/events/api/events'
import { getMyRole } from '@/features/organizations/api/organizations'
import { isStaffRole, type OrgRole } from '@/features/organizations/lib/roles'

export async function isStaffForEvent(eventId: string, token: string): Promise<boolean> {
  const event = await getEvent(eventId, token)
  if (!event.organization_id) return false
  const myRole = await getMyRole(event.organization_id, token)
  return isStaffRole(myRole.role as OrgRole)
}
