import type { TableAsset, TableCell, TableCellPlacement, TablePlacement } from './tableTypes'
import { cellBounds, sum } from './tableGeometry'
import { cloneTable } from './tableOperations'

function lineMetrics(old: TableCellPlacement) {
  return old.textSpacing?.lines ?? [{ width: old.textWidth ?? 0, gap: 0 }]
}

// Reuse native ink metrics while resizing, including formulas and explicit lines.
export function fitTableCellFont(
  table: TableAsset,
  cell: TableCell,
  old?: TableCellPlacement,
  fallback = table.fontSize,
): number {
  if (!old || old.text !== cell.text || old.textWidth === undefined || old.textHeight === undefined)
    return fallback
  if (!cell.text) return old.fontSize ?? fallback
  const box = cellBounds(table, cell),
    previous = old.fontSize ?? fallback
  let size = 40
  for (const line of lineMetrics(old))
    if (line.width > 0)
      size = Math.min(size, ((box.width - 2 * table.padding - line.gap) * previous) / line.width)
  if (old.textHeight > 0)
    size = Math.min(size, ((box.height - 2 * table.padding) * previous) / old.textHeight)
  return Math.max(1, size)
}

export function fitTableFont(source: TableAsset, preview?: TablePlacement): TableAsset {
  if (!preview?.fontSize) return source
  const table = cloneTable(source),
    metrics = table.cells.flatMap((cell) => {
      const old = preview.cells.find((c) => c.row === cell.row && c.col === cell.col)
      if (
        !old ||
        old.text !== cell.text ||
        old.textWidth === undefined ||
        old.textHeight === undefined
      )
        return []
      return [
        {
          cell,
          lines: lineMetrics(old).map((line) => ({
            width: line.width / (old.fontSize ?? preview.fontSize!),
            gap: line.gap,
          })),
          height: old.textHeight / (old.fontSize ?? preview.fontSize!),
        },
      ]
    })
  let size = metrics.length !== table.cells.length ? table.fontSize : 40
  if (metrics.every(({ lines, height }) => lines.every((line) => !line.width) && !height))
    size = Math.min(size, ...table.rowHeights.map((height) => height - 2 * table.padding))
  for (const { cell, lines, height } of metrics) {
    const box = cellBounds(table, cell)
    for (const line of lines)
      if (line.width > 0)
        size = Math.min(size, (box.width - 2 * table.padding - line.gap) / line.width)
    if (height > 0) size = Math.min(size, (box.height - 2 * table.padding) / height)
  }
  table.fontSize = Math.max(1, size)
  // At 1 mm, stop shrinking the cells instead of clipping text or bouncing
  // back to the previous font's row heights after the native preview arrives.
  for (const { cell, lines, height } of metrics) {
    const w = sum(table.columnWidths.slice(cell.col, cell.col + cell.colSpan)),
      h = sum(table.rowHeights.slice(cell.row, cell.row + cell.rowSpan)),
      width = Math.max(0, ...lines.map((line) => line.width * table.fontSize + line.gap)),
      dw = Math.max(0, width + 2 * table.padding - w),
      dh = Math.max(0, height * table.fontSize + 2 * table.padding - h)
    if (dw > 1e-8)
      for (let col = cell.col; col < cell.col + cell.colSpan; col++)
        table.columnWidths[col]! += dw / cell.colSpan
    if (dh > 1e-8) table.rowHeights[cell.row + cell.rowSpan - 1]! += dh
  }
  return table
}
