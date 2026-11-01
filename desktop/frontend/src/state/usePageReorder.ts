import { onBeforeUnmount, onMounted, ref, watch, type Ref } from 'vue'

export function usePageReorder(
  rail: Ref<HTMLElement | undefined>,
  disabled: () => boolean,
  pages: () => unknown,
  move: (from: number, to: number) => void,
) {
  const source = ref<number | null>(null)
  const insertion = ref<number | null>(null)
  const preview = ref<{ x: number; y: number; number: number } | null>(null)
  let pending:
    | {
        id: number
        from: number
        x: number
        y: number
        originX: number
        originY: number
        button: HTMLElement
      }
    | undefined
  let frame = 0,
    lastTime = 0,
    suppressUntil = 0

  function updateTarget() {
    const root = rail.value,
      drag = pending
    if (!root || !drag) return
    const bounds = root.getBoundingClientRect()
    preview.value = { x: drag.x + 14, y: drag.y + 12, number: drag.from + 1 }
    if (
      drag.x < bounds.left - 20 ||
      drag.x > bounds.right + 20 ||
      drag.y < bounds.top - 24 ||
      drag.y > bounds.bottom + 24
    ) {
      insertion.value = null
      return
    }
    const buttons = [...root.querySelectorAll<HTMLButtonElement>('button')]
    let slot = buttons.findIndex((button) => {
      const rect = button.getBoundingClientRect()
      return drag.y < rect.top + rect.height / 2
    })
    if (slot < 0) slot = buttons.length
    const target = slot > drag.from ? slot - 1 : slot
    insertion.value = target === drag.from ? null : slot
  }
  function tick(time: number) {
    if (!pending || source.value === null || !rail.value) return
    const root = rail.value,
      bounds = root.getBoundingClientRect()
    const elapsed = Math.min(32, time - (lastTime || time))
    lastTime = time
    if (pending.x >= bounds.left - 20 && pending.x <= bounds.right + 20) {
      const edge = Math.min(28, bounds.height / 4)
      const speed = pending.y < bounds.top + edge ? -1 : pending.y > bounds.bottom - edge ? 1 : 0
      root.scrollTop += speed * elapsed * 0.5
    }
    updateTarget()
    frame = requestAnimationFrame(tick)
  }
  function cancel() {
    const drag = pending
    if (source.value !== null) suppressUntil = performance.now() + 500
    pending = undefined
    source.value = null
    insertion.value = null
    preview.value = null
    lastTime = 0
    cancelAnimationFrame(frame)
    if (drag?.button.hasPointerCapture(drag.id)) drag.button.releasePointerCapture(drag.id)
  }
  function pointerDown(event: PointerEvent, from: number) {
    if (
      event.button !== 0 ||
      pending ||
      disabled() ||
      !rail.value ||
      rail.value.querySelectorAll('button').length < 2
    )
      return
    const button = event.currentTarget as HTMLElement
    pending = {
      id: event.pointerId,
      from,
      x: event.clientX,
      y: event.clientY,
      originX: event.clientX,
      originY: event.clientY,
      button,
    }
    button.setPointerCapture(event.pointerId)
  }
  function pointerMove(event: PointerEvent) {
    if (!pending || pending.id !== event.pointerId) return
    pending.x = event.clientX
    pending.y = event.clientY
    if (source.value === null) {
      if (Math.hypot(pending.x - pending.originX, pending.y - pending.originY) < 5) return
      source.value = pending.from
      frame = requestAnimationFrame(tick)
    }
    event.preventDefault()
    updateTarget()
  }
  function pointerUp(event: PointerEvent) {
    if (!pending || event.pointerId !== pending.id) return
    const from = pending.from,
      slot = insertion.value
    const active = source.value !== null
    cancel()
    if (active && slot !== null && !disabled()) move(from, slot > from ? slot - 1 : slot)
  }
  function click(event: MouseEvent) {
    if (performance.now() >= suppressUntil) return false
    suppressUntil = 0
    event.preventDefault()
    event.stopPropagation()
    return true
  }
  function escape(event: KeyboardEvent) {
    if (event.key === 'Escape' && pending) {
      event.preventDefault()
      cancel()
    }
  }
  watch([disabled, pages], cancel)
  onMounted(() => {
    window.addEventListener('keydown', escape, true)
    window.addEventListener('blur', cancel)
  })
  onBeforeUnmount(() => {
    cancel()
    window.removeEventListener('keydown', escape, true)
    window.removeEventListener('blur', cancel)
  })
  return { source, insertion, preview, pointerDown, pointerMove, pointerUp, cancel, click }
}
