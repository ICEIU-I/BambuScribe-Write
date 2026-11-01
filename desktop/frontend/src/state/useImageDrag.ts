import { onBeforeUnmount, ref, type Ref } from 'vue'
import type { ImageAsset, LayoutSettings, Point } from '../types'
import { nearestPaper } from './paperGeometry'

const clamp = (value: number, max: number) => Math.max(0, Math.min(max, value))

// Keep the overlay in its original component until pointerup so crossing a
// page never loses pointer capture. The draft uses destination-page geometry.
export function useImageDrag(
  host: Ref<HTMLElement | undefined>,
  asset: () => ImageAsset,
  settings: () => LayoutSettings,
  scale: () => number,
  change: (image: ImageAsset) => void,
) {
  const draft = ref<ImageAsset | null>(null),
    offset = ref<Point>({ x: 0, y: 0 })
  let gesture:
    | {
        kind: 'move' | 'size'
        image: ImageAsset
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
  let scrollFrame = 0

  function update() {
    const g = gesture
    if (!g?.moved) return
    const f = g.image,
      s = settings()
    const origin = g.origin?.getBoundingClientRect()
    const destination = g.kind === 'move' && g.stack ? nearestPaper(g.stack, g.point.y) : g.origin
    const rect = destination?.getBoundingClientRect()
    const k = rect ? rect.width / s.paperWidth : scale()
    const x = rect ? (g.point.x - rect.left) / k - g.grab.x : f.x + (g.point.x - g.start.x) / k
    const y = rect ? (g.point.y - rect.top) / k - g.grab.y : f.y + (g.point.y - g.start.y) / k
    if (g.kind === 'move') {
      draft.value = {
        ...f,
        page: destination ? Number(destination.dataset.paper) : f.page,
        x: clamp(x, s.paperWidth - f.width),
        y: clamp(y, s.paperHeight - f.height),
      }
      offset.value =
        origin && rect ? { x: rect.left - origin.left, y: rect.top - origin.top } : { x: 0, y: 0 }
    } else {
      if (f.diagram) {
        const ratio = f.diagram.height / f.diagram.width
        const width = Math.max(
          Math.max(2, 2 / ratio),
          Math.min(
            s.paperWidth - f.x,
            (s.paperHeight - f.y) / ratio,
            Math.max(f.width + x - f.x, (f.height + y - f.y) / ratio),
          ),
        )
        draft.value = { ...f, width, height: width * ratio }
        return
      }
      draft.value = {
        ...f,
        width: Math.max(2, Math.min(s.paperWidth - f.x, f.width + x - f.x)),
        height: Math.max(2, Math.min(s.paperHeight - f.y, f.height + y - f.y)),
      }
    }
  }

  function autoScroll() {
    const g = gesture
    if (!g?.moved || g.kind !== 'move' || !g.viewport) return
    const view = g.viewport,
      r = view.getBoundingClientRect()
    const speed =
      g.point.y < r.top + 48
        ? -Math.min(18, (r.top + 48 - g.point.y) / 3)
        : g.point.y > r.bottom - 48
          ? Math.min(18, (g.point.y - r.bottom + 48) / 3)
          : 0
    if (speed) {
      view.scrollTop += speed
      update()
    }
    scrollFrame = requestAnimationFrame(autoScroll)
  }

  function move(event: PointerEvent) {
    const g = gesture
    if (!g || event.pointerId !== g.pointer) return
    g.point = { x: event.clientX, y: event.clientY }
    if (!g.moved && Math.hypot(g.point.x - g.start.x, g.point.y - g.start.y) < 3) return
    if (!g.moved) {
      g.moved = true
      g.target.setPointerCapture(g.pointer)
      scrollFrame = requestAnimationFrame(autoScroll)
    }
    event.preventDefault()
    update()
  }

  function cleanup() {
    const g = gesture
    gesture = undefined
    cancelAnimationFrame(scrollFrame)
    if (g?.target.hasPointerCapture(g.pointer)) g.target.releasePointerCapture(g.pointer)
    window.removeEventListener('pointermove', move)
    window.removeEventListener('pointerup', finish)
    window.removeEventListener('pointercancel', cleanup)
    window.removeEventListener('scroll', update, true)
    window.removeEventListener('keydown', keydown)
    draft.value = null
    offset.value = { x: 0, y: 0 }
  }
  function finish(event: PointerEvent) {
    if (!gesture || event.pointerId !== gesture.pointer) return
    gesture.point = { x: event.clientX, y: event.clientY }
    update()
    const image = draft.value
    cleanup()
    if (image) change(image)
  }
  function keydown(event: KeyboardEvent) {
    if (event.key === 'Escape') cleanup()
  }
  function begin(event: PointerEvent, kind: 'move' | 'size') {
    cleanup()
    const image = { ...asset() },
      start = { x: event.clientX, y: event.clientY }
    const origin = host.value?.closest<HTMLElement>('[data-paper]') ?? undefined
    const rect = origin?.getBoundingClientRect(),
      k = rect ? rect.width / settings().paperWidth : scale()
    gesture = {
      kind,
      image,
      start,
      point: start,
      grab: rect
        ? { x: (start.x - rect.left) / k - image.x, y: (start.y - rect.top) / k - image.y }
        : { x: 0, y: 0 },
      origin,
      stack: host.value?.closest<HTMLElement>('.document-stack') ?? undefined,
      viewport: host.value?.closest<HTMLElement>('.document-viewport') ?? undefined,
      target: event.currentTarget as HTMLElement,
      pointer: event.pointerId,
      moved: false,
    }
    window.addEventListener('pointermove', move)
    window.addEventListener('pointerup', finish)
    window.addEventListener('pointercancel', cleanup)
    window.addEventListener('scroll', update, true)
    window.addEventListener('keydown', keydown)
  }
  onBeforeUnmount(cleanup)
  return { draft, offset, begin }
}
