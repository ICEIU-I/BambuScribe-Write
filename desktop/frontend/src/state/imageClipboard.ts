import type { ImageAsset, LayoutSettings } from '../types'
import type { PaperPosition } from './paperGeometry'

export const imageMime = 'application/x-bambuscribe-image'
export interface CopiedImage {
  version: 1
  token: string
  image: ImageAsset
}
const newId = () =>
  globalThis.crypto?.randomUUID?.() ?? `image-${Date.now()}-${Math.random().toString(16).slice(2)}`
const clone = <T>(value: T): T => JSON.parse(JSON.stringify(value))

export function copyImage(image: ImageAsset): CopiedImage {
  return { version: 1, token: `BambuScribe:image:${newId()}`, image: clone(image) }
}
export function writeImage(data: DataTransfer, copied: CopiedImage) {
  // The token identifies this exact copy when a WebView drops custom MIME data.
  data.setData('text/plain', copied.token)
  try {
    data.setData(imageMime, JSON.stringify(copied))
  } catch {
    /* Plain-text fallback. */
  }
}
export function readImage(
  data: Pick<DataTransfer, 'getData'>,
  fallback?: CopiedImage,
): CopiedImage | undefined {
  try {
    const copied = JSON.parse(data.getData(imageMime)),
      image = copied.image
    if (
      copied.version === 1 &&
      typeof copied.token === 'string' &&
      typeof image?.id === 'string' &&
      typeof image.data === 'string' &&
      image.data.startsWith('data:image/') &&
      [image.x, image.y, image.width, image.height, image.rotation].every(Number.isFinite) &&
      image.width > 0 &&
      image.height > 0 &&
      Number.isInteger(image.page) &&
      image.page >= 0 &&
      ['line', 'hatch', 'stipple'].includes(image.mode) &&
      (image.textRegions === undefined || Array.isArray(image.textRegions))
    )
      return copied
  } catch {
    /* Ignore unrelated clipboard contents. */
  }
  if (fallback && data.getData('text/plain') === fallback.token) return fallback
}
export function pasteImage(
  copied: CopiedImage,
  settings: LayoutSettings,
  position: Pick<PaperPosition, 'page'> & Partial<PaperPosition>,
): ImageAsset {
  const image = clone(copied.image)
  const k = Math.min(1, settings.paperWidth / image.width, settings.paperHeight / image.height)
  const width = image.width * k,
    height = image.height * k
  return {
    ...image,
    id: newId(),
    page: position.page,
    width,
    height,
    x: Math.max(0, Math.min(settings.paperWidth - width, position.x ?? image.x + 2)),
    y: Math.max(0, Math.min(settings.paperHeight - height, position.y ?? image.y + 2)),
  }
}
