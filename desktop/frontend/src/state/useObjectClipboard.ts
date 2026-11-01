import type { Document, ImageAsset, ImageTextRegion, Point } from '../types'
import { displayRegions } from './imageEditing'
import { isTextBox } from './textBox'
import { copyImage, pasteImage, readImage, writeImage, type CopiedImage } from './imageClipboard'
import type { PaperPosition } from './paperGeometry'
import {
  copyTextBox,
  pasteTextBox,
  readTextBox,
  writeTextBox,
  type CopiedTextBox,
} from './textBoxClipboard'

export interface ImageClipboardPlacement {
  location: () => Pick<PaperPosition, 'page'> & Partial<PaperPosition>
  pasted: (image: ImageAsset) => void
}
interface ClipboardContext {
  active?: boolean
  disabled: boolean
  document: Document | null
}
interface ClipboardActions {
  images: () => ImageAsset[]
  selection: () => { imageId: string; regionId: string; point?: Point }
  paperInput: () => HTMLElement | undefined
  advanceText: (point: Point) => void
  addImage: (image: ImageAsset) => void
  removeImage: (id: string) => void
  addRegion: (id: string, region: ImageTextRegion) => void
  removeRegion: (id: string, rid: string) => void
}

export function useObjectClipboard(
  props: ClipboardContext,
  actions: ClipboardActions,
  placement?: ImageClipboardPlacement,
) {
  let copiedImage: CopiedImage | undefined, copiedText: CopiedTextBox | undefined
  function allowed(event: ClipboardEvent, paste = false) {
    const target = event.target as HTMLElement | null
    if (props.active === false || props.disabled || target?.closest('dialog,[role="dialog"]'))
      return false
    if (target?.closest('input,textarea,[contenteditable="true"]'))
      return paste && target === actions.paperInput()
    return true
  }
  function copy(event: ClipboardEvent, cut = false) {
    if (event.defaultPrevented) return
    const selected = actions.selection(),
      data = event.clipboardData
    const image = actions.images().find((i) => i.id === selected.imageId)
    if (!allowed(event) || !image || !data) {
      copiedImage = undefined
      copiedText = undefined
      return
    }
    const region = image.textRegions?.find((r) => r.id === selected.regionId && isTextBox(r))
    if (selected.regionId && !region) return
    if (region) {
      const next = copyTextBox(image, region)
      writeTextBox(data, next)
      copiedText = next
      copiedImage = undefined
    } else {
      const preview = props.document?.pages
        .flatMap((p) => p.images ?? [])
        .find((i) => i.id === image.id)
      const next = copyImage({ ...image, textRegions: displayRegions(image, preview) })
      writeImage(data, next)
      copiedImage = next
      copiedText = undefined
    }
    event.preventDefault()
    if (cut) {
      if (region) actions.removeRegion(image.id, region.id)
      else actions.removeImage(image.id)
    }
  }
  function paste(event: ClipboardEvent): boolean {
    const data = event.clipboardData
    if (event.defaultPrevented || !data || !allowed(event, true)) return false
    const copied = readImage(data, copiedImage)
    if (copied && props.document && placement) {
      const image = pasteImage(copied, props.document.settings, placement.location())
      event.preventDefault()
      actions.paperInput()?.blur()
      actions.addImage(image)
      placement.pasted(image)
      return true
    }
    const selected = actions.selection()
    const box = readTextBox(data, copiedText)
    const image = actions.images().find((i) => i.id === selected.imageId)
    if (!box || !image) return false
    const region = pasteTextBox(box, image, selected.point)
    event.preventDefault()
    actions.addRegion(image.id, region)
    actions.advanceText({ x: region.x + 2 / image.width, y: region.y + 2 / image.height })
    return true
  }
  return {
    copy: (event: ClipboardEvent) => copy(event),
    cut: (event: ClipboardEvent) => copy(event, true),
    paste,
  }
}
