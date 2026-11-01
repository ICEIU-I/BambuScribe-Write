import type { ImageAsset, ImageTextRegion, Point } from '../types'

const mime = 'application/x-bambuscribe-textbox'
export interface CopiedTextBox {
  version: 1
  text: string
  width: number
  height: number
  x: number
  y: number
}
export function copyTextBox(image: ImageAsset, region: ImageTextRegion): CopiedTextBox {
  return {
    version: 1,
    text: region.text,
    width: region.width * image.width,
    height: region.height * image.height,
    x: region.x * image.width,
    y: region.y * image.height,
  }
}
export function writeTextBox(data: DataTransfer, box: CopiedTextBox) {
  data.setData('text/plain', box.text)
  try {
    data.setData(mime, JSON.stringify(box))
  } catch {
    /* WebViews may only retain plain text. */
  }
}
export function readTextBox(
  data: DataTransfer,
  fallback?: CopiedTextBox,
): CopiedTextBox | undefined {
  try {
    const box = JSON.parse(data.getData(mime))
    if (
      box.version === 1 &&
      typeof box.text === 'string' &&
      box.text.length <= 1000 &&
      [box.x, box.y, box.width, box.height].every(Number.isFinite) &&
      box.width > 0 &&
      box.height > 0
    )
      return box
  } catch {
    /* Ignore unrelated clipboard contents. */
  }
  // Only use our in-memory object when the system clipboard still contains
  // what we copied; copying unrelated text must not paste a stale object.
  if (fallback && data.getData('text/plain') === fallback.text) return fallback
}
export function pasteTextBox(
  box: CopiedTextBox,
  image: ImageAsset,
  position?: Point,
): ImageTextRegion {
  const k = Math.min(1, image.width / box.width, image.height / box.height)
  const width = (box.width * k) / image.width,
    height = (box.height * k) / image.height
  return {
    id:
      globalThis.crypto?.randomUUID?.() ??
      `text-${Date.now()}-${Math.random().toString(16).slice(2)}`,
    text: box.text,
    source: 'manual',
    mode: 'handwriting',
    width,
    height,
    x: Math.max(0, Math.min(1 - width, position?.x ?? (box.x + 2) / image.width)),
    y: Math.max(0, Math.min(1 - height, position?.y ?? (box.y + 2) / image.height)),
  }
}
