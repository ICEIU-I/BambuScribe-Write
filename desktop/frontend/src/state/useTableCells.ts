import {
  computed,
  nextTick,
  onBeforeUnmount,
  ref,
  watch,
  type ComponentPublicInstance,
  type Ref,
} from 'vue'
import type { TableAsset, TableCell, TableSelection } from './tableTypes'
import { cellAt, offsets, selectedCells, selectionBounds } from './tableGeometry'
import { cloneTable, mergeCells, pasteCells, setCellText, splitCells } from './tableOperations'
import { parseTableText, readTable, writeTable, tableText } from './tableClipboard'
import { flushEditorEvent } from './editorCommands'
interface CellEvents {
  select: (id: string) => void
  selection: (value: TableSelection | null) => void
  change: (table: TableAsset) => void
  remove: (id: string) => void
  settings: () => void
  notice: (message: string) => void
  undo: () => void
  redo: () => void
}
export function useTableCells(
  props: { asset: TableAsset; selected: boolean; disabled: boolean; interactive: boolean },
  table: () => TableAsset,
  host: Ref<HTMLElement | undefined>,
  scale: () => number,
  events: CellEvents,
) {
  const input = ref<HTMLTextAreaElement>(),
    selection = ref<TableSelection | null>(null),
    editing = ref<{ row: number; col: number; text: string } | null>(null)
  const selected = computed(() => (selection.value ? selectedCells(table(), selection.value) : []))
  const setInput = (el: Element | ComponentPublicInstance | null) => {
    input.value = el instanceof HTMLTextAreaElement ? el : undefined
  }
  function hitAxis(sizes: number[], point: number) {
    const i = offsets(sizes).findIndex((n, i) => i > 0 && n > point)
    return i < 0 ? sizes.length - 1 : Math.max(0, i - 1)
  }
  function openSettings() {
    focus()
    events.settings()
  }
  function focus() {
    host.value?.focus({ preventScroll: true })
    events.select(props.asset.id)
  }
  function range(value: TableSelection | null) {
    selection.value = value
    events.selection(value)
  }
  function startCell(e: PointerEvent, c: TableCell) {
    if (props.disabled || !props.interactive || e.button !== 0) return
    e.preventDefault()
    save()
    focus()
    const anchor =
      e.shiftKey && selection.value
        ? selection.value
        : { r0: c.row, c0: c.col, r1: c.row, c1: c.col }
    range(
      selectionBounds(table(), {
        r0: anchor.r0,
        c0: anchor.c0,
        r1: c.row + c.rowSpan - 1,
        c1: c.col + c.colSpan - 1,
      }),
    )
    const move = (event: PointerEvent) => {
      const rect = host.value?.getBoundingClientRect()
      if (!rect) return
      const x = (event.clientX - rect.left) / scale(),
        y = (event.clientY - rect.top) / scale(),
        r = hitAxis(table().rowHeights, y),
        col = hitAxis(table().columnWidths, x)
      range(selectionBounds(table(), { r0: anchor.r0, c0: anchor.c0, r1: r, c1: col }))
    }
    const end = () => {
      window.removeEventListener('pointermove', move)
      window.removeEventListener('pointerup', end)
      window.removeEventListener('pointercancel', end)
      stopSelecting = () => {}
    }
    stopSelecting()
    stopSelecting = end
    window.addEventListener('pointermove', move)
    window.addEventListener('pointerup', end)
    window.addEventListener('pointercancel', end)
  }
  let stopSelecting = () => {}
  async function edit(c: TableCell) {
    if (props.disabled) return
    save()
    focus()
    range({ r0: c.row, c0: c.col, r1: c.row + c.rowSpan - 1, c1: c.col + c.colSpan - 1 })
    editing.value = { row: c.row, col: c.col, text: c.text }
    await nextTick()
    input.value?.focus({ preventScroll: true })
    input.value?.select()
  }
  function save() {
    const value = editing.value
    if (!value) return
    editing.value = null
    const current = cellAt(props.asset, value.row, value.col)
    if (current && value.text !== current.text)
      events.change(setCellText(props.asset, value.row, value.col, value.text))
  }
  function moveCell(back = false) {
    const c = selection.value ? cellAt(table(), selection.value.r0, selection.value.c0) : undefined,
      cells = [...table().cells].sort((a, b) => a.row - b.row || a.col - b.col),
      i = c ? cells.indexOf(c) : -1,
      next = cells[(i + (back ? -1 : 1) + cells.length) % cells.length]
    if (next)
      void nextTick(() => {
        const fresh = cellAt(table(), next.row, next.col)
        if (fresh) void edit(fresh)
      })
  }
  function key(e: KeyboardEvent) {
    if (e.isComposing || props.disabled) return
    const k = e.key.toLowerCase()
    if (editing.value && (e.metaKey || e.ctrlKey) && (k === 's' || k === 'o')) {
      save()
      return
    }
    if (editing.value) {
      if (k === 'escape') {
        e.preventDefault()
        editing.value = null
        host.value?.focus()
      } else if (k === 'tab' || (k === 'enter' && !e.shiftKey)) {
        e.preventDefault()
        save()
        if (k === 'tab') moveCell(e.shiftKey)
        else host.value?.focus()
      }
      return
    }
    if ((e.metaKey || e.ctrlKey) && (k === 'z' || k === 'y')) {
      e.preventDefault()
      if (k === 'y' || e.shiftKey) events.redo()
      else events.undo()
      return
    }
    if ((e.metaKey || e.ctrlKey) && k === 'a') {
      e.preventDefault()
      range({
        r0: 0,
        c0: 0,
        r1: table().rowHeights.length - 1,
        c1: table().columnWidths.length - 1,
      })
      return
    }
    if (k === 'tab') {
      e.preventDefault()
      moveCell(e.shiftKey)
    } else if (k === 'enter' && selection.value) {
      e.preventDefault()
      const c = cellAt(table(), selection.value.r0, selection.value.c0)
      if (c) void edit(c)
    } else if (k === 'delete' || k === 'backspace') {
      e.preventDefault()
      if (!selection.value) events.remove(props.asset.id)
      else {
        const t = cloneTable(props.asset)
        for (const c of selectedCells(t, selection.value)) c.text = ''
        events.change(t)
      }
    } else if (k.startsWith('arrow')) {
      e.preventDefault()
      const b = selection.value ?? { r0: 0, c0: 0, r1: 0, c1: 0 },
        r = Math.max(
          0,
          Math.min(
            table().rowHeights.length - 1,
            k === 'arrowdown' ? b.r1 + 1 : k === 'arrowup' ? b.r0 - 1 : b.r0,
          ),
        ),
        col = Math.max(
          0,
          Math.min(
            table().columnWidths.length - 1,
            k === 'arrowright' ? b.c1 + 1 : k === 'arrowleft' ? b.c0 - 1 : b.c0,
          ),
        ),
        c = cellAt(table(), r, col)
      if (c)
        range(
          selectionBounds(table(), {
            r0: e.shiftKey ? b.r0 : c.row,
            c0: e.shiftKey ? b.c0 : c.col,
            r1: c.row + c.rowSpan - 1,
            c1: c.col + c.colSpan - 1,
          }),
        )
    }
  }
  function copy(e: ClipboardEvent, cut = false) {
    if (editing.value || !e.clipboardData || props.disabled) return
    e.preventDefault()
    e.stopPropagation()
    e.clipboardData.setData('text/plain', tableText(table(), selection.value ?? undefined))
    if (!selection.value) writeTable(e.clipboardData, props.asset)
    if (cut) {
      if (!selection.value) events.remove(props.asset.id)
      else {
        const t = cloneTable(props.asset)
        for (const c of selectedCells(t, selection.value)) c.text = ''
        events.change(t)
      }
    }
  }
  function paste(e: ClipboardEvent) {
    const data = e.clipboardData
    if (!data || props.disabled) return
    if (readTable(data) && !editing.value && !selection.value) return
    const text = data.getData('text/plain')
    if (editing.value && !text.includes('\t')) return
    e.preventDefault()
    e.stopPropagation()
    save()
    const b = selection.value ?? { r0: 0, c0: 0, r1: 0, c1: 0 }
    try {
      const t = pasteCells(props.asset, b.r0, b.c0, parseTableText(text))
      events.change(t)
      host.value?.focus()
    } catch (error) {
      events.notice(String((error as Error).message))
    }
  }
  function merge() {
    if (selection.value) events.change(mergeCells(props.asset, selection.value))
  }
  function split() {
    if (selection.value) events.change(splitCells(props.asset, selection.value))
  }
  watch(
    () => props.selected,
    (selected) => {
      if (!selected) {
        save()
        range(null)
      }
    },
  )
  watch(
    () => [props.asset.rowHeights.length, props.asset.columnWidths.length],
    () => {
      if (selection.value) range(selectionBounds(props.asset, selection.value))
    },
  )
  window.addEventListener(flushEditorEvent, save)
  onBeforeUnmount(() => {
    stopSelecting()
    window.removeEventListener(flushEditorEvent, save)
  })

  return {
    selection,
    editing,
    selected,
    setInput,
    openSettings,
    focus,
    range,
    startCell,
    save,
    edit,
    key,
    copy,
    paste,
    merge,
    split,
  }
}
