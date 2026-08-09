import { apiDelete, apiGet, apiPost, apiPut } from '@/shared/api/client'

export type InvitationEmailTemplate = {
  subject: string
  headline: string
  greeting: string
  body_html: string
  footer_text: string
  show_qr: boolean
  show_seat_code: boolean
  show_ticket_code: boolean
  qr_label: string
  ticket_code_label: string
  banner_enabled: boolean
  banner_image_url: string
  banner_alt: string
  primary_color: string
  is_default?: boolean
  updated_at?: string | null
}

export type InvitationEmailPreview = {
  subject: string
  html: string
}

export function getInvitationEmailTemplate(eventId: string, token: string) {
  return apiGet<InvitationEmailTemplate>(`/events/${eventId}/email-template/invitation`, token)
}

export function saveInvitationEmailTemplate(
  eventId: string,
  body: InvitationEmailTemplateBody,
  token: string,
) {
  return apiPut<InvitationEmailTemplate>(`/events/${eventId}/email-template/invitation`, body, token)
}

export function resetInvitationEmailTemplate(eventId: string, token: string) {
  return apiDelete<void>(`/events/${eventId}/email-template/invitation`, token)
}

export function previewInvitationEmailTemplate(
  eventId: string,
  body: InvitationEmailTemplateBody | undefined,
  token: string,
) {
  return apiPost<InvitationEmailPreview>(
    `/events/${eventId}/email-template/invitation/preview`,
    body ?? {},
    token,
  )
}

export function testSendInvitationEmailTemplate(
  eventId: string,
  body: InvitationEmailTemplateBody | undefined,
  token: string,
) {
  return apiPost<void>(
    `/events/${eventId}/email-template/invitation/test-send`,
    body ?? {},
    token,
  )
}

export type BannerUploadResponse = {
  url: string
}

const baseUrl = import.meta.env.VITE_API_BASE_URL ?? '/api'

export async function uploadInvitationBanner(eventId: string, file: File, token: string) {
  const form = new FormData()
  form.append('file', file)

  const response = await fetch(`${baseUrl}/events/${eventId}/email-template/invitation/banner`, {
    method: 'POST',
    headers: {
      Accept: 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: form,
  })

  const payload = (await response.json()) as { success?: boolean; data?: BannerUploadResponse; error?: { code?: string; message?: string } }
  if (!response.ok) {
    throw new Error(payload.error?.message ?? 'Upload failed')
  }
  return payload.data!.url
}

/** Use banner URL as-is (Supabase public https URL). */
export function bannerDisplayUrl(url: string): string {
  return url
}

export function defaultInvitationForm(): InvitationEmailTemplate {
  return {
    subject: 'Your ticket — {{event_name}}',
    headline: '',
    greeting: 'Hi {{guest_name}},',
    body_html: '',
    footer_text: 'Present this code at the door.',
    show_qr: true,
    show_seat_code: true,
    show_ticket_code: true,
    qr_label: 'Scan this code at the door:',
    ticket_code_label: 'Ticket code:',
    banner_enabled: false,
    banner_image_url: '',
    banner_alt: '',
    primary_color: '#1a1a2e',
  }
}

export type InvitationEmailTemplateBody = Omit<
  InvitationEmailTemplate,
  'is_default' | 'updated_at' | 'banner_image_url'
> & {
  banner_image_url: string | null
}

export function templatePayload(form: InvitationEmailTemplate): InvitationEmailTemplateBody {
  return {
    subject: form.subject,
    headline: form.headline,
    greeting: form.greeting,
    body_html: form.body_html,
    footer_text: form.footer_text,
    show_qr: form.show_qr,
    show_seat_code: form.show_seat_code,
    show_ticket_code: form.show_ticket_code,
    qr_label: form.qr_label,
    ticket_code_label: form.ticket_code_label,
    banner_enabled: form.banner_enabled,
    banner_image_url: form.banner_image_url.trim() ? form.banner_image_url.trim() : null,
    banner_alt: form.banner_alt,
    primary_color: form.primary_color,
  }
}
