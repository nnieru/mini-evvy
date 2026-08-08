<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import PageHeader from '@/shared/ui/PageHeader.vue'
import Card from '@/shared/ui/Card.vue'
import Button from '@/shared/ui/Button.vue'
import Input from '@/shared/ui/Input.vue'
import Alert from '@/shared/ui/Alert.vue'
import LoadingSpinner from '@/shared/ui/LoadingSpinner.vue'
import Badge from '@/shared/ui/Badge.vue'
import { getErrorMessage } from '@/shared/lib/errors'
import {
  defaultInvitationForm,
  bannerDisplayUrl,
  type InvitationEmailTemplate,
} from '../api/emailTemplate'
import {
  useInvitationEmailTemplateQuery,
  usePreviewInvitationEmailMutation,
  useResetInvitationEmailTemplateMutation,
  useSaveInvitationEmailTemplateMutation,
  useTestSendInvitationEmailMutation,
  useUploadInvitationBannerMutation,
} from '../queries/emailTemplate'

const route = useRoute()
const eventId = computed(() => route.params.eventId as string)

const form = reactive<InvitationEmailTemplate>(defaultInvitationForm())
const previewHtml = ref('')
const previewSubject = ref('')
const message = ref('')
const error = ref('')
const dirty = ref(false)
let previewTimer: ReturnType<typeof setTimeout> | undefined

const { data: template, isLoading, refetch } = useInvitationEmailTemplateQuery(eventId)
const saveMutation = useSaveInvitationEmailTemplateMutation(eventId)
const resetMutation = useResetInvitationEmailTemplateMutation(eventId)
const previewMutation = usePreviewInvitationEmailMutation(eventId)
const testSendMutation = useTestSendInvitationEmailMutation(eventId)
const uploadBannerMutation = useUploadInvitationBannerMutation(eventId)
const bannerFileInput = ref<HTMLInputElement | null>(null)

const bannerPreviewSrc = computed(() => bannerDisplayUrl(form.banner_image_url))

const mergeTags = ['{{guest_name}}', '{{event_name}}', '{{seat_code}}', '{{ticket_code}}']

function applyTemplate(data: InvitationEmailTemplate) {
  Object.assign(form, {
    subject: data.subject,
    headline: data.headline,
    greeting: data.greeting,
    body_html: data.body_html,
    footer_text: data.footer_text,
    show_qr: data.show_qr,
    show_seat_code: data.show_seat_code,
    show_ticket_code: data.show_ticket_code,
    banner_enabled: data.banner_enabled,
    banner_image_url: data.banner_image_url ?? '',
    banner_alt: data.banner_alt,
    primary_color: data.primary_color,
  })
  dirty.value = false
}

watch(
  template,
  (value) => {
    if (!value) return
    applyTemplate(value)
    void refreshPreview()
  },
  { immediate: true },
)

watch(
  form,
  () => {
    dirty.value = true
    schedulePreview()
  },
  { deep: true },
)

function schedulePreview() {
  if (previewTimer) clearTimeout(previewTimer)
  previewTimer = setTimeout(() => {
    void refreshPreview()
  }, 400)
}

async function refreshPreview() {
  error.value = ''
  try {
    const result = await previewMutation.mutateAsync({ ...form })
    previewHtml.value = result.html
    previewSubject.value = result.subject
  } catch (err) {
    error.value = getErrorMessage(err)
  }
}

async function save() {
  message.value = ''
  error.value = ''
  try {
    const saved = await saveMutation.mutateAsync({ ...form })
    applyTemplate(saved)
    message.value = 'Template saved.'
    await refreshPreview()
  } catch (err) {
    error.value = getErrorMessage(err)
  }
}

async function resetToDefault() {
  message.value = ''
  error.value = ''
  try {
    await resetMutation.mutateAsync()
    const { data } = await refetch()
    if (data) applyTemplate(data)
    message.value = 'Reset to system default.'
    await refreshPreview()
  } catch (err) {
    error.value = getErrorMessage(err)
  }
}

async function sendTest() {
  message.value = ''
  error.value = ''
  try {
    await testSendMutation.mutateAsync({ ...form })
    message.value = 'Test email sent to your account address.'
  } catch (err) {
    error.value = getErrorMessage(err)
  }
}

function insertTag(tag: string, field: 'subject' | 'greeting' | 'body_html' | 'footer_text') {
  form[field] = `${form[field]}${form[field] ? ' ' : ''}${tag}`
}

function pickBannerFile() {
  bannerFileInput.value?.click()
}

async function onBannerFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return

  message.value = ''
  error.value = ''
  try {
    const url = await uploadBannerMutation.mutateAsync(file)
    form.banner_image_url = url
    form.banner_enabled = true
    message.value = 'Banner uploaded.'
    dirty.value = true
    await refreshPreview()
  } catch (err) {
    error.value = getErrorMessage(err)
  }
}
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <PageHeader
        title="Invite email"
        description="Customize the ticket invitation email sent after payment or resend."
      />
      <div class="flex flex-wrap gap-2">
        <Badge v-if="template?.is_default" tone="muted">System default</Badge>
        <Badge v-else tone="default">Custom</Badge>
      </div>
    </div>

    <Alert v-if="message" variant="success">{{ message }}</Alert>
    <Alert v-if="error" variant="danger">{{ error }}</Alert>

    <LoadingSpinner v-if="isLoading" />

    <div v-else class="grid gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
      <Card class="space-y-5">
        <Input v-model="form.subject" label="Subject" placeholder="Your ticket — {{event_name}}" />
        <div class="flex flex-wrap gap-2">
          <button
            v-for="tag in mergeTags"
            :key="`subject-${tag}`"
            type="button"
            class="rounded-md border border-border px-2 py-1 text-xs text-ink-muted hover:bg-accent-soft"
            @click="insertTag(tag, 'subject')"
          >
            {{ tag }}
          </button>
        </div>

        <div class="space-y-3 rounded-xl border border-border p-4">
          <label class="flex items-center gap-2 text-sm font-medium text-ink">
            <input v-model="form.banner_enabled" type="checkbox" class="rounded border-border" />
            Show banner image
          </label>
          <div class="flex flex-wrap items-center gap-2">
            <input
              ref="bannerFileInput"
              type="file"
              accept="image/jpeg,image/png,image/webp,image/gif"
              class="hidden"
              @change="onBannerFileChange"
            />
            <Button
              type="button"
              variant="secondary"
              size="sm"
              :disabled="uploadBannerMutation.isPending.value"
              @click="pickBannerFile"
            >
              {{ uploadBannerMutation.isPending.value ? 'Uploading…' : 'Upload image' }}
            </Button>
            <span class="text-xs text-ink-muted">JPEG, PNG, WebP, or GIF · max 2 MiB</span>
          </div>
          <img
            v-if="bannerPreviewSrc && form.banner_enabled"
            :src="bannerPreviewSrc"
            alt="Banner preview"
            class="max-h-40 w-full rounded-lg border border-border object-cover"
          />
          <Input
            v-model="form.banner_image_url"
            label="Or external banner URL (https)"
            placeholder="https://example.com/banner.jpg"
            :disabled="!form.banner_enabled"
          />
          <Input
            v-model="form.banner_alt"
            label="Banner alt text"
            placeholder="{{event_name}}"
            :disabled="!form.banner_enabled"
          />
        </div>

        <Input v-model="form.headline" label="Headline" placeholder="Optional title under banner" />
        <Input v-model="form.greeting" label="Greeting" placeholder="Hi {{guest_name}}," />
        <div class="space-y-1.5">
          <label class="text-sm font-medium text-ink">Body (HTML)</label>
          <textarea
            v-model="form.body_html"
            rows="4"
            class="w-full rounded-lg border border-border bg-surface-raised px-3 py-2 text-sm outline-none transition focus:border-accent focus:ring-2 focus:ring-accent/20"
            placeholder="Optional extra copy. Allowed: &lt;p&gt;, &lt;strong&gt;, &lt;br&gt;, &lt;a&gt;"
          />
          <div class="flex flex-wrap gap-2">
            <button
              v-for="tag in mergeTags"
              :key="`body-${tag}`"
              type="button"
              class="rounded-md border border-border px-2 py-1 text-xs text-ink-muted hover:bg-accent-soft"
              @click="insertTag(tag, 'body_html')"
            >
              {{ tag }}
            </button>
          </div>
        </div>

        <div class="grid gap-3 sm:grid-cols-3">
          <label class="flex items-center gap-2 text-sm text-ink">
            <input v-model="form.show_qr" type="checkbox" class="rounded border-border" />
            Show QR
          </label>
          <label class="flex items-center gap-2 text-sm text-ink">
            <input v-model="form.show_seat_code" type="checkbox" class="rounded border-border" />
            Show seat
          </label>
          <label class="flex items-center gap-2 text-sm text-ink">
            <input v-model="form.show_ticket_code" type="checkbox" class="rounded border-border" />
            Show ticket code
          </label>
        </div>

        <Input v-model="form.footer_text" label="Footer" />
        <Input v-model="form.primary_color" label="Accent color" type="color" class="h-10 w-24" />

        <div class="flex flex-wrap gap-2 pt-2">
          <Button :disabled="saveMutation.isPending.value || !dirty" @click="save">
            {{ saveMutation.isPending.value ? 'Saving…' : 'Save' }}
          </Button>
          <Button variant="secondary" :disabled="previewMutation.isPending.value" @click="refreshPreview">
            Refresh preview
          </Button>
          <Button variant="secondary" :disabled="resetMutation.isPending.value" @click="resetToDefault">
            Reset default
          </Button>
          <Button variant="secondary" :disabled="testSendMutation.isPending.value" @click="sendTest">
            Send test to me
          </Button>
        </div>
      </Card>

      <Card class="space-y-4">
        <div>
          <h2 class="font-display text-lg font-semibold text-ink">Preview</h2>
          <p class="text-sm text-ink-muted">Sample guest data with debounced live updates.</p>
        </div>
        <p v-if="previewSubject" class="text-sm text-ink-muted">
          Subject: <span class="font-medium text-ink">{{ previewSubject }}</span>
        </p>
        <div class="overflow-hidden rounded-xl border border-border bg-[#f4f4f5]">
          <iframe
            v-if="previewHtml"
            title="Invitation email preview"
            class="h-[640px] w-full bg-white"
            sandbox=""
            :srcdoc="previewHtml"
          />
          <div v-else class="flex h-64 items-center justify-center text-sm text-ink-muted">
            Preview will appear here
          </div>
        </div>
      </Card>
    </div>
  </div>
</template>
