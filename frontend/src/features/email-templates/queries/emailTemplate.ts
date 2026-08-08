import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'
import {
  defaultInvitationForm,
  getInvitationEmailTemplate,
  previewInvitationEmailTemplate,
  resetInvitationEmailTemplate,
  saveInvitationEmailTemplate,
  templatePayload,
  testSendInvitationEmailTemplate,
  uploadInvitationBanner,
  type InvitationEmailTemplate,
} from '../api/emailTemplate'
import { useAuthStore } from '@/features/auth/stores/auth'

export const emailTemplateKeys = {
  invitation: (eventId: string) => ['email-template', 'invitation', eventId] as const,
}

export function useInvitationEmailTemplateQuery(eventId: Ref<string>) {
  const auth = useAuthStore()
  return useQuery({
    queryKey: computed(() => emailTemplateKeys.invitation(eventId.value)),
    queryFn: () => getInvitationEmailTemplate(eventId.value, auth.token!),
    enabled: computed(() => Boolean(auth.token && eventId.value)),
  })
}

export function useSaveInvitationEmailTemplateMutation(eventId: Ref<string>) {
  const auth = useAuthStore()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: InvitationEmailTemplate) =>
      saveInvitationEmailTemplate(eventId.value, templatePayload(body), auth.token!),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: emailTemplateKeys.invitation(eventId.value) }),
  })
}

export function useResetInvitationEmailTemplateMutation(eventId: Ref<string>) {
  const auth = useAuthStore()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => resetInvitationEmailTemplate(eventId.value, auth.token!),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: emailTemplateKeys.invitation(eventId.value) }),
  })
}

export function usePreviewInvitationEmailMutation(eventId: Ref<string>) {
  const auth = useAuthStore()
  return useMutation({
    mutationFn: (body?: InvitationEmailTemplate) =>
      previewInvitationEmailTemplate(
        eventId.value,
        body ? templatePayload(body) : undefined,
        auth.token!,
      ),
  })
}

export function useTestSendInvitationEmailMutation(eventId: Ref<string>) {
  const auth = useAuthStore()
  return useMutation({
    mutationFn: (body?: InvitationEmailTemplate) =>
      testSendInvitationEmailTemplate(
        eventId.value,
        body ? templatePayload(body) : undefined,
        auth.token!,
      ),
  })
}

export function useUploadInvitationBannerMutation(eventId: Ref<string>) {
  const auth = useAuthStore()
  return useMutation({
    mutationFn: (file: File) => uploadInvitationBanner(eventId.value, file, auth.token!),
  })
}

export { defaultInvitationForm }
