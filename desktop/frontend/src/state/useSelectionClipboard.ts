import type { Document, ImageAsset, TableAsset } from '../types'
import type { ObjectSelection } from './objectSelection'
import type { PaperPosition } from './paperGeometry'
import {
  copyObjects,
  pasteObjects,
  readObjects,
  writeObjects,
  objectGroupMime,
  type CopiedObjects,
} from './objectGroupClipboard'

export function useSelectionClipboard(
  props: {
    active: boolean
    disabled: boolean
    document: Document | null
    images: ImageAsset[]
    tables?: TableAsset[]
  },
  selection: () => ObjectSelection | null,
  input: () => HTMLElement | undefined,
  position: () => Pick<PaperPosition, 'page'> & Partial<PaperPosition>,
  add: (images: ImageAsset[], tables: TableAsset[]) => void,
  pasted: (position: PaperPosition) => void,
  notice: (message: string) => void,
) {
  let copied: CopiedObjects | undefined
  function allowed(event: ClipboardEvent, paste = false) {
    const target = event.target as HTMLElement | null
    return (
      props.active &&
      !props.disabled &&
      !event.defaultPrevented &&
      !target?.closest('dialog,[role="dialog"]') &&
      (!target?.closest('input,textarea,[contenteditable="true"]') || (paste && target === input()))
    )
  }
  function copy(event: ClipboardEvent) {
    const selected = selection(),
      doc = props.document,
      data = event.clipboardData
    if (!selected || !doc || !data || !allowed(event)) return false
    event.preventDefault()
    const next = copyObjects(doc, props.images, props.tables ?? [], selected)
    if (!next) {
      notice('框内没有可复制的图片、表格或文本框。')
      return true
    }
    writeObjects(data, next)
    copied = next
    return true
  }
  function paste(event: ClipboardEvent) {
    const data = event.clipboardData,
      doc = props.document
    if (!data || !doc || !allowed(event, true)) return false
    const objects = readObjects(data, copied)
    if (!objects) {
      if (
        data.getData(objectGroupMime) ||
        data.getData('text/html').includes('data-bambuscribe-objects=') ||
        data.getData('text/plain').startsWith('BambuScribe:objects:')
      ) {
        event.preventDefault()
        notice('选中内容无效，请重新框选复制。')
        return true
      }
      return false
    }
    event.preventDefault()
    try {
      const next = pasteObjects(objects, doc, props.images, props.tables ?? [], position())
      input()?.blur()
      add(next.images, next.tables)
      pasted({ page: next.page, x: next.x + 2, y: next.y + 2 })
    } catch (e) {
      notice((e as Error).message)
    }
    return true
  }
  return { copy, paste }
}
