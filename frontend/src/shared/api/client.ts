import { ApiError } from '@/shared/lib/errors'
import type { ErrorEnvelope, SuccessEnvelope } from './types'

const baseUrl = import.meta.env.VITE_API_BASE_URL ?? '/api'

type RequestOptions = Omit<RequestInit, 'body'> & {
  body?: unknown
  token?: string | null
  skipAuth?: boolean
}

let unauthorizedHandler: (() => void) | null = null

export function setUnauthorizedHandler(handler: () => void) {
  unauthorizedHandler = handler
}

export async function apiRequest<T>(
  path: string,
  options: RequestOptions = {},
): Promise<T> {
  const { body, token, skipAuth, headers, ...rest } = options

  const requestHeaders = new Headers(headers)
  requestHeaders.set('Accept', 'application/json')

  if (body !== undefined) {
    requestHeaders.set('Content-Type', 'application/json')
  }

  if (!skipAuth && token) {
    requestHeaders.set('Authorization', `Bearer ${token}`)
  }

  const response = await fetch(`${baseUrl}${path}`, {
    ...rest,
    headers: requestHeaders,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })

  if (response.status === 204) {
    return undefined as T
  }

  const isHealth = path === '/health'
  const payload = (await response.json()) as SuccessEnvelope<T> | ErrorEnvelope | T

  if (isHealth) {
    return payload as T
  }

  if (!response.ok) {
    const errorPayload = payload as ErrorEnvelope
    if (response.status === 401 && unauthorizedHandler) {
      unauthorizedHandler()
    }
    throw new ApiError(
      response.status,
      errorPayload.error?.code ?? 'UNKNOWN_ERROR',
      errorPayload.error?.message ?? response.statusText,
    )
  }

  const envelope = payload as SuccessEnvelope<T>
  if (envelope && typeof envelope === 'object' && 'success' in envelope) {
    if (!envelope.success) {
      const errorPayload = envelope as unknown as ErrorEnvelope
      throw new ApiError(
        response.status,
        errorPayload.error?.code ?? 'UNKNOWN_ERROR',
        errorPayload.error?.message ?? 'Request failed',
      )
    }
    return envelope.data
  }

  return payload as T
}

export function apiGet<T>(path: string, token?: string | null) {
  return apiRequest<T>(path, { method: 'GET', token })
}

export function apiPost<T>(path: string, body: unknown, token?: string | null) {
  return apiRequest<T>(path, { method: 'POST', body, token })
}

export function apiPatch<T>(path: string, body: unknown, token?: string | null) {
  return apiRequest<T>(path, { method: 'PATCH', body, token })
}

export function apiPut<T>(path: string, body: unknown, token?: string | null) {
  return apiRequest<T>(path, { method: 'PUT', body, token })
}

export function apiDelete<T>(path: string, token?: string | null) {
  return apiRequest<T>(path, { method: 'DELETE', token })
}
