export const MIN_ZOOM = 30
export const MAX_ZOOM = 150
export function clampZoom(value: number) {
  return Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, value))
}
export function wheelZoom(current: number, delta: number, mode: number, viewportHeight: number) {
  const pixels = delta * (mode === 1 ? 16 : mode === 2 ? viewportHeight : 1)
  return clampZoom(current * Math.exp(-Math.max(-200, Math.min(200, pixels)) * 0.008))
}
