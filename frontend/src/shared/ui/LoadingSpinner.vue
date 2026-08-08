<script setup lang="ts">
defineProps<{
  label?: string
}>()
</script>

<template>
  <div class="flex items-center justify-center gap-3 py-12 text-sm text-ink-muted">
    <span class="size-5 animate-spin rounded-full border-2 border-accent border-r-transparent" />
    <span>{{ label ?? 'Loading…' }}</span>
  </div>
</template>
