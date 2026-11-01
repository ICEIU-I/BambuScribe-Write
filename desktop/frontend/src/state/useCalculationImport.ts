import { nextTick, onBeforeUnmount, ref, watch, type Ref } from 'vue'
import { native, snapshot, errorText } from '../services/bridge'
import { flushEditors } from './editorCommands'
import { calculationAt, insertCalculation } from './calculationText'
import type {
  CalculationPlan,
  CalculationRecord,
  CalculationPreview,
  SourceImage,
} from '../services/calculationTypes'
import type { useWorkspace } from './useWorkspace'
interface CalculationEditor {
  textAnchor(): { from: number; to: number }
  replaceGenerated(from: number, to: number, value: string, record: CalculationRecord): void
}
export function useCalculationImport(
  w: ReturnType<typeof useWorkspace>,
  editor: Ref<CalculationEditor | undefined>,
) {
  const open = ref(false),
    working = ref(false),
    reading = ref(false),
    planning = ref(false),
    saving = ref(false)
  const data = ref<SourceImage[]>([]),
    references = ref<SourceImage[]>([]),
    instruction = ref('')
  const plan = ref<CalculationPlan>(),
    preview = ref<CalculationPreview>(),
    error = ref('')
  const fractionScale = ref(85),
    editing = ref(false)
  let opened = 0,
    version = 0,
    generation = 0,
    previewGeneration = 0
  let anchor = { from: 0, to: 0 },
    replacement: CalculationRecord | undefined
  let timer: ReturnType<typeof setTimeout> | undefined
  async function show(index?: number) {
    if (working.value || saving.value) {
      open.value = true
      return
    }
    flushEditors()
    await nextTick()
    opened = w.projectOpened.value
    version = w.contentVersion.value
    anchor = editor.value?.textAnchor() ?? { from: 0, to: 0 }
    const found =
      index === undefined
        ? undefined
        : calculationAt(w.preferences.text, w.preferences.calculations ?? [], index)
    replacement = found?.record
    editing.value = !!found
    if (found) {
      data.value = []
      references.value = []
      instruction.value = ''
      anchor = { from: found.from, to: found.to }
      plan.value = snapshot(found.record.plan)
    } else {
      plan.value = undefined
      data.value = []
      references.value = []
      instruction.value = ''
    }
    fractionScale.value = w.preferences.layout.fractionScale || 85
    error.value = ''
    preview.value = undefined
    open.value = true
    if (plan.value) void render()
  }
  async function add(role: 'data' | 'references', files: File[]) {
    if (working.value || reading.value || saving.value) return
    const group = role === 'data' ? data : references
    reading.value = true
    error.value = ''
    try {
      for (const file of files) {
        if (!['image/png', 'image/jpeg'].includes(file.type)) throw new Error('请选择PNG或JPEG图片')
        if (group.value.length >= 10) throw new Error('每组最多10张图片')
        if (file.size > 12 * 1024 * 1024) throw new Error('单张图片不能超过12 MB')
        if (
          [...data.value, ...references.value].reduce((n, i) => n + i.size, 0) + file.size >
          48 * 1024 * 1024
        )
          throw new Error('图片合计不能超过48 MB')
        const value = await new Promise<string>((resolve, reject) => {
          const r = new FileReader()
          r.onload = () => resolve(String(r.result))
          r.onerror = () => reject(new Error('图片读取失败，请重试'))
          r.readAsDataURL(file)
        })
        group.value.push({
          id: crypto.randomUUID(),
          name: file.name || '剪贴板图片',
          data: value,
          size: file.size,
        })
      }
    } catch (e) {
      error.value = errorText(e)
    } finally {
      reading.value = false
    }
  }
  function invalidate() {
    previewGeneration++
    preview.value = undefined
    clearTimeout(timer)
    planning.value = false
  }
  async function render() {
    invalidate()
    if (!plan.value || !open.value) return
    const ticket = previewGeneration
    planning.value = true
    error.value = ''
    try {
      const result = await native().PreviewCalculations(snapshot(plan.value), {
        ...snapshot(w.preferences.layout),
        pageLayouts: undefined,
        fractionScale: fractionScale.value,
      })
      if (ticket === previewGeneration && open.value) preview.value = result
    } catch (e) {
      if (ticket === previewGeneration) error.value = errorText(e)
    } finally {
      if (ticket === previewGeneration) planning.value = false
    }
  }
  function changed() {
    invalidate()
    timer = setTimeout(() => void render(), 250)
  }
  async function recognize() {
    if (!data.value.length || !references.value.length) {
      error.value = '请添加原始数据和参考样例图片'
      return
    }
    const ticket = ++generation
    working.value = true
    error.value = ''
    invalidate()
    try {
      const result = await native().RecognizeCalculations({
        data: data.value.map((i) => i.data),
        references: references.value.map((i) => i.data),
        instruction: instruction.value,
      })
      if (ticket !== generation || opened !== w.projectOpened.value || !open.value) return
      plan.value = result
      await render()
    } catch (e) {
      if (ticket === generation) error.value = errorText(e)
    } finally {
      if (ticket === generation) working.value = false
    }
  }
  async function close() {
    generation++
    invalidate()
    open.value = false
    if (working.value) {
      await native()
        .CancelCalculationRecognition()
        .catch(() => {})
      working.value = false
    }
  }
  async function insert() {
    if (!preview.value || !plan.value || planning.value || saving.value) return
    saving.value = true
    error.value = ''
    const result = preview.value,
      ticket = generation
    try {
      flushEditors()
      await nextTick()
      if (opened !== w.projectOpened.value || version !== w.contentVersion.value)
        throw new Error('文档已修改，请关闭后重新打开数据处理')
      const inserted = insertCalculation(w.preferences.text, anchor.from, anchor.to, result.text)
      const text =
        w.preferences.text.slice(0, anchor.from) + inserted + w.preferences.text.slice(anchor.to)
      await native().PreviewWithErasures(
        text,
        { ...snapshot(w.preferences.layout), fractionScale: fractionScale.value },
        snapshot(w.preferences.images),
        snapshot(w.preferences.tables),
        snapshot(w.preferences.erasures),
      )
      if (ticket !== generation || !open.value) return
      if (opened !== w.projectOpened.value || version !== w.contentVersion.value)
        throw new Error('文档已修改，请关闭后重新打开数据处理')
      const record: CalculationRecord = {
        id: replacement?.id ?? crypto.randomUUID(),
        text: result.text,
        plan: snapshot(plan.value),
      }
      // Settings and text are committed together before the next render/save.
      w.preferences.layout.fractionScale = fractionScale.value
      editor.value!.replaceGenerated(anchor.from, anchor.to, inserted, record)
      open.value = false
      w.tell(editing.value ? '已更新数据处理' : '已插入数据处理，双击公式可修改数据', 'success')
    } catch (e) {
      error.value = errorText(e)
    } finally {
      saving.value = false
    }
  }
  watch(w.projectOpened, () => {
    void close()
  })
  onBeforeUnmount(() => {
    clearTimeout(timer)
    void close()
  })
  return {
    open,
    working,
    reading,
    planning,
    saving,
    data,
    references,
    instruction,
    plan,
    preview,
    error,
    fractionScale,
    editing,
    show,
    add,
    recognize,
    render,
    changed,
    insert,
    close,
  }
}
