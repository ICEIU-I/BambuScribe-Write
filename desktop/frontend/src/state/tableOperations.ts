import type { LayoutSettings } from '../types'
import type {
  TableAsset,
  TableCell,
  TableInsertOptions,
  TableSelection,
  TableBorders,
} from './tableTypes'
import { cellAt, selectedCells, selectionBounds, sum } from './tableGeometry'

export const cloneTable = (t: TableAsset): TableAsset => JSON.parse(JSON.stringify(t))
export const newTableId = () => `table-${Date.now()}-${Math.random().toString(16).slice(2)}`
const blank = (row: number, col: number): TableCell => ({
  row,
  col,
  rowSpan: 1,
  colSpan: 1,
  text: '',
})
function fill(table: TableAsset) {
  for (let r = 0; r < table.rowHeights.length; r++)
    for (let c = 0; c < table.columnWidths.length; c++)
      if (!cellAt(table, r, c)) table.cells.push(blank(r, c))
  table.cells.sort((a, b) => a.row - b.row || a.col - b.col)
  return table
}
export function createTable(
  options: TableInsertOptions,
  settings: LayoutSettings,
  position: { page: number; x?: number; y?: number },
): TableAsset {
  const { rows, cols } = options
  if (
    !Number.isInteger(rows) ||
    rows < 1 ||
    rows > 50 ||
    !Number.isInteger(cols) ||
    cols < 1 ||
    cols > 20
  )
    throw new Error('表格须为 1–50 行、1–20 列。')
  if (
    !Number.isFinite(options.width) ||
    options.width < cols * 3 ||
    options.width > settings.paperWidth
  )
    throw new Error('表格宽度须在每列 3 mm 与纸张宽度之间。')
  const width = options.width,
    fontSize = settings.fontSize,
    height = Math.max(8, fontSize * 1.5 + 2)
  return fill({
    id: newTableId(),
    page: position.page,
    x: Math.max(0, Math.min(position.x ?? settings.marginLeft, settings.paperWidth - width)),
    y: Math.max(
      0,
      Math.min(position.y ?? settings.marginTop, Math.max(0, settings.paperHeight - rows * height)),
    ),
    columnWidths: Array(cols).fill(width / cols),
    rowHeights: Array(rows).fill(height),
    cells: [],
    style: options.style,
    fontSize,
    padding: 1,
    align: 'center',
    verticalAlign: 'middle',
  })
}
export function mergeCells(source: TableAsset, selection: TableSelection) {
  const t = cloneTable(source),
    b = selectionBounds(t, selection),
    cells = selectedCells(t, b)
  if (cells.length < 2) return source
  const first = cells[0]!
  t.cells = t.cells.filter((c) => !cells.includes(c))
  t.cells.push({
    ...first,
    row: b.r0,
    col: b.c0,
    rowSpan: b.r1 - b.r0 + 1,
    colSpan: b.c1 - b.c0 + 1,
    text: cells
      .map((c) => c.text)
      .filter(Boolean)
      .join('\n'),
  })
  delete t.cells.at(-1)!.borders
  return fill(t)
}
export function splitCells(source: TableAsset, selection: TableSelection) {
  const t = cloneTable(source),
    cells = selectedCells(t, selection)
  for (const c of cells) {
    c.rowSpan = 1
    c.colSpan = 1
    delete c.borders
  }
  return fill(t)
}
export function changeAxis(source: TableAsset, axis: 'row' | 'col', index: number, remove = false) {
  const t = cloneTable(source),
    sizes = axis === 'row' ? t.rowHeights : t.columnWidths,
    length = sizes.length,
    span = axis === 'row' ? 'rowSpan' : 'colSpan'
  if (remove && length === 1) throw new Error('至少保留一行和一列；删除整张表格请使用删除表格。')
  if (!remove && length >= (axis === 'row' ? 50 : 20))
    throw new Error(axis === 'row' ? '表格最多 50 行。' : '表格最多 20 列。')
  index = Math.max(0, Math.min(remove ? length - 1 : length, index))
  if (remove) {
    sizes.splice(index, 1)
    t.cells = t.cells.filter((c) => !(c[axis] === index && c[span] === 1))
    for (const c of t.cells) {
      if (c[axis] > index) c[axis]--
      else if (c[axis] <= index && c[axis] + c[span] > index) c[span]--
    }
  } else {
    sizes.splice(index, 0, sizes[Math.min(index, length - 1)]!)
    for (const c of t.cells) {
      if (c[axis] >= index) c[axis]++
      else if (c[axis] + c[span] > index) c[span]++
    }
  }
  return fill(t)
}
export function setCellText(source: TableAsset, row: number, col: number, text: string) {
  const t = cloneTable(source),
    cell = cellAt(t, row, col)
  if (cell) cell.text = text
  return t
}
export function setCellOptions(
  source: TableAsset,
  selection: TableSelection,
  options: Partial<Pick<TableCell, 'align' | 'verticalAlign'>>,
) {
  const t = cloneTable(source)
  for (const c of selectedCells(t, selection)) Object.assign(c, options)
  return t
}
// Update both sides of shared borders. Hiding an edge then restoring it works
// even when the neighbouring cell was previously edited independently.
export function setBorders(
  source: TableAsset,
  selection: TableSelection,
  mode: 'all' | 'outer' | 'none' | 'reset',
) {
  const t = cloneTable(source),
    b = selectionBounds(t, selection)
  for (const c of selectedCells(t, b)) {
    const borders: TableBorders = {
      top: mode === 'all' || (mode === 'outer' && c.row === b.r0),
      bottom: mode === 'all' || (mode === 'outer' && c.row + c.rowSpan - 1 === b.r1),
      left: mode === 'all' || (mode === 'outer' && c.col === b.c0),
      right: mode === 'all' || (mode === 'outer' && c.col + c.colSpan - 1 === b.c1),
    }
    c.borders = mode === 'reset' ? undefined : borders
  }
  for (const c of t.cells) {
    if (c.row + c.rowSpan === b.r0 && c.col <= b.c1 && c.col + c.colSpan > b.c0)
      c.borders = { ...c.borders, bottom: mode === 'reset' ? undefined : mode !== 'none' }
    if (c.row === b.r1 + 1 && c.col <= b.c1 && c.col + c.colSpan > b.c0)
      c.borders = { ...c.borders, top: mode === 'reset' ? undefined : mode !== 'none' }
    if (c.col + c.colSpan === b.c0 && c.row <= b.r1 && c.row + c.rowSpan > b.r0)
      c.borders = { ...c.borders, right: mode === 'reset' ? undefined : mode !== 'none' }
    if (c.col === b.c1 + 1 && c.row <= b.r1 && c.row + c.rowSpan > b.r0)
      c.borders = { ...c.borders, left: mode === 'reset' ? undefined : mode !== 'none' }
  }
  return t
}
export function resizeTable(source: TableAsset, width: number, height: number, scaleText = false) {
  const t = cloneTable(source),
    sx = width / sum(t.columnWidths),
    sy = height / sum(t.rowHeights)
  if (
    !Number.isFinite(sx) ||
    !Number.isFinite(sy) ||
    sx <= 0 ||
    sy <= 0 ||
    t.columnWidths.some((v) => v * sx < 3) ||
    t.rowHeights.some((v) => v * sy < 3)
  )
    throw new Error('行高和列宽至少为 3 mm。')
  t.columnWidths = t.columnWidths.map((v) => v * sx)
  t.rowHeights = t.rowHeights.map((v) => v * sy)
  if (scaleText) t.fontSize = Math.max(1, Math.min(40, t.fontSize * Math.min(sx, sy)))
  return t
}
export function pasteCells(source: TableAsset, row: number, col: number, data: string[][]) {
  const nr = data.length,
    nc = Math.max(...data.map((r) => r.length))
  if (row + nr > 50 || col + nc > 20)
    throw new Error('粘贴后的表格超过 50 行或 20 列，请分块粘贴。')
  let t = cloneTable(source)
  while (t.rowHeights.length < row + nr) t = changeAxis(t, 'row', t.rowHeights.length)
  while (t.columnWidths.length < col + nc) t = changeAxis(t, 'col', t.columnWidths.length)
  const selected = { r0: row, c0: col, r1: row + nr - 1, c1: col + nc - 1 }
  t = splitCells(t, selected)
  for (let r = 0; r < nr; r++)
    for (let c = 0; c < nc; c++) {
      const cell = cellAt(t, row + r, col + c)
      if (cell) cell.text = data[r]?.[c] ?? ''
    }
  return t
}
