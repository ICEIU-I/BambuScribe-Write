import type { Document, WritingSettings } from '../types'

export interface OriginRange {
  x: [number, number]
  y: [number, number]
}

// Auto origin relabels the starting position; it cannot add physical travel.
// Include the start/park and every page so one position is safe for the batch.
export function writingOriginRange(
  document: Document | null,
  writing: WritingSettings,
): OriginRange | null {
  if (!document || writing.originMode !== 'auto') return null
  const { marginLeft, marginTop } = document.settings
  if (![marginLeft, marginTop, writing.offsetX, writing.offsetY].every(Number.isFinite)) return null
  let minX = 0,
    maxX = 0,
    minDown = 0,
    maxDown = 0,
    hasInk = false
  for (const page of document.pages) {
    for (const stroke of page.strokes) {
      for (const point of stroke) {
        const x = point.x - marginLeft + writing.offsetX
        const down = point.y - marginTop + writing.offsetY
        if (!Number.isFinite(x) || !Number.isFinite(down)) return null
        minX = Math.min(minX, x)
        maxX = Math.max(maxX, x)
        minDown = Math.min(minDown, down)
        maxDown = Math.max(maxDown, down)
        hasInk = true
      }
    }
  }
  if (!hasInk) return null
  // Round inward to tenths so the displayed interval remains safe.
  const lower = (value: number) => Math.max(0, Math.ceil(value * 10 - 1e-8) / 10)
  const upper = (value: number) => Math.floor(value * 10 + 1e-8) / 10
  const x: [number, number] = [lower(-minX), upper(256 - maxX)]
  const y: [number, number] = [lower(maxDown), upper(256 + minDown)]
  if (x[0] > x[1] || y[0] > y[1]) return null
  return { x, y }
}
