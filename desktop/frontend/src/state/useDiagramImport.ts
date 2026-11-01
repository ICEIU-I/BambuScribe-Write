import { nextTick, ref, watch, type Ref } from 'vue'
import { native, snapshot } from '../services/bridge'
import { projectPayload } from './projectFile'
import { flushEditors } from './editorCommands'
import type { useWorkspace } from './useWorkspace'
import type { ImageAsset } from '../types'
import type { RecognizedDiagram } from '../services/diagramTypes'
import { useDiagramTask } from './useDiagramTask'
import { placeDiagram, nextDiagramAnchor } from './diagramPlacement'

interface DiagramEditor {
  imageImportAnchor(): { page: number; x: number; y: number }
  selectedImage(): ImageAsset | undefined
  applyImages(images: ImageAsset[]): void
}

export function useDiagramImport(
  w: ReturnType<typeof useWorkspace>,
  editor: Ref<DiagramEditor | undefined>,
) {
  const replacement = ref<ImageAsset>()
  const task = useDiagramTask(submit, (message) => w.tell(message, 'error'))
  let opened = 0,
    anchor = { page: 0, x: 0, y: 0 }
  function show() {
    if (task.retained.value || task.busy.value) {
      task.resume()
      return
    }
    opened = w.projectOpened.value
    anchor = editor.value?.imageImportAnchor() ?? { page: w.pageIndex.value, x: 8, y: 8 }
    replacement.value = snapshot(editor.value?.selectedImage() ?? null) ?? undefined
    task.prepare(replacement.value?.data ?? '')
  }
  async function submit(sources: RecognizedDiagram[], cancelled: () => boolean) {
    if (opened !== w.projectOpened.value) throw new Error('当前项目已切换，请重新导入')
    for (let attempt = 0; attempt < 3; attempt++) {
      flushEditors()
      await nextTick()
      if (cancelled()) throw new Error('已取消导入')
      const revision = w.contentVersion.value
      const replaceIndex = replacement.value
        ? sources.findIndex((s) => s.data === replacement.value!.data)
        : -1
      const current = w.preferences.images.find((i) => i.id === replacement.value?.id)
      if (replaceIndex >= 0 && (!current || current.data !== replacement.value!.data))
        throw new Error('原图已删除或更换，请重新选择')
      const settings = w.preferences.layout
      let cursor = { ...anchor }
      const imported = sources.map(({ data, scene }, index) => {
        const image =
          index === replaceIndex
            ? { ...snapshot(current!), diagram: snapshot(scene) }
            : placeDiagram(data, snapshot(scene), settings, cursor)
        cursor = nextDiagramAnchor(image, settings)
        return image
      })
      const images = [
        ...w.preferences.images.map((i) => imported.find((next) => next.id === i.id) ?? i),
        ...imported.filter((i) => i.id !== current?.id),
      ]
      const p = projectPayload({ ...snapshot(w.preferences), images: snapshot(images) })
      await native().PreviewWithErasures(
        p.text,
        p.layout,
        p.images,
        p.tables ?? [],
        p.erasures ?? [],
      )
      if (opened !== w.projectOpened.value) throw new Error('当前项目已切换，请重新导入')
      if (cancelled()) throw new Error('已取消导入')
      if (revision !== w.contentVersion.value) continue
      w.tab.value = 'write'
      editor.value!.applyImages(imported)
      w.tell(
        sources.length > 1
          ? `已导入 ${sources.length} 张图像，可撤销恢复`
          : replaceIndex >= 0
            ? '已重建图像，可撤销恢复'
            : '已导入图像，可拖动和缩放',
        'success',
      )
      return
    }
    throw new Error('文档正在修改，请结束操作后重试')
  }
  watch(w.projectOpened, () => {
    void task.cancel()
  })
  return { task, replacement, show }
}
