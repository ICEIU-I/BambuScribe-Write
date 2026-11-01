import type { Point } from '../types'
import type { TableAsset, TablePlacement } from './tableTypes'
import { offsets } from './tableGeometry'

// Use the engine's fraction bars as well as its letters. Map row/column
// coordinates during a resize; edited topology waits for a fresh preview.
export function previewTableRules(asset: TableAsset, preview?: TablePlacement): Point[][] | null {
  if (
    !preview ||
    preview.style !== asset.style ||
    preview.columnWidths.length !== asset.columnWidths.length ||
    preview.rowHeights.length !== asset.rowHeights.length ||
    preview.cells.length !== asset.cells.length ||
    !asset.cells.every((c, i) => {
      const old = preview.cells[i]!
      return (
        c.row === old.row &&
        c.col === old.col &&
        c.rowSpan === old.rowSpan &&
        c.colSpan === old.colSpan &&
        JSON.stringify(c.borders) === JSON.stringify(old.borders)
      )
    })
  )
    return null
  const count = preview.strokes.length - preview.cells.reduce((n, c) => n + c.strokes.length, 0)
  const oldX = offsets(preview.columnWidths),
    newX = offsets(asset.columnWidths)
  const oldY = offsets(preview.rowHeights),
    newY = offsets(asset.rowHeights)
  const map = (value: number, from: number[], to: number[]) => {
    let i = 0
    while (i < from.length - 2 && value > from[i + 1]!) i++
    if (from[i] === to[i] && from[i + 1] === to[i + 1]) return value
    return to[i]! + ((value - from[i]!) * (to[i + 1]! - to[i]!)) / (from[i + 1]! - from[i]!)
  }
  return preview.strokes
    .slice(0, count)
    .map((s) => s.map((p) => ({ x: map(p.x, oldX, newX), y: map(p.y, oldY, newY) })))
}
