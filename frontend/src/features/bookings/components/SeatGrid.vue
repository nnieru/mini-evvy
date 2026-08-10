<script setup lang="ts">
import { computed } from 'vue'
import type { Seat } from '@/shared/api/types'
import { withAlpha } from '../lib/categoryColors'

const props = defineProps<{
  seats: Seat[]
  selectedIds: string[]
  categoryColorById?: Record<string, string>
  disabled?: boolean
}>()

const emit = defineEmits<{
  toggle: [seatId: string]
}>()

const gridSeats = computed(() =>
  props.seats.filter((seat) => seat.row != null && seat.col != null),
)

const unplacedSeats = computed(() =>
  props.seats.filter((seat) => seat.row == null || seat.col == null),
)

const gridBounds = computed(() => {
  if (!gridSeats.value.length) return null
  let minRow = Infinity
  let maxRow = -Infinity
  let minCol = Infinity
  let maxCol = -Infinity
  for (const seat of gridSeats.value) {
    minRow = Math.min(minRow, seat.row!)
    maxRow = Math.max(maxRow, seat.row!)
    minCol = Math.min(minCol, seat.col!)
    maxCol = Math.max(maxCol, seat.col!)
  }
  return { minRow, maxRow, minCol, maxCol }
})

const seatAt = computed(() => {
  const map = new Map<string, Seat>()
  for (const seat of gridSeats.value) {
    map.set(`${seat.row}-${seat.col}`, seat)
  }
  return map
})

function categoryColor(seat: Seat): string {
  if (!seat.category_id) return '#94a3b8'
  return props.categoryColorById?.[seat.category_id] ?? '#94a3b8'
}

function isSelected(seatId: string) {
  return props.selectedIds.includes(seatId)
}

function isSelectable(seat: Seat) {
  if (props.disabled) return false
  return seat.status === 'available'
}

function seatClasses(seat: Seat) {
  const selected = isSelected(seat.id!)
  const selectable = isSelectable(seat)
  if (selected) return 'ring-2 ring-accent/40 cursor-pointer'
  if (!selectable) return 'text-ink-muted opacity-60 cursor-not-allowed'
  switch (seat.status) {
    case 'available':
      return 'text-ink cursor-pointer hover:brightness-95'
    case 'reserved':
      return 'border-warning/40 bg-warning-soft text-warning'
    case 'occupied':
      return 'border-accent/40 bg-accent-soft/60 text-accent'
    case 'held':
      return 'border-dashed border-ink-muted/50 bg-surface text-ink-muted'
    case 'blocked':
      return 'border-border bg-surface text-ink-muted'
    default:
      return 'border-border bg-surface-raised text-ink'
  }
}

function seatStyle(seat: Seat): Record<string, string> {
  const color = categoryColor(seat)
  const selected = isSelected(seat.id!)
  const selectable = isSelectable(seat)

  if (selected) {
    return {
      borderColor: color,
      backgroundColor: withAlpha(color, 0.35),
      color: color,
    }
  }

  if (seat.status === 'available' && selectable) {
    return {
      borderColor: color,
      backgroundColor: withAlpha(color, 0.2),
    }
  }

  if (seat.category_id && props.categoryColorById?.[seat.category_id]) {
    return {
      borderColor: withAlpha(color, 0.45),
      backgroundColor: withAlpha(color, 0.08),
    }
  }

  return {}
}

function onSeatClick(seat: Seat) {
  if (!isSelectable(seat)) return
  emit('toggle', seat.id!)
}
</script>

<template>
  <div class="space-y-4">
    <div v-if="gridBounds" class="overflow-x-auto">
      <div
        class="inline-grid gap-1.5"
        :style="{
          gridTemplateColumns: `repeat(${gridBounds.maxCol - gridBounds.minCol + 1}, minmax(2.75rem, 1fr))`,
        }"
      >
        <template v-for="row in gridBounds.maxRow - gridBounds.minRow + 1" :key="row">
          <button
            v-for="col in gridBounds.maxCol - gridBounds.minCol + 1"
            :key="`${row}-${col}`"
            type="button"
            class="flex h-11 min-w-11 items-center justify-center rounded-md border px-1 text-xs font-medium transition"
            :class="
              seatAt.get(`${row + gridBounds.minRow - 1}-${col + gridBounds.minCol - 1}`)
                ? seatClasses(seatAt.get(`${row + gridBounds.minRow - 1}-${col + gridBounds.minCol - 1}`)!)
                : 'border-transparent bg-transparent'
            "
            :style="
              seatAt.get(`${row + gridBounds.minRow - 1}-${col + gridBounds.minCol - 1}`)
                ? seatStyle(seatAt.get(`${row + gridBounds.minRow - 1}-${col + gridBounds.minCol - 1}`)!)
                : undefined
            "
            :disabled="
              !seatAt.get(`${row + gridBounds.minRow - 1}-${col + gridBounds.minCol - 1}`) ||
              !isSelectable(seatAt.get(`${row + gridBounds.minRow - 1}-${col + gridBounds.minCol - 1}`)!)
            "
            @click="
              seatAt.get(`${row + gridBounds.minRow - 1}-${col + gridBounds.minCol - 1}`) &&
                onSeatClick(seatAt.get(`${row + gridBounds.minRow - 1}-${col + gridBounds.minCol - 1}`)!)
            "
          >
            {{
              seatAt.get(`${row + gridBounds.minRow - 1}-${col + gridBounds.minCol - 1}`)?.code ?? ''
            }}
          </button>
        </template>
      </div>
    </div>

    <div v-if="unplacedSeats.length" class="space-y-2">
      <p class="text-xs font-medium uppercase tracking-wide text-ink-muted">Unplaced seats</p>
      <div class="flex flex-wrap gap-2">
        <button
          v-for="seat in unplacedSeats"
          :key="seat.id"
          type="button"
          class="rounded-md border px-3 py-1.5 text-xs font-medium transition"
          :class="seatClasses(seat)"
          :style="seatStyle(seat)"
          :disabled="!isSelectable(seat)"
          @click="onSeatClick(seat)"
        >
          {{ seat.code }}
        </button>
      </div>
    </div>

    <div class="flex flex-wrap gap-3 text-xs text-ink-muted">
      <span class="inline-flex items-center gap-1.5">
        <span class="h-3 w-3 rounded border border-border bg-surface-raised" /> Available (by category color)
      </span>
      <span class="inline-flex items-center gap-1.5">
        <span class="h-3 w-3 rounded border-2 border-accent bg-accent-soft ring-2 ring-accent/30" /> Selected
      </span>
      <span class="inline-flex items-center gap-1.5">
        <span class="h-3 w-3 rounded border border-warning/40 bg-warning-soft" /> Reserved
      </span>
      <span class="inline-flex items-center gap-1.5">
        <span class="h-3 w-3 rounded border border-dashed border-ink-muted/50 bg-surface" /> Held (draft)
      </span>
      <span class="inline-flex items-center gap-1.5">
        <span class="h-3 w-3 rounded border border-accent/40 bg-accent-soft/60" /> Occupied
      </span>
    </div>
  </div>
</template>
