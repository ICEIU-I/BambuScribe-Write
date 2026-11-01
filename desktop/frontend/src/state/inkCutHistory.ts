import type { Document, InkCut, PageErasure } from '../types'
import type { ImageAsset, TableAsset } from '../types'

// A normal textarea edit replaces one range. Retain source characters on both
// sides; new or replaced characters get fresh ink even at the same position.
export function rebaseBodyCuts(
  erasures: PageErasure[],
  before: string,
  after: string,
): PageErasure[] {
  if (before === after) return erasures
  let start = 0,
    end = before.length,
    nextEnd = after.length
  while (start < end && start < nextEnd && before[start] === after[start]) start++
  while (end > start && nextEnd > start && before[end - 1] === after[nextEnd - 1]) {
    end--
    nextEnd--
  }
  return mapBodyCuts(erasures, (n) => (n < start ? n : n >= end ? n + nextEnd - end : null))
}
export function mapBodyCuts(
  erasures: PageErasure[],
  map: (n: number) => number | null,
): PageErasure[] {
  return erasures.map((e) => ({
    ...e,
    ...(e.cuts
      ? {
          cuts: e.cuts.flatMap((c) => {
            if (c.objectId) return [c]
            const n = map(Number(c.part))
            return n === null ? [] : [{ ...c, part: String(n) }]
          }),
        }
      : {}),
  }))
}
// Page operations insert explicit separators, so map source offsets by sheet.
export function remapPageCuts(
  erasures: PageErasure[],
  doc: Document,
  order: number[],
  sections: string[],
): PageErasure[] {
  const oldStarts: number[] = [],
    newStarts = new Map<number, number>()
  let n = 0
  for (const p of doc.pages) {
    oldStarts.push(n)
    n += p.text.length
  }
  n = 0
  order.forEach((old, i) => {
    newStarts.set(old, n)
    n += sections[i]!.length + 1
  })
  return mapBodyCuts(erasures, (index) => {
    let p = oldStarts.length - 1
    while (p > 0 && oldStarts[p]! > index) p--
    const target = newStarts.get(p),
      local = index - oldStarts[p]!
    return target === undefined || local >= sections[order.indexOf(p)]!.length
      ? null
      : target + local
  })
}
export function cutsOnPage(e: PageErasure, doc: Document, index: number): InkCut[] | undefined {
  if (!e.cuts) return undefined
  const page = doc.pages[index]!,
    start = doc.pages.slice(0, index).reduce((n, p) => n + p.text.length, 0)
  const ids = new Set([...(page.images ?? []), ...(page.tables ?? [])].map((o) => o.id))
  return e.cuts.filter((c) =>
    c.objectId
      ? ids.has(c.objectId)
      : Number(c.part) >= start && Number(c.part) < start + page.text.length,
  )
}

export function resetChangedObjectCuts(
  erasures: PageErasure[],
  beforeTables: TableAsset[],
  afterTables: TableAsset[],
  beforeImages: ImageAsset[],
  afterImages: ImageAsset[],
): PageErasure[] {
  const changed = new Map<string, Set<string> | null>()
  for (const old of beforeTables) {
    const next = afterTables.find((t) => t.id === old.id)
    if (!next) {
      changed.set(old.id, null)
      continue
    }
    const topology = (t: TableAsset) =>
      JSON.stringify([
        t.style,
        t.columnWidths.length,
        t.rowHeights.length,
        t.cells.map((c) => [c.row, c.col, c.rowSpan, c.colSpan, c.borders]),
      ])
    if (topology(old) !== topology(next)) {
      changed.set(old.id, null)
      continue
    }
    const parts = new Set(
      old.cells
        .filter((c) => next.cells.find((v) => v.row === c.row && v.col === c.col)?.text !== c.text)
        .map((c) => `cell:${c.row}:${c.col}`),
    )
    if (parts.size) changed.set(old.id, parts)
  }
  for (const old of beforeImages) {
    const next = afterImages.find((i) => i.id === old.id)
    if (
      !next ||
      old.data !== next.data ||
      old.mode !== next.mode ||
      JSON.stringify(old.diagram) !== JSON.stringify(next.diagram)
    ) {
      changed.set(old.id, null)
      continue
    }
    const parts = new Set(
      (old.textRegions ?? [])
        .filter((r) => {
          const n = next.textRegions?.find((v) => v.id === r.id)
          return !n || n.text !== r.text || n.textFormat !== r.textFormat || n.mode !== r.mode
        })
        .map((r) => `region:${r.id}`),
    )
    if (parts.size) changed.set(old.id, parts)
  }
  if (!changed.size) return erasures
  return erasures.map((e) => ({
    ...e,
    ...(e.cuts
      ? {
          cuts: e.cuts.filter(
            (c) =>
              !changed.has(c.objectId) ||
              (changed.get(c.objectId)! && !changed.get(c.objectId)!.has(c.part)),
          ),
        }
      : {}),
  }))
}
