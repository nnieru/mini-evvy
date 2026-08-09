import { ref, watch, type Ref } from 'vue'

export function useDebouncedSearch(onChange?: () => void, delayMs = 300) {
  const searchInput = ref('')
  const searchQuery = ref('')
  let timer: ReturnType<typeof setTimeout> | undefined

  watch(searchInput, (value) => {
    clearTimeout(timer)
    timer = setTimeout(() => {
      searchQuery.value = value.trim()
      onChange?.()
    }, delayMs)
  })

  return { searchInput, searchQuery }
}

export function resetPageOnSearch(page: Ref<number>) {
  return () => {
    page.value = 1
  }
}
