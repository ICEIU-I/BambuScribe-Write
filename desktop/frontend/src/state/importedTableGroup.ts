import type { Document, LayoutSettings, TableAsset } from '../types'
import { placedObjects } from './objectLayout'
import { sum } from './tableGeometry'

export function importedGroupGeometry(
  id: string,
  tables: TableAsset[],
  doc: Document | null,
  wholeGroup = true,
) {
  const placed = placedObjects(tables, doc),
    anchor = placed.find((t) => t.id === id)
  if (!anchor?.groupId || !doc) return
  const members = wholeGroup ? placed.filter((t) => t.groupId === anchor.groupId) : [anchor]
  if (wholeGroup && members.length < 2) return
  const rects = members.map((t) => ({
    x: t.x,
    y: t.y,
    width: doc.pages[t.page]?.tables?.find((o) => o.id === t.id)?.width ?? sum(t.columnWidths),
    height: doc.pages[t.page]?.tables?.find((o) => o.id === t.id)?.height ?? sum(t.rowHeights),
  }))
  return {
    anchor,
    members,
    split: members.some((t) => t.page !== anchor.page),
    left: Math.min(...rects.map((r) => r.x)),
    top: Math.min(...rects.map((r) => r.y)),
    right: Math.max(...rects.map((r) => r.x + r.width)),
    bottom: Math.max(...rects.map((r) => r.y + r.height)),
  }
}

export function clampImportedGroup(
  next: TableAsset,
  group: NonNullable<ReturnType<typeof importedGroupGeometry>>,
  settings: LayoutSettings,
): TableAsset {
  // The same measured group bounds govern both the drag preview and the drop.
  if (
    group.split ||
    group.right - group.left > settings.paperWidth ||
    group.bottom - group.top > settings.paperHeight
  )
    return next
  const dx = Math.max(
      -group.left,
      Math.min(settings.paperWidth - group.right, next.x - group.anchor.x),
    ),
    dy = Math.max(
      -group.top,
      Math.min(settings.paperHeight - group.bottom, next.y - group.anchor.y),
    )
  return { ...next, x: group.anchor.x + dx, y: group.anchor.y + dy }
}
// Imported members move independently unless the user chooses the group handle.
// Preserve their association so the adjusted arrangement can still move together.
export function moveImportedTable(
  next: TableAsset,
  tables: TableAsset[],
  doc: Document | null,
  wholeGroup = false,
): TableAsset[] | undefined {
  const source = tables.find((t) => t.id === next.id)
  if (
    !source?.groupId ||
    !doc ||
    (source.x === next.x && source.y === next.y && source.page === next.page)
  )
    return
  const content = (t: TableAsset) => JSON.stringify({ ...t, page: 0, x: 0, y: 0 })
  if (content(source) !== content(next)) return
  const group = importedGroupGeometry(next.id, tables, doc, wholeGroup)
  if (!group) return
  if (group.split) throw new Error('同组内容分布在多页，请先分别移到同一页')
  const { members, anchor: old } = group
  next = clampImportedGroup(next, group, doc.settings)
  const dx = next.x - old.x,
    dy = next.y - old.y
  const moved = members.map((t) => ({ ...t, page: next.page, x: t.x + dx, y: t.y + dy }))
  const rects = moved.map((t) => ({
    x: t.x,
    y: t.y,
    width: sum(t.columnWidths),
    height: doc.pages[old.page]?.tables?.find((o) => o.id === t.id)?.height ?? sum(t.rowHeights),
  }))
  const s = doc.settings
  if (
    rects.some(
      (r) =>
        r.x < 0 ||
        r.y < 0 ||
        r.x + r.width > s.paperWidth + 1e-6 ||
        r.y + r.height > s.paperHeight + 1e-6,
    )
  )
    throw new Error(
      wholeGroup ? '表格连同表名、说明超过整页，请缩小尺寸' : '对象超过整页，请缩小尺寸',
    )
  // Objects can overlap. The native layout moves whole body lines clear of
  // the final bounds, so a drop does not reject text that will be reflowed.
  const replacements = new Map(moved.map((t) => [t.id, t]))
  return tables.map((t) => replacements.get(t.id) ?? t)
}
