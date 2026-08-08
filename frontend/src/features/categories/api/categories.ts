import { apiGet, apiPatch, apiPost } from '@/shared/api/client'
import type {
  CreateCategoryRequest,
  SeatCategory,
  UpdateCategoryRequest,
} from '@/shared/api/types'

export function listCategories(eventId: string, token: string) {
  return apiGet<SeatCategory[]>(`/events/${eventId}/categories`, token)
}

export function getCategory(categoryId: string, token: string) {
  return apiGet<SeatCategory>(`/categories/${categoryId}`, token)
}

export function createCategory(eventId: string, body: CreateCategoryRequest, token: string) {
  return apiPost<SeatCategory>(`/events/${eventId}/categories`, body, token)
}

export function updateCategory(categoryId: string, body: UpdateCategoryRequest, token: string) {
  return apiPatch<SeatCategory>(`/categories/${categoryId}`, body, token)
}
