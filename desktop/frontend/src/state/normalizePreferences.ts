import type { ImageTextRegion, LayoutSettings, Preferences, PageErasure } from '../types'
import { validInkCuts } from './inkCuts'
import { defaultPreferences } from './defaults'
import { defaultFontPercent, fontMillimeters } from './fontSizing'

type SavedPreferences = Partial<Omit<Preferences, 'layout' | 'writing'>> & {
  layout?: Partial<LayoutSettings> | null
  writing?: Partial<Preferences['writing']> | null
}

// Fill fields introduced by the desktop editor without discarding saved text,
// paper sizes, margins or explicit invalid values that the engine must validate.
export function normalizePreferences(input: SavedPreferences | null | undefined): Preferences {
  const base = defaultPreferences()
  const source = input ?? {}
  const layout = { ...base.layout, ...source.layout }
  if (layout.letterSpacing === undefined || layout.letterSpacing === 0) layout.letterSpacing = 100
  if (layout.lineSpacing === undefined || layout.lineSpacing === 0) layout.lineSpacing = 100
  if (!layout.templateName?.trim()) layout.templateName = base.layout.templateName
  if (!source.layout?.lineMode) {
    layout.lineMode = 'fixed'
    const height = layout.paperHeight - layout.marginTop - layout.marginBottom
    const oldLineHeight = source.layout?.lineHeight
    if (oldLineHeight && Number.isFinite(oldLineHeight) && oldLineHeight > 0 && height > 0) {
      layout.lineCount = Math.max(1, Math.floor(height / oldLineHeight))
    }
  } else if (source.layout.lineCount === undefined) {
    layout.lineCount = base.layout.lineCount
  }
  layout.lineHeight =
    (layout.paperHeight - layout.marginTop - layout.marginBottom) / layout.lineCount
  if (
    source.layout?.fontSize === undefined &&
    Number.isFinite(layout.lineHeight) &&
    layout.lineHeight > 0
  ) {
    layout.fontSize = fontMillimeters(defaultFontPercent, layout.lineHeight)
  }
  // Keep only supported writing fields when loading old drafts or projects.
  const writing = { ...base.writing, ...source.writing }
  return {
    theme: source.theme === 'dark' ? 'dark' : 'light',
    language: ['zh', 'en'].includes(source.language as string)
      ? (source.language as 'zh' | 'en')
      : 'system',
    text: typeof source.text === 'string' ? source.text : base.text,
    layout,
    writing: {
      offsetX: writing.offsetX,
      offsetY: writing.offsetY,
      penUp: writing.penUp,
      penDown: writing.penDown,
      originMode: writing.originMode,
      sound: writing.sound,
    },
    erasures: Array.isArray(source.erasures) ? source.erasures.filter(validPageErasure) : [],
    tables: Array.isArray(source.tables) ? source.tables : [],
    images: Array.isArray((source as SavedPreferences & { images?: unknown }).images)
      ? ((source as SavedPreferences & { images?: Preferences['images'] }).images ?? [])
          .filter(
            (image) =>
              image &&
              typeof image.id === 'string' &&
              typeof image.data === 'string' &&
              Number.isFinite(image.x) &&
              Number.isFinite(image.y) &&
              Number.isFinite(image.width) &&
              Number.isFinite(image.height),
          )
          .map((image) => ({
            ...image,
            textRegions: Array.isArray(image.textRegions)
              ? image.textRegions.filter(validTextRegion)
              : undefined,
          }))
      : [],
  }
}

function validPageErasure(e: PageErasure): boolean {
  const valid = (v: number) => Number.isFinite(v) && v >= 0 && v <= 1000
  return (
    !!e &&
    typeof e.id === 'string' &&
    validInkCuts(e.cuts) &&
    !!e.id &&
    (e.objectIds === undefined ||
      (Array.isArray(e.objectIds) &&
        e.objectIds.length <= 200 &&
        e.objectIds.every((id) => typeof id === 'string' && !!id && id.length <= 1000) &&
        new Set(e.objectIds).size === e.objectIds.length)) &&
    Number.isInteger(e.page) &&
    e.page >= 0 &&
    e.page <= 20000 &&
    [e.x, e.y, e.width, e.height].every(valid) &&
    (e.path?.length
      ? Array.isArray(e.path) &&
        e.path.every((p) => !!p && valid(p.x) && valid(p.y)) &&
        Number.isFinite(e.radius) &&
        e.radius! > 0 &&
        e.radius! <= 20
      : e.width > 0 && e.height > 0)
  )
}

function validTextRegion(region: ImageTextRegion): boolean {
  return (
    !!region &&
    typeof region.id === 'string' &&
    typeof region.text === 'string' &&
    Number.isFinite(region.x) &&
    Number.isFinite(region.y) &&
    Number.isFinite(region.width) &&
    Number.isFinite(region.height) &&
    (region.mode === 'handwriting' || region.mode === 'erase' || region.mode === 'keep') &&
    (region.source === undefined || region.source === 'ocr' || region.source === 'manual')
  )
}
