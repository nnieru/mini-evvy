<script setup lang="ts">
import { cn } from '@/shared/lib/cn'

const props = defineProps<{
  title?: string
  description?: string
  class?: string
}>()
</script>

<template>
  <section :class="cn('rounded-2xl border border-border bg-surface-raised p-4 sm:p-6', props.class)">
    <header v-if="title || description" class="mb-4 space-y-1">
      <h2 v-if="title" class="font-display text-lg font-semibold text-ink">{{ title }}</h2>
      <p v-if="description" class="text-sm text-ink-muted">{{ description }}</p>
    </header>
    <slot />
  </section>
</template>
