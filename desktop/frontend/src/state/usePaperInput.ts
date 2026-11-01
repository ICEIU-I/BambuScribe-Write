import { formulaKey, formulaSelection } from './formulaSelection'
import { computed, nextTick, onMounted, ref, watch, type Ref } from 'vue'
import type { Document } from '../types'
import { caretAt, hitCaret, rowsFor, type LocatedCaret } from './caretGeometry'
import { blankRows } from './blankCaret'

export function usePaperInput(
  input: Ref<HTMLTextAreaElement | undefined>,
  getText: () => string,
  getDocument: () => Document | null,
  update: (text: string) => void,
) {
  const start = ref(0),
    end = ref(0),
    backward = ref(false),
    focused = ref(false)
  const composition = ref(''),
    preferred = ref<LocatedCaret | null>(null)
  const pendingCaret = ref<{ documentId: string; stop: LocatedCaret } | null>(null)
  const emptyRows = computed(() => blankRows(getDocument(), getText().includes('\f')))
  const caret = computed(() => {
    const index = backward.value ? start.value : end.value
    if (preferred.value?.blankLines !== undefined && emptyRows.value.length)
      return (
        emptyRows.value.find(
          (row) =>
            row.page === preferred.value?.page &&
            row.stops[0].index === index &&
            row.stops[0].blankLines === preferred.value?.blankLines,
        )?.stops[0] ?? caretAt(getDocument(), index)
      )
    const pending = pendingCaret.value
    if (pending && pending.documentId === getDocument()?.id && pending.stop.index === index)
      return pending.stop
    return caretAt(getDocument(), index, preferred.value)
  })
  const rows = computed(() => {
    const blank = new Set(emptyRows.value.map((r) => `${r.page}:${r.y}`))
    return [
      ...rowsFor(getDocument()).filter((r) => !blank.has(`${r.page}:${r.y}`)),
      ...emptyRows.value,
    ].sort((a, b) => a.page - b.page || a.y - b.y)
  })
  let composing = false,
    anchor = 0,
    dragging = false,
    desiredX: number | null = null

  function readSelection() {
    const el = input.value
    if (!el || composing) return
    const normalized = formulaSelection(getText(), el.selectionStart, el.selectionEnd)
    if (normalized.start !== el.selectionStart || normalized.end !== el.selectionEnd)
      el.setSelectionRange(normalized.start, normalized.end, el.selectionDirection)
    start.value = normalized.start
    end.value = normalized.end
    backward.value = el.selectionDirection === 'backward'
  }
  function changed() {
    if (composing || !input.value) return
    const el = input.value
    if (getText() === el.value) return
    const target = preferred.value
    const padding = target?.blankLines ?? 0
    pendingCaret.value = null
    if (
      target?.blankLines !== undefined &&
      start.value === end.value &&
      start.value === target.index
    ) {
      const [from, to] = [el.selectionStart, el.selectionEnd]
      el.value =
        el.value.slice(0, target.index) + '\n'.repeat(padding) + el.value.slice(target.index)
      el.setSelectionRange(from + padding, to + padding)
      pendingCaret.value = {
        documentId: getDocument()!.id,
        stop: { ...target, index: to + padding },
      }
    }
    preferred.value = null
    desiredX = null
    update(input.value.value)
    readSelection()
  }
  function select(from: number, to: number, affinity: LocatedCaret | null = null) {
    const el = input.value
    if (!el) return
    pendingCaret.value = null
    preferred.value = affinity
    el.focus({ preventScroll: true })
    el.setSelectionRange(Math.min(from, to), Math.max(from, to), to < from ? 'backward' : 'forward')
    readSelection()
  }
  function locate(event: PointerEvent, root: HTMLElement): LocatedCaret | null {
    const doc = getDocument()
    if (!doc) return null
    const elements = [...root.querySelectorAll<HTMLElement>('[data-paper]')]
    let paper = elements.find((el) => {
      const r = el.getBoundingClientRect()
      return event.clientY >= r.top && event.clientY <= r.bottom
    })
    if (!paper)
      paper = elements.reduce<HTMLElement | undefined>((nearest, el) => {
        const distance = Math.abs(el.getBoundingClientRect().top - event.clientY)
        return !nearest || distance < Math.abs(nearest.getBoundingClientRect().top - event.clientY)
          ? el
          : nearest
      }, undefined)
    if (!paper) return null
    const rect = paper.getBoundingClientRect(),
      scale = rect.width / doc.settings.paperWidth
    return hitCaret(
      doc,
      Number(paper.dataset.paper),
      (event.clientX - rect.left) / scale,
      (event.clientY - rect.top) / scale,
      rows.value,
    )
  }
  function pointerDown(event: PointerEvent, root: HTMLElement) {
    if (event.button !== 0 || composing) return
    const target = locate(event, root)
    if (!target) return
    event.preventDefault()
    desiredX = null
    anchor = event.shiftKey ? (backward.value ? end.value : start.value) : target.index
    dragging = true
    root.setPointerCapture(event.pointerId)
    select(anchor, target.index, target)
  }
  function placeCaret(event: PointerEvent, root: HTMLElement) {
    if (composing) return
    desiredX = null
    const target = locate(event, root)
    if (target) select(target.index, target.index, target)
  }
  function pointerMove(event: PointerEvent, root: HTMLElement) {
    if (!dragging) return
    const target = locate(event, root)
    if (target) select(anchor, target.index, target)
  }
  function pointerUp() {
    dragging = false
  }
  function keydown(event: KeyboardEvent) {
    if (composing || event.isComposing) return
    let key = event.key
    const mathKey = formulaKey(getText(), start.value, end.value, key)
    if (mathKey && !event.metaKey && !event.ctrlKey && !event.altKey) {
      event.preventDefault()
      if (mathKey.remove && input.value) {
        input.value.value = getText().slice(0, mathKey.start) + getText().slice(mathKey.end)
        input.value.setSelectionRange(mathKey.start, mathKey.start)
        changed()
      } else
        select(
          event.shiftKey ? (backward.value ? end.value : start.value) : mathKey.start,
          mathKey.end,
        )
      return
    }
    if (
      preferred.value?.blankLines &&
      start.value === end.value &&
      !event.metaKey &&
      !event.ctrlKey &&
      !event.altKey
    ) {
      if (key === 'Delete') {
        event.preventDefault()
        return
      }
      if (key === 'Backspace') key = 'ArrowUp'
    }
    if (!['ArrowUp', 'ArrowDown', 'Home', 'End'].includes(key) || event.metaKey || event.ctrlKey) {
      desiredX = null
      // Typing and IME key events must keep the chosen empty row until input.
      if (
        preferred.value?.blankLines === undefined ||
        ['ArrowLeft', 'ArrowRight', 'Backspace', 'Delete'].includes(key) ||
        ((event.metaKey || event.ctrlKey) && key.toLowerCase() === 'a')
      )
        preferred.value = null
      return
    }
    event.preventDefault()
    const current = caret.value,
      all = rows.value
    const rowIndex = all.findIndex((row) => row.page === current.page && row.y === current.y)
    if (rowIndex < 0) return
    let target: LocatedCaret
    if (key === 'Home' || key === 'End') {
      const stops = all[rowIndex].stops
      target = key === 'Home' ? stops[0] : stops[stops.length - 1]
      desiredX = null
    } else {
      desiredX ??= current.x
      const next =
        all[Math.max(0, Math.min(all.length - 1, rowIndex + (key === 'ArrowUp' ? -1 : 1)))]
      target = next.stops.reduce((a, b) =>
        Math.abs(a.x - desiredX!) <= Math.abs(b.x - desiredX!) ? a : b,
      )
    }
    select(
      event.shiftKey ? (backward.value ? end.value : start.value) : target.index,
      target.index,
      target,
    )
  }
  function compositionStart() {
    composing = true
    composition.value = ''
  }
  function compositionUpdate(event: CompositionEvent) {
    composition.value = event.data
  }
  function compositionEnd() {
    composing = false
    composition.value = ''
    changed()
  }
  function command(name: 'undo' | 'redo') {
    input.value?.focus({ preventScroll: true })
    window.document.execCommand(name)
    changed()
  }
  function focus() {
    input.value?.focus({ preventScroll: true })
    readSelection()
  }
  watch(getText, (value) => {
    if (value !== '') preferred.value = null
    const el = input.value
    if (el && !composing && el.value !== value) {
      const [a, b] = [el.selectionStart, el.selectionEnd]
      el.value = value
      el.setSelectionRange(Math.min(a, value.length), Math.min(b, value.length))
      readSelection()
    }
  })
  onMounted(async () => {
    await nextTick()
    if (input.value) input.value.value = getText()
  })
  return {
    start,
    end,
    caret,
    focused,
    composition,
    changed,
    readSelection,
    select,
    pointerDown,
    placeCaret,
    pointerMove,
    pointerUp,
    keydown,
    compositionStart,
    compositionUpdate,
    compositionEnd,
    command,
    focus,
  }
}
