import type { TableAsset, TableCell, TableSelection } from './tableTypes'
export const sum = (values: number[]) => values.reduce((n, v) => n + v, 0)
export function offsets(sizes: number[]) {
  const points = [0]
  for (const size of sizes) points.push(points.at(-1)! + size)
  return points
}
export function cellAt(table: TableAsset, row: number, col: number): TableCell | undefined {
  return table.cells.find(
    (c) => row >= c.row && row < c.row + c.rowSpan && col >= c.col && col < c.col + c.colSpan,
  )
}
export function cellBounds(table: TableAsset, cell: TableCell) {
  const xs = offsets(table.columnWidths),
    ys = offsets(table.rowHeights)
  return {
    x: xs[cell.col]!,
    y: ys[cell.row]!,
    width: xs[cell.col + cell.colSpan]! - xs[cell.col]!,
    height: ys[cell.row + cell.rowSpan]! - ys[cell.row]!,
  }
}
// Expand selection until every intersecting merge is completely included.
export function selectionBounds(table: TableAsset, selection: TableSelection): TableSelection {
  const result = {
    r0: Math.max(0, Math.min(table.rowHeights.length - 1, selection.r0, selection.r1)),
    c0: Math.max(0, Math.min(table.columnWidths.length - 1, selection.c0, selection.c1)),
    r1: Math.max(0, Math.min(table.rowHeights.length - 1, Math.max(selection.r0, selection.r1))),
    c1: Math.max(0, Math.min(table.columnWidths.length - 1, Math.max(selection.c0, selection.c1))),
  }
  let changed = true
  while (changed) {
    changed = false
    for (const c of table.cells) {
      if (
        c.row > result.r1 ||
        c.col > result.c1 ||
        c.row + c.rowSpan <= result.r0 ||
        c.col + c.colSpan <= result.c0
      )
        continue
      const before = `${result.r0},${result.c0},${result.r1},${result.c1}`
      result.r0 = Math.min(result.r0, c.row)
      result.c0 = Math.min(result.c0, c.col)
      result.r1 = Math.max(result.r1, c.row + c.rowSpan - 1)
      result.c1 = Math.max(result.c1, c.col + c.colSpan - 1)
      if (before !== `${result.r0},${result.c0},${result.r1},${result.c1}`) changed = true
    }
  }
  return result
}
export function selectedCells(table: TableAsset, selection: TableSelection) {
  const b = selectionBounds(table, selection)
  return table.cells.filter(
    (c) =>
      c.row >= b.r0 &&
      c.col >= b.c0 &&
      c.row + c.rowSpan - 1 <= b.r1 &&
      c.col + c.colSpan - 1 <= b.c1,
  )
}
export function tableLines(table: TableAsset) {
  const rows = table.rowHeights.length,
    cols = table.columnWidths.length
  const owner = Array.from({ length: rows }, (_, r) =>
    Array.from({ length: cols }, (_, c) => cellAt(table, r, c)),
  )
  const h = Array.from({ length: rows + 1 }, (_, r) =>
    Array.from(
      { length: cols },
      (_, c) =>
        (r === 0 || r === rows || owner[r - 1]?.[c] !== owner[r]?.[c]) &&
        (table.style === 'grid' ||
          table.style === 'horizontal' ||
          (table.style === 'three-line' && (r === 0 || r === 1 || r === rows))),
    ),
  )
  const v = Array.from({ length: cols + 1 }, (_, c) =>
    Array.from(
      { length: rows },
      (_, r) =>
        table.style === 'grid' && (c === 0 || c === cols || owner[r]?.[c - 1] !== owner[r]?.[c]),
    ),
  )
  for (const enabled of [true, false])
    for (const c of table.cells) {
      const b = c.borders
      if (!b) continue
      for (let col = c.col; col < c.col + c.colSpan; col++) {
        if (b.top === enabled) h[c.row]![col] = enabled
        if (b.bottom === enabled) h[c.row + c.rowSpan]![col] = enabled
      }
      for (let row = c.row; row < c.row + c.rowSpan; row++) {
        if (b.left === enabled) v[c.col]![row] = enabled
        if (b.right === enabled) v[c.col + c.colSpan]![row] = enabled
      }
    }
  const xs = offsets(table.columnWidths),
    ys = offsets(table.rowHeights),
    lines: { x: number; y: number }[][] = []
  h.forEach((edges, r) => {
    for (let c = 0; c < cols; ) {
      if (!edges[c]) {
        c++
        continue
      }
      const start = c
      while (c < cols && edges[c]) c++
      lines.push([
        { x: xs[start]!, y: ys[r]! },
        { x: xs[c]!, y: ys[r]! },
      ])
    }
  })
  v.forEach((edges, c) => {
    for (let r = 0; r < rows; ) {
      if (!edges[r]) {
        r++
        continue
      }
      const start = r
      while (r < rows && edges[r]) r++
      lines.push([
        { x: xs[c]!, y: ys[start]! },
        { x: xs[c]!, y: ys[r]! },
      ])
    }
  })
  return lines
}
