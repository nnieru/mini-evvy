export type PaginatedList<T> = {
  items: T[]
  total: number
  page: number
  page_size: number
}

export type ListPageParams = {
  page?: number
  page_size?: number
  q?: string
}

export function appendListPageParams(search: URLSearchParams, params?: ListPageParams) {
  if (params?.page) search.set('page', String(params.page))
  if (params?.page_size) search.set('page_size', String(params.page_size))
  if (params?.q) search.set('q', params.q)
}

export async function fetchAllPages<T>(
  fetchPage: (page: number, pageSize: number) => Promise<PaginatedList<T>>,
  pageSize = 100,
): Promise<T[]> {
  const first = await fetchPage(1, pageSize)
  const all = [...first.items]
  const pages = Math.ceil(first.total / pageSize)
  for (let p = 2; p <= pages; p++) {
    const next = await fetchPage(p, pageSize)
    all.push(...next.items)
  }
  return all
}
