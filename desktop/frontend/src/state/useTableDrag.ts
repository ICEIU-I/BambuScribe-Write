import { computed, inject, onBeforeUnmount, ref, type Ref } from 'vue'
import type { LayoutSettings, Point } from '../types'
import type { TableAsset, TablePlacement } from './tableTypes'
import { cloneTable, resizeTable } from './tableOperations'
import { sum } from './tableGeometry'
import { nearestPaper } from './paperGeometry'
import { importedTableDragKey } from './useImportedTableDrag'
import { fitTableFont } from './tableFontFit'
import { useSelectedObjectDrag } from './useSelectionDrag'

export function useTableDrag(
  host: Ref<HTMLElement | undefined>,
  asset: () => TableAsset,
  table: () => TableAsset,
  settings: () => LayoutSettings,
  scale: () => number,
  change: (t: TableAsset) => void,
  preview: () => TablePlacement | undefined,
) {
  const draft = ref<TableAsset | null>(null),
    offset = ref<Point>({ x: 0, y: 0 })
  const group = inject(importedTableDragKey, undefined)
  const selectedDrag = useSelectedObjectDrag(() => asset().id)
  const groupPreview = computed(() => {
    const p = group?.preview.value
    return p?.ids.has(asset().id) ? p : null
  })
  let gesture:
    | {
        kind: 'move' | 'size' | 'row' | 'col'
        wholeGroup: boolean
        index: number
        base: TableAsset
        source: TableAsset
        start: Point
        point: Point
        grab: Point
        origin?: HTMLElement
        stack?: HTMLElement
        viewport?: HTMLElement
        target: HTMLElement
        pointer: number
        moved: boolean
      }
    | undefined
  let frame = 0
  const clamp = (v: number, max: number) => Math.max(0, Math.min(max, v))
  function update() {
    const g = gesture
    if (!g?.moved) return
    const s = settings(),
      origin = g.origin?.getBoundingClientRect(),
      dest = g.kind === 'move' && g.stack ? nearestPaper(g.stack, g.point.y) : g.origin,
      rect = dest?.getBoundingClientRect(),
      k = rect ? rect.width / s.paperWidth : scale(),
      base = g.base
    const dx = (g.point.x - g.start.x) / k,
      dy = (g.point.y - g.start.y) / k
    let next = cloneTable(base)
    if (g.kind === 'move') {
      next.page = dest ? Number(dest.dataset.paper) : base.page
      next.x = clamp(
        rect ? (g.point.x - rect.left) / k - g.grab.x : base.x + dx,
        s.paperWidth - sum(base.columnWidths),
      )
      next.y = clamp(
        rect ? (g.point.y - rect.top) / k - g.grab.y : base.y + dy,
        s.paperHeight - sum(base.rowHeights),
      )
      offset.value =
        origin && rect ? { x: rect.left - origin.left, y: rect.top - origin.top } : { x: 0, y: 0 }
      if (g.wholeGroup) next = group?.move(next, offset.value) ?? next
    } else if (g.kind === 'size') {
      const w = Math.max(
          base.columnWidths.length * 3,
          Math.min(s.paperWidth - base.x, sum(base.columnWidths) + dx),
        ),
        h = Math.max(
          base.rowHeights.length * 3,
          Math.min(s.paperHeight - base.y, sum(base.rowHeights) + dy),
        )
      // Preserve at least 3 mm for each independently sized row and column.
      const minW = sum(base.columnWidths) * Math.max(...base.columnWidths.map((v) => 3 / v)),
        minH = sum(base.rowHeights) * Math.max(...base.rowHeights.map((v) => 3 / v))
      next = resizeTable(base, Math.max(minW, w), Math.max(minH, h))
    } else {
      const axis = g.kind === 'row' ? next.rowHeights : next.columnWidths,
        before = axis[g.index - 1]!,
        after = axis[g.index]!,
        delta = Math.max(3 - before, Math.min(after - 3, g.kind === 'row' ? dy : dx))
      axis[g.index - 1] = before + delta
      axis[g.index] = after - delta
    }
    draft.value = g.kind === 'move' ? next : fitTableFont(next, preview())
  }
  function autoScroll() {
    const g = gesture
    if (!g?.moved || g.kind !== 'move' || !g.viewport) return
    const rect = g.viewport.getBoundingClientRect(),
      speed =
        g.point.y < rect.top + 48
          ? -Math.min(18, (rect.top + 48 - g.point.y) / 3)
          : g.point.y > rect.bottom - 48
            ? Math.min(18, (g.point.y - rect.bottom + 48) / 3)
            : 0
    if (speed) {
      g.viewport.scrollTop += speed
      update()
    }
    frame = requestAnimationFrame(autoScroll)
  }
  function move(e: PointerEvent) {
    const g = gesture
    if (!g || g.pointer !== e.pointerId) return
    g.point = { x: e.clientX, y: e.clientY }
    if (!g.moved && Math.hypot(g.point.x - g.start.x, g.point.y - g.start.y) < 3) return
    if (!g.moved) {
      g.moved = true
      g.target.setPointerCapture(g.pointer)
      frame = requestAnimationFrame(autoScroll)
    }
    e.preventDefault()
    update()
  }
  function cancel() {
    const g = gesture
    gesture = undefined
    group?.clear(asset().id)
    cancelAnimationFrame(frame)
    if (g?.target.hasPointerCapture(g.pointer)) g.target.releasePointerCapture(g.pointer)
    window.removeEventListener('pointermove', move)
    window.removeEventListener('pointerup', finish)
    window.removeEventListener('pointercancel', cancel)
    window.removeEventListener('scroll', update, true)
    window.removeEventListener('keydown', key)
    draft.value = null
    offset.value = { x: 0, y: 0 }
  }
  function finish(e: PointerEvent) {
    if (!gesture || e.pointerId !== gesture.pointer) return
    gesture.point = { x: e.clientX, y: e.clientY }
    update()
    const g = gesture,
      next = draft.value ? cloneTable(draft.value) : null
    if (next) {
      // Automatic text growth is measured geometry, not a new minimum height.
      // Only an explicit height resize may store those displayed dimensions.
      next.rowHeights = next.rowHeights.map((height, i) =>
        g.kind === 'size' || (g.kind === 'row' && (i === g.index - 1 || i === g.index))
          ? height
          : g.source.rowHeights[i]!,
      )
    }
    cancel()
    if (next) {
      if (g.wholeGroup && group) group.commit(next)
      else change(next)
    }
  }
  function key(e: KeyboardEvent) {
    if (e.key === 'Escape') cancel()
  }
  function begin(
    e: PointerEvent,
    kind: 'move' | 'size' | 'row' | 'col',
    index = 0,
    wholeGroup = false,
  ) {
    cancel()
    if (e.button !== 0) return
    const base = cloneTable(table()),
      start = { x: e.clientX, y: e.clientY },
      origin = host.value?.closest<HTMLElement>('[data-paper]') ?? undefined,
      rect = origin?.getBoundingClientRect(),
      k = rect ? rect.width / settings().paperWidth : scale()
    gesture = {
      kind,
      wholeGroup: kind === 'move' && wholeGroup,
      index,
      base,
      source: cloneTable(asset()),
      start,
      point: start,
      grab: rect
        ? { x: (start.x - rect.left) / k - base.x, y: (start.y - rect.top) / k - base.y }
        : { x: 0, y: 0 },
      origin,
      stack: host.value?.closest<HTMLElement>('.document-stack') ?? undefined,
      viewport: host.value?.closest<HTMLElement>('.document-viewport') ?? undefined,
      target: e.currentTarget as HTMLElement,
      pointer: e.pointerId,
      moved: false,
    }
    window.addEventListener('pointermove', move)
    window.addEventListener('pointerup', finish)
    window.addEventListener('pointercancel', cancel)
    window.addEventListener('scroll', update, true)
    window.addEventListener('keydown', key)
  }
  onBeforeUnmount(cancel)
  return {
    draft: computed(() => {
      const selected = selectedDrag.value
      if (selected) {
        const base = table()
        return {
          ...base,
          page: selected.page,
          x: base.x + selected.delta.x,
          y: base.y + selected.delta.y,
        }
      }
      if (draft.value) return draft.value
      const p = groupPreview.value,
        base = table()
      return p ? { ...base, page: p.page, x: base.x + p.dx, y: base.y + p.dy } : null
    }),
    offset: computed(
      () => selectedDrag.value?.offset ?? groupPreview.value?.offset ?? offset.value,
    ),
    begin,
  }
}
