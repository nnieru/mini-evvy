<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import PageHeader from '@/shared/ui/PageHeader.vue'
import Card from '@/shared/ui/Card.vue'
import Button from '@/shared/ui/Button.vue'
import Input from '@/shared/ui/Input.vue'
import Modal from '@/shared/ui/Modal.vue'
import LoadingSpinner from '@/shared/ui/LoadingSpinner.vue'
import EmptyState from '@/shared/ui/EmptyState.vue'
import Alert from '@/shared/ui/Alert.vue'
import { getErrorMessage } from '@/shared/lib/errors'
import { formatMoney } from '@/shared/lib/format'
import {
  useCategoriesQuery,
  useCreateCategoryMutation,
} from '../queries/categories'

const route = useRoute()
const eventId = computed(() => route.params.eventId as string)

const showCreate = ref(false)
const error = ref('')
const form = reactive({ name: '', code: '', price: '0', currency: 'IDR' })

const { data: categories, isLoading } = useCategoriesQuery(eventId)
const createMutation = useCreateCategoryMutation(eventId)

async function createCategory() {
  error.value = ''
  try {
    await createMutation.mutateAsync({
      name: form.name,
      code: form.code || null,
      price: Number(form.price),
      currency: form.currency,
    })
    showCreate.value = false
    form.name = ''
    form.code = ''
    form.price = '0'
  } catch (err) {
    error.value = getErrorMessage(err)
  }
}
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <PageHeader title="Categories" description="Seat categories and pricing for this event." />
      <Button @click="showCreate = true">New category</Button>
    </div>

    <LoadingSpinner v-if="isLoading" />
    <EmptyState v-else-if="!categories?.length" title="No categories yet" description="Add categories before creating seats and guests." />

    <div v-else class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      <Card v-for="category in categories" :key="category.id">
        <h3 class="font-display text-lg font-semibold text-ink">{{ category.name }}</h3>
        <p class="mt-1 text-sm text-ink-muted">{{ category.code || 'No code' }}</p>
        <p class="mt-3 text-sm font-medium text-accent">
          {{ formatMoney(String(category.price), category.currency) }}
        </p>
      </Card>
    </div>

    <Modal :open="showCreate" title="Create category" @close="showCreate = false">
      <form class="space-y-4" @submit.prevent="createCategory">
        <Alert v-if="error" tone="error" :title="error" />
        <Input v-model="form.name" label="Name" required />
        <Input v-model="form.code" label="Code" />
        <Input v-model="form.price" label="Price" type="number" min="0" step="0.01" required />
        <Input v-model="form.currency" label="Currency" required />
        <div class="flex justify-end gap-2">
          <Button variant="secondary" type="button" @click="showCreate = false">Cancel</Button>
          <Button type="submit" :loading="createMutation.isPending.value">Create</Button>
        </div>
      </form>
    </Modal>
  </div>
</template>
