<script setup lang="ts">
defineProps<{
  title: string
  description?: string
}>()
</script>

<template>
  <div class="min-h-dvh bg-[radial-gradient(circle_at_top_left,_#d8efe8_0%,_#f6f4ef_45%,_#efe9df_100%)]">
    <div class="mx-auto flex min-h-dvh max-w-6xl items-center px-4 py-10 sm:px-6">
      <div class="grid w-full gap-10 lg:grid-cols-2 lg:items-center">
        <div class="space-y-4">
          <p class="font-display text-sm font-semibold uppercase tracking-[0.2em] text-accent">
            mini-evvy
          </p>
          <h1 class="font-display text-4xl font-semibold tracking-tight text-ink sm:text-5xl">
            {{ title }}
          </h1>
          <p v-if="description" class="max-w-md text-base text-ink-muted">{{ description }}</p>
        </div>
        <div class="rounded-3xl border border-border bg-surface-raised/90 p-6 shadow-sm backdrop-blur sm:p-8">
          <slot />
        </div>
      </div>
    </div>
  </div>
</template>
