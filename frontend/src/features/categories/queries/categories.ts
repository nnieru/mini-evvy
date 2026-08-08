import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'
import { createCategory, listCategories, updateCategory } from '../api/categories'
import { useAuthStore } from '@/features/auth/stores/auth'
import type { CreateCategoryRequest, UpdateCategoryRequest } from '@/shared/api/types'

export const categoryKeys = {
  list: (eventId: string) => ['categories', eventId] as const,
}

export function useCategoriesQuery(eventId: Ref<string>) {
  const auth = useAuthStore()
  return useQuery({
    queryKey: computed(() => categoryKeys.list(eventId.value)),
    queryFn: () => listCategories(eventId.value, auth.token!),
    enabled: computed(() => Boolean(auth.token && eventId.value)),
  })
}

export function useCreateCategoryMutation(eventId: Ref<string>) {
  const auth = useAuthStore()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: CreateCategoryRequest) =>
      createCategory(eventId.value, body, auth.token!),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: categoryKeys.list(eventId.value) }),
  })
}

export function useUpdateCategoryMutation(eventId: Ref<string>) {
  const auth = useAuthStore()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ categoryId, body }: { categoryId: string; body: UpdateCategoryRequest }) =>
      updateCategory(categoryId, body, auth.token!),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: categoryKeys.list(eventId.value) }),
  })
}
