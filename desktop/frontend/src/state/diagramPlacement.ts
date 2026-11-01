import type { ImageAsset, LayoutSettings } from '../types'
import type { DiagramScene } from '../services/diagramTypes'

type Anchor = { page: number; x: number; y: number }

export function placeDiagram(
  data: string,
  scene: DiagramScene,
  settings: LayoutSettings,
  anchor: Anchor,
): ImageAsset {
  const scale = Math.min(
    150 / scene.width,
    (settings.paperWidth - settings.marginLeft - settings.marginRight) / scene.width,
    (settings.paperHeight - settings.marginTop - settings.marginBottom) / scene.height,
  )
  const width = scene.width * scale,
    height = scene.height * scale
  if (width < 2 || height < 2) throw new Error('图像无法放入当前纸张，请调整排版设置')
  return {
    id: `image-${globalThis.crypto.randomUUID()}`,
    data,
    diagram: scene,
    page: Math.max(0, anchor.page),
    rotation: 0,
    mode: 'line',
    width,
    height,
    x: Math.max(
      settings.marginLeft,
      Math.min(anchor.x, settings.paperWidth - settings.marginRight - width),
    ),
    // Keep the insertion row. The engine moves following text out of the
    // frame, and carries a figure past the paper edge onto the next sheet.
    y: Math.max(settings.marginTop, anchor.y),
  }
}

export function nextDiagramAnchor(image: ImageAsset, settings: LayoutSettings): Anchor {
  const overflow = image.y + image.height > settings.paperHeight + 1e-8
  return {
    page: image.page + (overflow ? 1 : 0),
    x: image.x,
    y: (overflow ? settings.marginTop : image.y) + image.height + 4,
  }
}
