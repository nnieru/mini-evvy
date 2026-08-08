<script setup lang="ts">
import { cn } from '@/shared/lib/cn'

type Option = { label: string; value: string }

const props = defineProps<{
  modelValue?: string
  label?: string
  options: Option[]
  placeholder?: string
  required?: boolean
  disabled?: boolean
  class?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()
</script>

<template>
  <label class="block space-y-1.5">
    <span v-if="label" class="text-sm font-medium text-ink">{{ label }}</span>
    <select
      :value="modelValue"
      :required="required"
      :disabled="disabled"
      :class="
        cn(
          'w-full rounded-lg border border-border bg-surface-raised px-3 py-2 text-sm outline-none transition focus:border-accent focus:ring-2 focus:ring-accent/20 disabled:opacity-60',
          props.class,
        )
      "
      @change="emit('update:modelValue', ($event.target as HTMLSelectElement).value)"
    >
      <option v-if="placeholder" disabled value="">{{ placeholder }}</option>
      <option v-for="option in options" :key="option.value" :value="option.value">
        {{ option.label }}
      </option>
    </select>
  </label>
</template>
