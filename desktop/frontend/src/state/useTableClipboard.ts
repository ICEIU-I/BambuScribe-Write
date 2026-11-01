import type { Document, TableAsset } from '../types'
import type { TableInsertOptions } from './tableTypes'
import { createTable, cloneTable, newTableId, pasteCells } from './tableOperations'
import { parseTableText, readTable } from './tableClipboard'
import { sum } from './tableGeometry'
import type { PaperPosition } from './paperGeometry'

export function useTableClipboard(
  props: { active?: boolean; disabled: boolean; document: Document | null },
  input: () => HTMLElement | undefined,
  location: () => Pick<PaperPosition, 'page'> & Partial<PaperPosition>,
  add: (table: TableAsset) => void,
  notice: (message: string) => void,
) {
  function insert(options: TableInsertOptions) {
    if (props.disabled || !props.document) return
    try {
      add(createTable(options, props.document.settings, location()))
    } catch (e) {
      notice((e as Error).message)
    }
  }
  function paste(event: ClipboardEvent): boolean {
    if (
      event.defaultPrevented ||
      props.active === false ||
      props.disabled ||
      !props.document ||
      !event.clipboardData
    )
      return false
    const target = event.target as HTMLElement | null
    if (target?.closest('dialog,[role="dialog"]')) return false
    if (target?.closest('input,textarea,[contenteditable="true"]') && target !== input())
      return false
    const data = event.clipboardData,
      copied = readTable(data),
      text = data.getData('text/plain')
    if (!copied && !text.includes('\t') && !/<table\b/i.test(data.getData('text/html')))
      return false
    event.preventDefault()
    input()?.blur()
    try {
      const settings = props.document.settings,
        position = location()
      if (copied) {
        const t = cloneTable(copied)
        t.id = newTableId()
        t.page = position.page
        t.x = Math.max(
          0,
          Math.min(position.x ?? copied.x + 2, settings.paperWidth - sum(t.columnWidths)),
        )
        t.y = Math.max(
          0,
          Math.min(position.y ?? copied.y + 2, settings.paperHeight - sum(t.rowHeights)),
        )
        add(t)
      } else {
        const values = parseTableText(text),
          cols = Math.max(...values.map((r) => r.length)),
          rows = values.length
        const t = createTable(
          {
            rows,
            cols,
            width: Math.min(
              settings.paperWidth,
              Math.max(
                cols * 12,
                Math.min(120, settings.paperWidth - settings.marginLeft - settings.marginRight),
              ),
            ),
            style: 'grid',
          },
          settings,
          position,
        )
        add(pasteCells(t, 0, 0, values))
      }
    } catch (e) {
      notice((e as Error).message)
    }
    return true
  }
  return { insert, paste }
}
