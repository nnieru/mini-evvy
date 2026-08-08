<script setup lang="ts">
import { cn } from '@/shared/lib/cn'

type Tone = 'default' | 'success' | 'warning' | 'danger' | 'muted'

const props = withDefaults(
  defineProps<{
    tone?: Tone
    class?: string
  }>(),
  { tone: 'default' },
)

const toneClasses: Record<Tone, string> = {
  default: 'bg-accent-soft text-accent',
  success: 'bg-accent-soft text-accent',
  warning: 'bg-warning-soft text-warning',
  danger: 'bg-danger-soft text-danger',
  muted: 'bg-surface text-ink-muted border border-border',
}
</script>

<template>
  <span
    :class="
      cn(
        'inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium capitalize',
        toneClasses[tone],
        props.class,
      )
    "
  >
    <slot />
  </span>
</template>
