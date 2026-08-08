<script setup lang="ts">
defineProps<{
  title?: string
  description?: string
}>()
</script>

<template>
  <div class="flex flex-col items-center justify-center rounded-2xl border border-dashed border-border bg-surface-raised px-6 py-12 text-center">
    <h3 v-if="title" class="font-display text-lg font-semibold text-ink">{{ title }}</h3>
    <p v-if="description" class="mt-2 max-w-md text-sm text-ink-muted">{{ description }}</p>
    <div v-if="$slots.default" class="mt-4">
      <slot />
    </div>
  </div>
</template>
