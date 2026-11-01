import type { TableAsset, TableSelection } from './tableTypes'
import { cellAt } from './tableGeometry'

export const tableMime = 'application/x-bambuscribe-table-v1'
// Excel quotes cells containing tabs/newlines; retain those as cell content.
export function parseTableText(text: string): string[][] {
  const rows: string[][] = [],
    row: string[] = []
  let value = '',
    quoted = false
  text = text.replace(/\r\n?/g, '\n')
  for (let i = 0; i < text.length; i++) {
    const c = text[i]!
    if (c === '"' && (quoted || value === '')) {
      if (quoted && text[i + 1] === '"') {
        value += '"'
        i++
      } else quoted = !quoted
    } else if (!quoted && (c === '\t' || c === '\n')) {
      row.push(value)
      value = ''
      if (c === '\n') {
        rows.push([...row])
        row.length = 0
      }
    } else value += c
  }
  row.push(value)
  if (row.length > 1 || value !== '' || !rows.length) rows.push(row)
  return rows
}
const quote = (text: string) => (/[\t\n\r"]/.test(text) ? `"${text.replace(/"/g, '""')}"` : text)
export function tableText(table: TableAsset, selection?: TableSelection) {
  const b = selection ?? {
      r0: 0,
      c0: 0,
      r1: table.rowHeights.length - 1,
      c1: table.columnWidths.length - 1,
    },
    rows: string[] = []
  for (let r = b.r0; r <= b.r1; r++) {
    const values: string[] = []
    for (let c = b.c0; c <= b.c1; c++) {
      const owner = cellAt(table, r, c)
      values.push(quote(owner?.row === r && owner.col === c ? owner.text : ''))
    }
    rows.push(values.join('\t'))
  }
  return rows.join('\n')
}
export function readTable(data: Pick<DataTransfer, 'getData'>): TableAsset | undefined {
  try {
    const custom = data.getData(tableMime)
    const html = custom ? '' : data.getData('text/html')
    const embedded = html.match(/data-bambuscribe-table="([^"]+)"/)
    const table = JSON.parse(
      custom || (embedded ? decodeURIComponent(embedded[1]!) : ''),
    ) as TableAsset
    if (
      !table ||
      (table.kind !== undefined && table.kind !== 'textbox') ||
      typeof table.id !== 'string' ||
      !Array.isArray(table.columnWidths) ||
      !Array.isArray(table.rowHeights) ||
      !Array.isArray(table.cells) ||
      table.columnWidths.length < 1 ||
      table.columnWidths.length > 20 ||
      table.rowHeights.length < 1 ||
      table.rowHeights.length > 50 ||
      table.cells.length > 1000
    )
      return
    if (
      table.kind === 'textbox' &&
      (table.columnWidths.length !== 1 ||
        table.rowHeights.length !== 1 ||
        table.cells.length !== 1 ||
        table.style !== 'none' ||
        table.cells[0]?.borders)
    )
      return
    if (
      !Number.isInteger(table.page) ||
      table.page < 0 ||
      table.page > 20000 ||
      table.fontSize < 1 ||
      table.fontSize > 40 ||
      table.padding < 0 ||
      table.padding > 10 ||
      !['grid', 'three-line', 'horizontal', 'none'].includes(table.style) ||
      !['left', 'center', 'right'].includes(table.align) ||
      !['top', 'middle', 'bottom'].includes(table.verticalAlign) ||
      [...table.columnWidths, ...table.rowHeights].some((v) => v < 3 || v > 1000)
    )
      return
    if (
      ![
        table.x,
        table.y,
        table.fontSize,
        table.padding,
        ...table.columnWidths,
        ...table.rowHeights,
      ].every(Number.isFinite)
    )
      return
    if (
      !table.cells.every(
        (c) =>
          typeof c.text === 'string' &&
          [c.row, c.col, c.rowSpan, c.colSpan].every(Number.isInteger) &&
          c.row >= 0 &&
          c.col >= 0 &&
          c.rowSpan >= 1 &&
          c.colSpan >= 1 &&
          c.row + c.rowSpan <= table.rowHeights.length &&
          c.col + c.colSpan <= table.columnWidths.length,
      )
    )
      return
    const owners = new Set<string>()
    for (const c of table.cells)
      for (let r = c.row; r < c.row + c.rowSpan; r++)
        for (let col = c.col; col < c.col + c.colSpan; col++) {
          const key = `${r}:${col}`
          if (owners.has(key)) return
          owners.add(key)
        }
    if (owners.size !== table.columnWidths.length * table.rowHeights.length) return
    delete table.groupId
    return table
  } catch {
    return
  }
}

// HTML is preserved by native WebViews even when custom clipboard types are
// stripped. Other spreadsheet apps can still read the standard table markup.
export function writeTable(data: DataTransfer, table: TableAsset) {
  data.setData(tableMime, JSON.stringify(table))
  data.setData('text/plain', tableText(table))
  const escape = (text: string) =>
    text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
  const rows = table.rowHeights
    .map(
      (_, row) =>
        '<tr>' +
        table.cells
          .filter((c) => c.row === row)
          .sort((a, b) => a.col - b.col)
          .map(
            (c) =>
              `<td rowspan="${c.rowSpan}" colspan="${c.colSpan}">${escape(c.text).replace(/\n/g, '<br>')}</td>`,
          )
          .join('') +
        '</tr>',
    )
    .join('')
  data.setData(
    'text/html',
    `<table data-bambuscribe-table="${encodeURIComponent(JSON.stringify(table))}">${rows}</table>`,
  )
}
