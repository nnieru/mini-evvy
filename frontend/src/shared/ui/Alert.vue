<script setup lang="ts">
import { cn } from '@/shared/lib/cn'

type Tone = 'info' | 'success' | 'warning' | 'error'

const props = withDefaults(
  defineProps<{
    tone?: Tone
    title?: string
    class?: string
  }>(),
  { tone: 'info' },
)

const toneClasses: Record<Tone, string> = {
  info: 'border-border bg-surface-raised text-ink',
  success: 'border-accent/20 bg-accent-soft text-accent',
  warning: 'border-warning/20 bg-warning-soft text-warning',
  error: 'border-danger/20 bg-danger-soft text-danger',
}
</script>

<template>
  <div
    :class="
      cn('rounded-xl border px-4 py-3 text-sm', toneClasses[tone], props.class)
    "
    role="alert"
  >
    <p v-if="title" class="font-medium">{{ title }}</p>
    <slot />
  </div>
</template>
