import { nextTick, ref, watch, type Ref } from 'vue'
import { native, snapshot } from '../services/bridge'
import { projectPayload } from './projectFile'
import { flushEditors } from './editorCommands'
import type { useWorkspace } from './useWorkspace'
import type { ImageAsset, LayoutSettings } from '../types'
import type { DiagramScene } from '../services/diagramTypes'
import { useDiagramTask } from './useDiagramTask'

interface DiagramEditor {
  imageImportAnchor(): { page: number; x: number; y: number }
  selectedImage(): ImageAsset | undefined
  applyImage(image: ImageAsset, replace: boolean): void
}

export function placeDiagram(
  data: string,
  scene: DiagramScene,
  settings: LayoutSettings,
  anchor: { page: number; x: number; y: number },
): ImageAsset {
  const availableWidth = settings.paperWidth - settings.marginLeft - settings.marginRight
  const availableHeight = settings.paperHeight - settings.marginTop - settings.marginBottom
  const scale = Math.min(
    150 / scene.width,
    availableWidth / scene.width,
    availableHeight / scene.height,
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
    y: Math.max(
      settings.marginTop,
      Math.min(anchor.y, settings.paperHeight - settings.marginBottom - height),
    ),
  }
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
  async function submit(data: string, scene: DiagramScene, cancelled: () => boolean) {
    if (opened !== w.projectOpened.value) throw new Error('当前项目已切换，请重新导入')
    for (let attempt = 0; attempt < 3; attempt++) {
      flushEditors()
      await nextTick()
      if (cancelled()) throw new Error('已取消导入')
      const revision = w.contentVersion.value
      const current =
        replacement.value && w.preferences.images.find((i) => i.id === replacement.value!.id)
      if (replacement.value && (!current || current.data !== replacement.value.data))
        throw new Error('原图已删除或更换，请重新选择')
      const replacing = !!current && data === current.data
      const image = replacing
        ? { ...snapshot(current!), diagram: snapshot(scene) }
        : placeDiagram(data, snapshot(scene), w.preferences.layout, anchor)
      const images = replacing
        ? w.preferences.images.map((i) => (i.id === image.id ? image : i))
        : [...w.preferences.images, image]
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
      editor.value!.applyImage(image, replacing)
      w.tell(replacing ? '已重建图像，可撤销恢复' : '已导入图像，可拖动和缩放', 'success')
      return
    }
    throw new Error('文档正在修改，请结束操作后重试')
  }
  watch(w.projectOpened, () => {
    void task.cancel()
  })
  return { task, replacement, show }
}
