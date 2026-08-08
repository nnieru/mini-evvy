const CATEGORY_PALETTE = [
  '#0d9488',
  '#c2410c',
  '#0369a1',
  '#b45309',
  '#be123c',
  '#4d7c0f',
  '#7c3aed',
  '#0e7490',
  '#a16207',
  '#9d174d',
] as const

export function buildCategoryColorMap(categoryIds: string[]): Record<string, string> {
  const sorted = [...categoryIds].sort()
  const map: Record<string, string> = {}
  sorted.forEach((id, index) => {
    map[id] = CATEGORY_PALETTE[index % CATEGORY_PALETTE.length]
  })
  return map
}

export function withAlpha(hex: string, alpha: number): string {
  const normalized = hex.replace('#', '')
  const r = Number.parseInt(normalized.slice(0, 2), 16)
  const g = Number.parseInt(normalized.slice(2, 4), 16)
  const b = Number.parseInt(normalized.slice(4, 6), 16)
  return `rgba(${r}, ${g}, ${b}, ${alpha})`
}
