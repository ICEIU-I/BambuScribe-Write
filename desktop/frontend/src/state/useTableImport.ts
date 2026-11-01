import { nextTick, onBeforeUnmount, onMounted, ref, watch, type Ref } from 'vue'
import { errorText, native, snapshot } from '../services/bridge'
import { projectPayload } from './projectFile'
import { flushEditors } from './editorCommands'
import type { useWorkspace } from './useWorkspace'
import type { TableAsset } from '../types'
import type {
  Extraction,
  ImportPlacement,
  ImportRequest,
  LiveRequest,
} from '../services/tableImportTypes'
interface ImportEditor {
  importAnchor(): ImportPlacement
  addImportedTables(objects: TableAsset[]): void
}
export function useTableImport(
  w: ReturnType<typeof useWorkspace>,
  editor: Ref<ImportEditor | undefined>,
) {
  const open = ref(false)
  let importProject = w.projectOpened.value
  watch(open, (value) => {
    if (value) importProject = w.projectOpened.value
  })
  let stop = () => {}
  let pointerActive = false
  const pointerDown = (event: PointerEvent) => {
    if ((event.target as Element | null)?.closest?.('.document-viewport')) pointerActive = true
  }
  const pointerUp = () => {
    pointerActive = false
  }
  let queue: Promise<unknown> = Promise.resolve()
  const payload = () =>
    projectPayload({
      text: w.preferences.text,
      calculations: snapshot(w.preferences.calculations ?? []),
      layout: snapshot(w.preferences.layout),
      writing: snapshot(w.preferences.writing),
      images: snapshot(w.preferences.images),
      tables: snapshot(w.preferences.tables),
      erasures: snapshot(w.preferences.erasures),
    })
  async function current(full = false) {
    flushEditors()
    await nextTick()
    if (!w.ready.value || !editor.value) throw new Error('编辑器尚未就绪，请稍后重试')
    if (pointerActive) throw new Error('请结束拖动后再导入表格')
    const project = payload()
    if (!full)
      project.images.forEach((im) => {
        im.data = ''
      })
    return {
      project,
      revision: w.contentVersion.value,
      page: w.pageIndex.value + 1,
      projectPath: w.projectPath.value,
    }
  }
  async function insert(
    request: ImportRequest,
    cancelled: () => boolean = () => false,
    active?: () => Promise<boolean>,
  ) {
    const opened = w.projectOpened.value
    for (let attempt = 0; attempt < 3; attempt++) {
      if (cancelled()) throw new Error('已取消导入')
      const state = await current(true)
      if (request.expectedRevision !== undefined && request.expectedRevision !== state.revision)
        throw new Error('文档已修改，请读取最新快照后重试')
      const where = request.placement ?? editor.value!.importAnchor()
      const plan = await native().PlanTableImport(
        state.project,
        request.extraction,
        snapshot(where),
      )
      if (active && !(await active())) throw new Error('编辑请求已取消或超时')
      flushEditors()
      await nextTick()
      if (pointerActive) throw new Error('请结束拖动后再导入表格')
      if (opened !== w.projectOpened.value) throw new Error('当前项目已切换，请在目标项目重新导入')
      if (state.revision !== w.contentVersion.value) continue
      if (cancelled()) throw new Error('已取消导入')
      w.tab.value = 'write'
      editor.value!.addImportedTables(plan.objects)
      await nextTick()
      const warnings = [...new Set(plan.warnings)]
      w.tell(
        `已导入 ${request.extraction.tables.length} 个表格，位于第 ${[...new Set(plan.pages)].join('、')} 页${warnings.length ? `；${warnings.join('；')}` : '。'}`,
        'success',
      )
      return { ok: true, plan, revision: w.contentVersion.value }
    }
    throw new Error('文档正在持续修改，请结束拖动或输入后重试')
  }
  async function preview(extraction: Extraction, placement: ImportPlacement) {
    if (importProject !== w.projectOpened.value) throw new Error('当前项目已切换，请重新打开导入')
    const opened = w.projectOpened.value
    const state = await current(true)
    const plan = await native().PlanTableImport(
      state.project,
      snapshot(extraction),
      snapshot(placement),
    )
    if (opened !== w.projectOpened.value || state.revision !== w.contentVersion.value)
      throw new Error('文档已修改，请重新预览')
    return {
      ...plan,
      settings: state.project.layout,
      revision: state.revision,
      opened,
      occupied: (w.document.value?.pages ?? []).flatMap((page, i) =>
        [...page.cells, ...(page.tables ?? []), ...(page.images ?? [])].map((o) => ({
          page: i + 1,
          x: o.x,
          y: o.y,
          width: o.width,
          height: o.height,
        })),
      ),
    }
  }
  async function submit(request: ImportRequest, opened: number, cancelled: () => boolean) {
    if (opened !== w.projectOpened.value) throw new Error('当前项目已切换，请重新预览')
    return insert(request, cancelled)
  }
  onMounted(() => {
    window.addEventListener('pointerdown', pointerDown, true)
    window.addEventListener('pointerup', pointerUp, true)
    window.addEventListener('pointercancel', pointerUp, true)
    stop =
      window.runtime?.EventsOn('editor:request', (request: LiveRequest) => {
        queue = queue
          .catch(() => {})
          .then(async () => {
            try {
              const result =
                request.action === 'snapshot'
                  ? await current(request.params?.full)
                  : await insert(
                      request.params,
                      () => false,
                      () =>
                        native().EditorRequestActive
                          ? native().EditorRequestActive(request.id)
                          : Promise.resolve(true),
                    )
              await native().CompleteEditorRequest(request.id, { result })
            } catch (e) {
              await native().CompleteEditorRequest(request.id, { error: errorText(e) })
            }
          })
      }) ?? (() => {})
    if (w.ready.value && native().SetEditorReady) void native().SetEditorReady(true)
  })
  watch(w.ready, (ready) => {
    if (native().SetEditorReady) void native().SetEditorReady(ready)
  })
  onBeforeUnmount(() => {
    stop()
    window.removeEventListener('pointerdown', pointerDown, true)
    window.removeEventListener('pointerup', pointerUp, true)
    window.removeEventListener('pointercancel', pointerUp, true)
    if (native().SetEditorReady) void native().SetEditorReady(false)
  })
  return {
    open,
    preview,
    submit,
    anchor: () => editor.value?.importAnchor() ?? { page: w.pageIndex.value + 1, y: 0 },
  }
}
