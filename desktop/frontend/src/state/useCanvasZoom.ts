import { nextTick, onBeforeUnmount, onMounted, watch, type Ref } from 'vue'
import { clampZoom, wheelZoom } from './canvasZoom'

type GestureEvent = Event & { scale: number; clientX?: number; clientY?: number }
type Anchor = { page: HTMLElement; x: number; y: number; clientX: number; clientY: number }

export function useCanvasZoom(
  viewport: Ref<HTMLElement | undefined>,
  getZoom: () => number,
  update: (zoom: number) => void,
) {
  let targetZoom = getZoom(),
    gestureBase: number | null = null,
    frame = 0
  let anchor: Anchor | null = null
  function capture(clientX?: number, clientY?: number): Anchor | null {
    const view = viewport.value
    if (!view) return null
    const box = view.getBoundingClientRect()
    const x = clientX ?? box.left + box.width / 2,
      y = clientY ?? box.top + box.height / 2
    const pages = [...view.querySelectorAll<HTMLElement>('[data-paper]')]
    const page = pages.reduce<HTMLElement | null>((nearest, el) => {
      const distance = (p: HTMLElement) => {
        const r = p.getBoundingClientRect()
        return y < r.top ? r.top - y : y > r.bottom ? y - r.bottom : 0
      }
      return !nearest || distance(el) < distance(nearest) ? el : nearest
    }, null)
    if (!page) return null
    const r = page.getBoundingClientRect()
    return { page, x: (x - r.left) / r.width, y: (y - r.top) / r.height, clientX: x, clientY: y }
  }
  function queue(zoom: number, clientX?: number, clientY?: number) {
    targetZoom = clampZoom(zoom)
    if (frame) return
    anchor = capture(clientX, clientY)
    frame = requestAnimationFrame(() => {
      frame = 0
      update(Math.round(targetZoom * 1000) / 1000)
    })
  }
  function wheel(event: WheelEvent) {
    if (!event.ctrlKey && !event.metaKey) return
    event.preventDefault()
    // WebKit can emit wheel events during its native gesture sequence as well.
    if (gestureBase !== null) return
    queue(
      wheelZoom(targetZoom, event.deltaY, event.deltaMode, viewport.value?.clientHeight ?? 800),
      event.clientX,
      event.clientY,
    )
  }
  function gestureStart(event: Event) {
    event.preventDefault()
    gestureBase = targetZoom
  }
  function gestureChange(event: Event) {
    const gesture = event as GestureEvent
    if (gestureBase === null || !Number.isFinite(gesture.scale) || gesture.scale <= 0) return
    event.preventDefault()
    queue(gestureBase * gesture.scale, gesture.clientX, gesture.clientY)
  }
  function gestureEnd(event: Event) {
    event.preventDefault()
    gestureBase = null
  }
  watch(
    getZoom,
    async (zoom) => {
      const point = anchor ?? capture()
      anchor = null
      if (!frame) targetZoom = zoom
      await nextTick()
      if (!point || !viewport.value || !point.page.isConnected) return
      const r = point.page.getBoundingClientRect()
      viewport.value.scrollLeft += r.left + point.x * r.width - point.clientX
      viewport.value.scrollTop += r.top + point.y * r.height - point.clientY
    },
    { flush: 'pre' },
  )
  onMounted(() => {
    viewport.value?.addEventListener('wheel', wheel, { passive: false })
    viewport.value?.addEventListener('gesturestart', gestureStart, { passive: false })
    viewport.value?.addEventListener('gesturechange', gestureChange, { passive: false })
    viewport.value?.addEventListener('gestureend', gestureEnd, { passive: false })
  })
  onBeforeUnmount(() => {
    cancelAnimationFrame(frame)
    viewport.value?.removeEventListener('wheel', wheel)
    viewport.value?.removeEventListener('gesturestart', gestureStart)
    viewport.value?.removeEventListener('gesturechange', gestureChange)
    viewport.value?.removeEventListener('gestureend', gestureEnd)
  })
}
