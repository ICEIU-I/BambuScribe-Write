import { nextTick, onBeforeUnmount, type Ref } from 'vue'
import { errorText } from '../services/bridge'
import { imageDimensions, readImageFile } from '../services/imageFiles'
import { flushEditors } from './editorCommands'
import type { useWorkspace } from './useWorkspace'
import type { ImageAsset } from '../types'

interface ImageEditor {
  imageImportAnchor(): { page: number; x: number; y: number }
  applyImage(image: ImageAsset, replace: boolean): void
}

// Pasted raster images use the engine's existing OCR and line extraction.
// Model reconstruction is an explicit action in the smart-import dialog.
export function useImagePaste(
  w: ReturnType<typeof useWorkspace>,
  editor: Ref<ImageEditor | undefined>,
) {
  let disposed = false
  onBeforeUnmount(() => {
    disposed = true
  })
  async function paste(file: File) {
    const project = w.projectOpened.value
    const anchor = editor.value?.imageImportAnchor() ?? { page: w.pageIndex.value, x: 8, y: 8 }
    const current = () =>
      !disposed && project === w.projectOpened.value && w.ready.value && !!editor.value
    try {
      const data = await readImageFile(file)
      const dimensions = await imageDimensions(data)
      if (!current()) return
      const s = w.preferences.layout
      const scale = Math.min(
        80 / dimensions.width,
        (s.paperWidth - 16) / dimensions.width,
        (s.paperHeight - 16) / dimensions.height,
      )
      const width = dimensions.width * scale,
        height = dimensions.height * scale
      if (width < 2 || height < 2) throw new Error('图片无法放入当前纸张，请调整排版设置')
      const asset: ImageAsset = {
        id: `image-${globalThis.crypto.randomUUID()}`,
        data,
        page: Math.max(0, anchor.page),
        x: Math.max(0, Math.min(anchor.x, s.paperWidth - width)),
        y: Math.max(0, Math.min(anchor.y, s.paperHeight - height)),
        width,
        height,
        rotation: 0,
        mode: 'line',
      }
      flushEditors()
      await nextTick()
      if (current()) editor.value!.applyImage(asset, false)
    } catch (error) {
      if (current()) w.tell(errorText(error), 'error')
    }
  }
  return { paste }
}
