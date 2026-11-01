import { provide, shallowRef, type InjectionKey, type ShallowRef } from 'vue'
import type { Document, Point, TableAsset } from '../types'
import { clampImportedGroup, importedGroupGeometry } from './importedTableGroup'

interface GroupPreview {
  owner: string
  ids: Set<string>
  page: number
  dx: number
  dy: number
  offset: Point
}

// Keep every member mounted on its source page until the drop, so moving to
// another page does not destroy pointer capture or commit intermediate edits.
export function provideImportedTableDrag(
  tables: () => TableAsset[],
  doc: () => Document | null,
  commit: (next: TableAsset) => void,
) {
  const preview = shallowRef<GroupPreview | null>(null)
  const context = {
    preview,
    commit,
    move(next: TableAsset, offset: Point) {
      const document = doc(),
        group = importedGroupGeometry(next.id, tables(), document)
      if (!group || group.split || !document) return next
      const limited = clampImportedGroup(next, group, document.settings)
      preview.value = {
        owner: next.id,
        ids: new Set(group.members.map((t) => t.id)),
        page: limited.page,
        dx: limited.x - group.anchor.x,
        dy: limited.y - group.anchor.y,
        offset,
      }
      return limited
    },
    clear(id: string) {
      if (preview.value?.owner === id) preview.value = null
    },
  }
  provide(importedTableDragKey, context)
}

export const importedTableDragKey: InjectionKey<{
  preview: ShallowRef<GroupPreview | null>
  move(next: TableAsset, offset: Point): TableAsset
  clear(id: string): void
  commit(next: TableAsset): void
}> = Symbol('imported-table-drag')
