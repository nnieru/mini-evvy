<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import Button from './Button.vue'

const props = defineProps<{
  open: boolean
  title: string
  description?: string
}>()

const emit = defineEmits<{
  close: []
}>()

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && props.open) emit('close')
}

onMounted(() => window.addEventListener('keydown', onKeydown))
onUnmounted(() => window.removeEventListener('keydown', onKeydown))
</script>

<template>
  <Teleport to="body">
    <div
      v-if="open"
      class="fixed inset-0 z-50 flex items-end justify-center bg-ink/40 p-4 sm:items-center"
      @click.self="emit('close')"
    >
      <div class="w-full max-w-lg rounded-2xl border border-border bg-surface-raised p-5 shadow-xl">
        <div class="space-y-1">
          <h2 class="font-display text-lg font-semibold text-ink">{{ title }}</h2>
          <p v-if="description" class="text-sm text-ink-muted">{{ description }}</p>
        </div>
        <div class="mt-4">
          <slot />
        </div>
        <div v-if="$slots.footer" class="mt-5 flex justify-end gap-2">
          <slot name="footer" />
        </div>
        <div v-else class="mt-5 flex justify-end">
          <Button variant="secondary" @click="emit('close')">Close</Button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
