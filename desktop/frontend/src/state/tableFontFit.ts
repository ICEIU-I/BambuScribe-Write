import type { TableAsset, TablePlacement } from './tableTypes'
import { cellBounds, sum } from './tableGeometry'
import { cloneTable } from './tableOperations'

// Match native table fitting while resizing: both grow and shrink to the largest
// uniform size. Native metrics include formulas, merged cells and explicit lines.
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
          width: old.textWidth / preview.fontSize!,
          height: old.textHeight / preview.fontSize!,
        },
      ]
    })
  let size = metrics.length !== table.cells.length ? table.fontSize : 40
  if (metrics.every(({ width, height }) => !width && !height))
    size = Math.min(size, ...table.rowHeights.map((height) => height - 2 * table.padding))
  for (const { cell, width, height } of metrics) {
    const box = cellBounds(table, cell)
    if (width > 0) size = Math.min(size, (box.width - 2 * table.padding) / width)
    if (height > 0) size = Math.min(size, (box.height - 2 * table.padding) / height)
  }
  table.fontSize = Math.max(1, size)
  // At 1 mm, stop shrinking the cells instead of clipping text or bouncing
  // back to the previous font's row heights after the native preview arrives.
  for (const { cell, width, height } of metrics) {
    const w = sum(table.columnWidths.slice(cell.col, cell.col + cell.colSpan)),
      h = sum(table.rowHeights.slice(cell.row, cell.row + cell.rowSpan)),
      dw = Math.max(0, width * table.fontSize + 2 * table.padding - w),
      dh = Math.max(0, height * table.fontSize + 2 * table.padding - h)
    if (dw > 1e-8)
      for (let col = cell.col; col < cell.col + cell.colSpan; col++)
        table.columnWidths[col]! += dw / cell.colSpan
    if (dh > 1e-8) table.rowHeights[cell.row + cell.rowSpan - 1]! += dh
  }
  return table
}
