import type { CaretStop, Document } from '../types'

export interface LocatedCaret extends CaretStop {
  page: number
  blankLines?: number
}
export interface CaretRow {
  page: number
  y: number
  stops: LocatedCaret[]
}

export function caretAt(
  doc: Document | null,
  index: number,
  preferred: LocatedCaret | null = null,
): LocatedCaret {
  let result: LocatedCaret = {
    index: 0,
    page: 0,
    x: doc?.settings.marginLeft ?? 25.4,
    y: doc?.settings.marginTop ?? 25.4,
  }
  let distance = Infinity
  doc?.pages.forEach((page, pi) =>
    page.carets.forEach((stop) => {
      if (Math.abs(stop.index - index) <= distance) {
        result = { ...stop, page: pi }
        distance = Math.abs(stop.index - index)
      }
    }),
  )
  if (preferred?.index === index) {
    const match = doc?.pages[preferred.page]?.carets.find(
      (stop) => stop.index === index && stop.y === preferred.y && stop.x === preferred.x,
    )
    if (match) return { ...match, page: preferred.page }
  }
  return result
}

export function rowsFor(doc: Document | null): CaretRow[] {
  const rows: CaretRow[] = []
  doc?.pages.forEach((page, pi) => {
    const byY = new Map<number, LocatedCaret[]>()
    page.carets.forEach((stop) => {
      const row = byY.get(stop.y) ?? []
      row.push({ ...stop, page: pi })
      byY.set(stop.y, row)
    })
    for (const [y, stops] of byY) rows.push({ page: pi, y, stops })
  })
  return rows
}

export function hitCaret(
  doc: Document,
  page: number,
  x: number,
  y: number,
  rows = rowsFor(doc),
): LocatedCaret {
  const stops = rows.filter((row) => row.page === page).flatMap((row) => row.stops)
  const height = Math.min(
    doc.settings.fontSize,
    doc.pages[page]?.lineHeight ?? doc.settings.fontSize,
  )
  let nearest = stops[0]
  let score = Infinity
  for (const stop of stops) {
    const rowDistance = Math.abs(stop.y + height / 2 - y)
    const value = rowDistance * 10000 + Math.abs(stop.x - x)
    if (value < score) {
      score = value
      nearest = stop
    }
  }
  return nearest ? { ...nearest, page } : caretAt(doc, 0)
}
