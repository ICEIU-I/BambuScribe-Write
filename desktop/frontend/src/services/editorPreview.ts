import type { Document, ImageAsset, LayoutSettings, PageErasure, Point, TableAsset } from '../types'
import { native } from './bridge'

export interface EditorPreview {
  document: unknown
  strokes: number[][]
}

const strokeFields = new Set(['strokes', 'textStrokes', 'graphicStrokes', 'sourceStrokes'])

export function unpackEditorPreview(preview: EditorPreview): Document {
  const strokes = preview.strokes.map((coordinates) => {
    const stroke: Point[] = []
    for (let i = 0; i < coordinates.length; i += 2)
      stroke.push({ x: coordinates[i]!, y: coordinates[i + 1]! })
    return stroke
  })
  function restore(value: unknown): unknown {
    if (Array.isArray(value)) return value.map(restore)
    if (!value || typeof value !== 'object') return value
    const object = value as Record<string, unknown>
    for (const [key, item] of Object.entries(object)) {
      object[key] =
        strokeFields.has(key) && Array.isArray(item)
          ? item.map((index: number) => strokes[index]!)
          : restore(item)
    }
    return object
  }
  return restore(preview.document) as Document
}

export async function previewEditor(
  text: string,
  settings: LayoutSettings,
  images: ImageAsset[],
  tables: TableAsset[],
  erasures: PageErasure[],
): Promise<Document> {
  const api = native()
  if (api.PreviewEditor)
    return unpackEditorPreview(await api.PreviewEditor(text, settings, images, tables, erasures))
  return erasures.length
    ? api.PreviewWithErasures(text, settings, images, tables, erasures)
    : tables.length
      ? api.PreviewWithObjects(text, settings, images, tables)
      : api.PreviewWithImages(text, settings, images)
}
