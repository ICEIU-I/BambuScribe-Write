import {
  computed,
  inject,
  nextTick,
  onBeforeUnmount,
  provide,
  shallowRef,
  watch,
  type InjectionKey,
  type Ref,
} from 'vue'
import type { Document, ImageAsset, ImageEditTool, Point, TableAsset } from '../types'
import type { ObjectSelection } from './objectSelection'
import {
  clampSelectionMove,
  selectedGeometry,
  translateObjects,
  translateSelection,
} from './objectSelectionMove'
import { nearestPaper } from './paperGeometry'

interface SelectionPreview {
  ids: Set<string>
  page: number
  delta: Point
  offset: Point
}
const selectionDragKey: InjectionKey<Ref<SelectionPreview | null>> = Symbol('selection-drag')
export function useSelectedObjectDrag(id: () => string) {
  const preview = inject(selectionDragKey, undefined)
  return computed(() => (preview?.value?.ids.has(id()) ? preview.value : null))
}
interface Context {
  document: Document | null
  modelValue: string
  images: ImageAsset[]
  tables: TableAsset[]
  tool: ImageEditTool
  active: boolean
  disabled: boolean
  zoom: number
}
export function useSelectionDrag(
  props: Context,
  stack: Ref<HTMLElement | undefined>,
  viewport: Ref<HTMLElement | undefined>,
  selection: Ref<ObjectSelection | null>,
  commit: (images: ImageAsset[], tables: TableAsset[]) => void,
  dropped: (page: number) => void,
) {
  const preview = shallowRef<SelectionPreview | null>(null)
  provide(selectionDragKey, preview)
  let gesture:
    | {
        base: NonNullable<ReturnType<typeof selectedGeometry>>
        paper: HTMLElement
        start: Point
        grab: Point
        point: Point
        pointer: number
        moved: boolean
      }
    | undefined
  let frame = 0
  function update() {
    const g = gesture,
      doc = props.document,
      root = stack.value
    if (!g?.moved || !doc || !root) return
    const target = nearestPaper(root, g.point.y) ?? g.paper,
      rect = target.getBoundingClientRect(),
      origin = g.paper.getBoundingClientRect(),
      k = rect.width / doc.settings.paperWidth
    const delta = clampSelectionMove(
      g.base.bounds,
      {
        x: (g.point.x - rect.left) / k - g.grab.x,
        y: (g.point.y - rect.top) / k - g.grab.y,
      },
      doc.settings,
    )
    preview.value = {
      ids: new Set(g.base.selection.objects.map((o) => o.id)),
      page: Number(target.dataset.paper),
      delta,
      offset: { x: rect.left - origin.left, y: rect.top - origin.top },
    }
  }
  function autoScroll() {
    const g = gesture,
      view = viewport.value
    if (!g?.moved || !view) return
    const r = view.getBoundingClientRect(),
      speed =
        g.point.y < r.top + 40
          ? -Math.min(16, (r.top + 40 - g.point.y) / 3)
          : g.point.y > r.bottom - 40
            ? Math.min(16, (g.point.y - r.bottom + 40) / 3)
            : 0
    if (speed) {
      view.scrollTop += speed
      update()
    }
    frame = requestAnimationFrame(autoScroll)
  }
  function cancel() {
    const g = gesture
    gesture = undefined
    preview.value = null
    cancelAnimationFrame(frame)
    if (g && stack.value?.hasPointerCapture(g.pointer)) stack.value.releasePointerCapture(g.pointer)
    window.removeEventListener('pointermove', move)
    window.removeEventListener('pointerup', finish)
    window.removeEventListener('pointercancel', cancel)
    window.removeEventListener('scroll', update, true)
  }
  function move(event: PointerEvent) {
    const g = gesture
    if (!g || g.pointer !== event.pointerId) return
    g.point = { x: event.clientX, y: event.clientY }
    if (!g.moved && Math.hypot(g.point.x - g.start.x, g.point.y - g.start.y) < 3) return
    if (!g.moved) {
      g.moved = true
      frame = requestAnimationFrame(autoScroll)
    }
    event.preventDefault()
    update()
  }
  async function finish(event: PointerEvent) {
    const g = gesture
    if (!g || g.pointer !== event.pointerId) return
    move(event)
    const p = preview.value
    cancel()
    if (
      !p ||
      (p.page === g.base.selection.page && Math.abs(p.delta.x) < 1e-8 && Math.abs(p.delta.y) < 1e-8)
    )
      return
    commit(
      translateObjects(g.base.images, p.delta, p.page),
      translateObjects(g.base.tables, p.delta, p.page),
    )
    // Prop updates clear stale selections. Restore only this completed move;
    // subsequent native layout refreshes resolve the selected IDs again.
    await nextTick()
    selection.value = translateSelection(g.base.selection, p.delta, p.page)
    dropped(p.page)
  }
  function begin(event: PointerEvent) {
    const doc = props.document,
      root = stack.value,
      selected = selection.value,
      target = event.target as HTMLElement | null
    if (
      !doc ||
      !root ||
      !selected?.objects.length ||
      props.disabled ||
      !props.active ||
      props.tool !== 'select' ||
      event.button !== 0 ||
      !target?.closest('.paper-marquee-layer')
    )
      return false
    const base = selectedGeometry(doc, props.images, props.tables, selected),
      paper = root.querySelector<HTMLElement>(`[data-paper="${selected.page}"]`)
    if (!base || !paper) return false
    event.preventDefault()
    event.stopPropagation()
    viewport.value?.focus({ preventScroll: true })
    const r = paper.getBoundingClientRect(),
      k = r.width / doc.settings.paperWidth,
      start = { x: event.clientX, y: event.clientY }
    gesture = {
      base,
      paper,
      start,
      point: start,
      pointer: event.pointerId,
      moved: false,
      grab: { x: (start.x - r.left) / k, y: (start.y - r.top) / k },
    }
    root.setPointerCapture(event.pointerId)
    window.addEventListener('pointermove', move)
    window.addEventListener('pointerup', finish)
    window.addEventListener('pointercancel', cancel)
    window.addEventListener('scroll', update, true)
    return true
  }
  watch(
    () => [
      props.document,
      props.modelValue,
      props.images,
      props.tables,
      props.tool,
      props.active,
      props.disabled,
      props.zoom,
    ],
    cancel,
  )
  watch(selection, (value) => {
    if (!value) cancel()
  })
  onBeforeUnmount(cancel)
  return {
    begin,
    cancel,
    lostCapture: () => {
      if (gesture) cancel()
    },
    selection: computed(() => {
      const p = preview.value,
        base = gesture?.base.selection
      if (!p || !base) return selection.value
      const k = ((96 / 25.4) * props.zoom) / 100
      return translateSelection(
        base,
        { x: p.delta.x + p.offset.x / k, y: p.delta.y + p.offset.y / k },
        base.page,
      )
    }),
  }
}
