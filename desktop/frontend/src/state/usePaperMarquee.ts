import { onBeforeUnmount, ref, watch, type Ref } from 'vue'
import type { Document, ImageAsset, ImageEditTool, Point, TableAsset } from '../types'
import { nearestPaper } from './paperGeometry'
import { onBodyText, selectionAt, type ObjectSelection } from './objectSelection'
import { selectedGeometry } from './objectSelectionMove'

interface Context {
  document: Document | null
  modelValue: string
  images: ImageAsset[]
  tables?: TableAsset[]
  active: boolean
  disabled: boolean
  tool: ImageEditTool
  zoom: number
}
export function usePaperMarquee(
  props: Context,
  stack: Ref<HTMLElement | undefined>,
  viewport: Ref<HTMLElement | undefined>,
  click: (event: PointerEvent) => void,
) {
  const selection = ref<ObjectSelection | null>(null)
  let gesture:
    | {
        paper: HTMLElement
        page: number
        start: Point
        screen: Point
        point: Point
        pointer: number
        moved: boolean
      }
    | undefined
  let frame = 0
  function locate(paper: HTMLElement, p: Point): Point {
    const r = paper.getBoundingClientRect(),
      s = props.document!.settings,
      k = r.width / s.paperWidth
    return {
      x: Math.max(0, Math.min(s.paperWidth, (p.x - r.left) / k)),
      y: Math.max(0, Math.min(s.paperHeight, (p.y - r.top) / k)),
    }
  }
  function update() {
    const g = gesture,
      doc = props.document
    if (!g?.moved || !doc) return
    selection.value = selectionAt(
      doc,
      props.images,
      props.tables ?? [],
      g.page,
      g.start,
      locate(g.paper, g.point),
    )
  }
  function autoScroll() {
    const g = gesture,
      view = viewport.value
    if (!g?.moved || !view) return
    const r = view.getBoundingClientRect()
    const speed =
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
  function cleanup() {
    const g = gesture
    gesture = undefined
    cancelAnimationFrame(frame)
    if (g && stack.value?.hasPointerCapture(g.pointer)) stack.value.releasePointerCapture(g.pointer)
    window.removeEventListener('pointermove', move)
    window.removeEventListener('pointerup', finish)
    window.removeEventListener('pointercancel', cancel)
    window.removeEventListener('scroll', update, true)
  }
  function clear() {
    cleanup()
    selection.value = null
  }
  function cancel() {
    clear()
  }
  function move(event: PointerEvent) {
    const g = gesture
    if (!g || event.pointerId !== g.pointer) return
    g.point = { x: event.clientX, y: event.clientY }
    if (!g.moved && Math.hypot(g.point.x - g.screen.x, g.point.y - g.screen.y) < 3) return
    if (!g.moved) {
      g.moved = true
      frame = requestAnimationFrame(autoScroll)
    }
    event.preventDefault()
    update()
  }
  function finish(event: PointerEvent) {
    const g = gesture
    if (!g || g.pointer !== event.pointerId) return
    move(event)
    cleanup()
    if (!g.moved) click(event)
  }
  function begin(event: PointerEvent) {
    clear()
    const doc = props.document,
      root = stack.value,
      target = event.target as HTMLElement | null
    if (
      !doc ||
      !root ||
      props.disabled ||
      !props.active ||
      props.tool !== 'select' ||
      event.button !== 0 ||
      event.shiftKey ||
      target?.closest(
        'input,textarea,button,[contenteditable="true"],.image-selection,.table-overlay,.free-text-box',
      )
    )
      return false
    const paper = nearestPaper(root, event.clientY)
    if (!paper) return false
    const screen = { x: event.clientX, y: event.clientY },
      start = locate(paper, screen),
      page = Number(paper.dataset.paper)
    if (onBodyText(doc, props.modelValue, page, start)) return false
    event.preventDefault()
    viewport.value?.focus({ preventScroll: true })
    gesture = { paper, page, start, screen, point: screen, pointer: event.pointerId, moved: false }
    root.setPointerCapture(event.pointerId)
    window.addEventListener('pointermove', move)
    window.addEventListener('pointerup', finish)
    window.addEventListener('pointercancel', cancel)
    window.addEventListener('scroll', update, true)
    return true
  }
  function lostCapture() {
    if (gesture) cancel()
  }
  watch(
    () => [
      props.modelValue,
      props.images,
      props.tables,
      props.active,
      props.disabled,
      props.tool,
      props.zoom,
    ],
    clear,
  )
  watch(
    () => props.document,
    (doc) => {
      if (gesture || !doc) {
        clear()
        return
      }
      if (selection.value?.objects.length)
        selection.value =
          selectedGeometry(doc, props.images, props.tables ?? [], selection.value)?.selection ??
          null
    },
  )
  onBeforeUnmount(clear)
  return { selection, begin, clear, lostCapture }
}
