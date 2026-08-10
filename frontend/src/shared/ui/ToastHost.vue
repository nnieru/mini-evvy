<script setup lang="ts">
import { useToast } from '@/shared/composables/useToast'
import { cn } from '@/shared/lib/cn'

const { toast } = useToast()

const toneClasses = {
  info: 'border-border bg-surface-raised text-ink',
  success: 'border-accent/20 bg-accent-soft text-accent',
  warning: 'border-warning/20 bg-warning-soft text-warning',
  error: 'border-danger/20 bg-danger-soft text-danger',
} as const
</script>

<template>
  <Teleport to="body">
    <div
      v-if="toast"
      :class="
        cn(
          'fixed bottom-6 right-6 z-[100] max-w-sm rounded-xl border px-4 py-3 text-sm shadow-lg',
          toneClasses[toast.tone],
        )
      "
      role="status"
    >
      {{ toast.message }}
    </div>
  </Teleport>
</template>
