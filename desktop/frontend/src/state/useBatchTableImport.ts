import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { errorText, native, snapshot } from '../services/bridge'
import type {
  Extraction,
  ImportPlacement,
  ImportPosition,
  ImportPreview,
} from '../services/tableImportTypes'
export interface ImportSource {
  id: number
  name: string
  data: string
  size: number
  result?: Extraction
  error?: string
  processing?: boolean
}
export function useBatchTableImport(
  preview: (e: Extraction, p: ImportPlacement) => Promise<ImportPreview>,
) {
  const sources = ref<ImportSource[]>([]),
    instruction = ref(''),
    error = ref(''),
    working = ref(false)
  const reading = ref(false),
    cancelled = ref(false),
    reviewing = ref(false)
  const extraction = ref<Extraction>({ tables: [] }),
    origins = ref<number[]>([])
  const plan = ref<ImportPreview>(),
    positions = ref<ImportPosition[]>(),
    planning = ref(false),
    valid = ref(false)
  let id = 0,
    generation = 0,
    placement: ImportPlacement = { page: 1, y: 0 }
  const processingCount = computed(() => sources.value.filter((s) => s.processing).length)
  const finishedCount = computed(() => sources.value.filter((s) => s.result || s.error).length)
  const complete = computed(() => sources.value.length > 0 && sources.value.every((s) => s.result))
  watch(instruction, () => {
    for (const s of sources.value) {
      s.result = undefined
      s.error = undefined
    }
    reviewing.value = false
    invalidate()
  })
  function invalidate() {
    generation++
    valid.value = false
    planning.value = false
  }
  async function add(files: File[]) {
    if (working.value || reading.value) return
    reading.value = true
    error.value = ''
    try {
      for (const f of files) {
        if (!['image/png', 'image/jpeg'].includes(f.type))
          throw new Error('请选择 PNG 或 JPEG 图片')
        if (sources.value.length >= 10) throw new Error('每批最多 10 张图片，请分批导入')
        if (f.size > 12 * 1024 * 1024) throw new Error(`${f.name} 超过 12 MB，请裁剪后重试`)
        if (sources.value.reduce((n, s) => n + s.size, 0) + f.size > 48 * 1024 * 1024)
          throw new Error('本批图片超过 48 MB，请分批导入')
        const data = await new Promise<string>((resolve, reject) => {
          const r = new FileReader()
          r.onload = () => resolve(String(r.result))
          r.onerror = () => reject(new Error('图片读取失败，请重新选择'))
          r.readAsDataURL(f)
        })
        sources.value.push({ id: ++id, name: f.name || '剪贴板图片', data, size: f.size })
        reviewing.value = false
        invalidate()
      }
    } catch (e) {
      error.value = errorText(e)
    } finally {
      reading.value = false
    }
  }
  function remove(index: number) {
    sources.value.splice(index, 1)
    reviewing.value = false
    invalidate()
  }
  async function arrange(next?: ImportPosition[]) {
    const ticket = ++generation
    positions.value = next ? snapshot(next) : undefined
    valid.value = false
    planning.value = true
    error.value = ''
    try {
      const result = await preview(
        snapshot(extraction.value),
        next ? { page: 1, y: 0, positions: snapshot(next) } : placement,
      )
      if (ticket !== generation || cancelled.value) return
      plan.value = result
      positions.value = result.groups.map(({ page, x, y, width }) => ({ page, x, y, width }))
      valid.value = true
    } catch (e) {
      if (ticket === generation) error.value = errorText(e)
    } finally {
      if (ticket === generation) planning.value = false
    }
  }
  async function run(anchor: ImportPlacement) {
    if (working.value) return
    working.value = true
    cancelled.value = false
    error.value = ''
    placement = anchor
    try {
      const pending = sources.value.filter((s) => !s.result)
      for (const s of pending) s.error = undefined
      const requestedInstruction = instruction.value
      let cursor = 0
      async function worker() {
        while (!cancelled.value && cursor < pending.length) {
          const s = pending[cursor++]!
          s.processing = true
          try {
            const result = await native().RecognizeTables(s.data, requestedInstruction)
            if (!cancelled.value) s.result = snapshot(result)
          } catch (e) {
            if (!cancelled.value) s.error = errorText(e)
          } finally {
            s.processing = false
          }
        }
      }
      await Promise.all(Array.from({ length: Math.min(3, pending.length) }, () => worker()))
      if (cancelled.value) return
      if (!complete.value) {
        error.value = '部分图片未完成，请重试或移除失败图片'
        return
      }
      extraction.value = { tables: sources.value.flatMap((s) => s.result!.tables) }
      origins.value = sources.value.flatMap((s, i) => s.result!.tables.map(() => i))
      plan.value = undefined
      positions.value = undefined
      reviewing.value = true
      await arrange()
    } finally {
      working.value = false
    }
  }
  async function cancel() {
    cancelled.value = true
    invalidate()
    try {
      await native().CancelTableRecognition()
    } catch (e) {
      error.value = errorText(e)
    }
  }
  function reset() {
    invalidate()
    sources.value = []
    plan.value = undefined
    positions.value = undefined
    reviewing.value = false
    error.value = ''
    extraction.value = { tables: [] }
  }
  onBeforeUnmount(() => {
    cancelled.value = true
    invalidate()
    if (working.value)
      void native()
        .CancelTableRecognition()
        .catch(() => {})
  })
  return {
    sources,
    instruction,
    error,
    working,
    reading,
    processingCount,
    finishedCount,
    cancelled,
    reviewing,
    extraction,
    origins,
    plan,
    positions,
    planning,
    valid,
    complete,
    add,
    remove,
    arrange,
    run,
    cancel,
    reset,
    invalidate,
  }
}
