import type { LayoutSettings } from '../types'

export const defaultFontPercent = 110

export function fontMillimeters(percent: number, lineHeight: number): number {
  return (lineHeight * percent) / 100
}

export function fontPercent(size: number, lineHeight: number): number {
  return (size * 100) / lineHeight
}

// Keep the saved/project format in millimeters. Percentage controls register
// against the configured row height, so changing rows or paper retains scale.
export function updateLayoutSettings(
  settings: LayoutSettings,
  patch: Partial<LayoutSettings>,
): LayoutSettings {
  const next = { ...settings, ...patch }
  if (patch.lineCount !== undefined || patch.lineMode !== undefined) next.pageLayouts = undefined
  next.lineHeight = (next.paperHeight - next.marginTop - next.marginBottom) / next.lineCount
  if (patch.fontSize === undefined && next.lineHeight !== settings.lineHeight) {
    next.fontSize = fontMillimeters(
      fontPercent(settings.fontSize, settings.lineHeight),
      next.lineHeight,
    )
  }
  return next
}
