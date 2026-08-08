<script setup lang="ts">
import { cn } from '@/shared/lib/cn'

const props = defineProps<{
  title: string
  description?: string
  class?: string
}>()
</script>

<template>
  <div :class="cn('space-y-1', props.class)">
    <h1 class="font-display text-2xl font-semibold tracking-tight text-ink sm:text-3xl">
      {{ title }}
    </h1>
    <p v-if="description" class="max-w-2xl text-sm text-ink-muted sm:text-base">
      {{ description }}
    </p>
  </div>
</template>
