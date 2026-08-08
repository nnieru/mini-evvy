<script setup lang="ts">
import { cn } from '@/shared/lib/cn'

const props = defineProps<{
  modelValue?: string | number
  label?: string
  type?: string
  placeholder?: string
  required?: boolean
  disabled?: boolean
  error?: string
  class?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()
</script>

<template>
  <label class="block space-y-1.5">
    <span v-if="label" class="text-sm font-medium text-ink">{{ label }}</span>
    <input
      :value="modelValue"
      :type="type ?? 'text'"
      :placeholder="placeholder"
      :required="required"
      :disabled="disabled"
      :class="
        cn(
          'w-full rounded-lg border border-border bg-surface-raised px-3 py-2 text-sm outline-none transition focus:border-accent focus:ring-2 focus:ring-accent/20 disabled:opacity-60',
          error && 'border-danger focus:border-danger focus:ring-danger/20',
          props.class,
        )
      "
      @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
    />
    <span v-if="error" class="text-xs text-danger">{{ error }}</span>
  </label>
</template>
