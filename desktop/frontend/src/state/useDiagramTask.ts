import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { errorText, native, snapshot } from '../services/bridge'
import { readImageFile } from '../services/imageFiles'
import type { DiagramScene, RecognizedDiagram } from '../services/diagramTypes'

interface DiagramSource {
  id: number
  name: string
  data: string
  size: number
  result?: DiagramScene
  error?: string
  processing?: boolean
}

// Visibility is independent of the batch lifetime. Results stay in source order
// and are committed together only after every source has succeeded.
export function useDiagramTask(
  submit: (sources: RecognizedDiagram[], cancelled: () => boolean) => Promise<void>,
  notifyFailure: (message: string) => void,
) {
  const open = ref(false),
    sources = ref<DiagramSource[]>([]),
    instruction = ref(''),
    error = ref(''),
    working = ref(false),
    reading = ref(false),
    stopping = ref(false),
    retained = ref(false),
    session = ref(0),
    seconds = ref(0)
  const busy = computed(() => working.value || reading.value || stopping.value)
  const processingCount = computed(() => sources.value.filter((s) => s.processing).length)
  const finishedCount = computed(() => sources.value.filter((s) => s.result || s.error).length)
  const failed = computed(() => sources.value.some((s) => s.error))
  const elapsed = computed(
    () => `${Math.floor(seconds.value / 60)}:${String(seconds.value % 60).padStart(2, '0')}`,
  )
  let generation = 0,
    nextId = 0,
    completion: Promise<void> | undefined,
    timer: ReturnType<typeof setInterval> | undefined
  function stopTimer() {
    clearInterval(timer)
    timer = undefined
  }
  watch(
    instruction,
    () => {
      for (const s of sources.value) {
        s.result = undefined
        s.error = undefined
      }
      error.value = ''
    },
    { flush: 'sync' },
  )
  function prepare(source: string) {
    generation++
    sources.value = source
      ? [{ id: ++nextId, name: '当前图片', data: source, size: Math.ceil(source.length * 0.75) }]
      : []
    error.value = ''
    seconds.value = 0
    retained.value = true
    session.value++
    open.value = true
  }
  async function add(files: File[], replaceIndex?: number) {
    if (busy.value || !files.length) return
    reading.value = true
    error.value = ''
    const ticket = generation
    try {
      const replacing = replaceIndex !== undefined
      if (replacing && files.length !== 1) throw new Error('更换时请选择一张图片')
      if (sources.value.length + files.length - (replacing ? 1 : 0) > 10)
        throw new Error('每批最多 10 张图片，请分批导入')
      const size = sources.value.reduce((sum, s, i) => sum + (i === replaceIndex ? 0 : s.size), 0)
      if (size + files.reduce((sum, f) => sum + f.size, 0) > 48 * 1024 * 1024)
        throw new Error('本批图片超过 48 MB，请分批导入')
      const additions: DiagramSource[] = []
      for (const file of files) {
        const data = await readImageFile(file)
        additions.push({ id: ++nextId, name: file.name || '剪贴板图片', data, size: file.size })
      }
      if (ticket !== generation) return
      if (replacing) sources.value.splice(replaceIndex, 1, ...additions)
      else sources.value.push(...additions)
    } catch (e) {
      if (ticket === generation) {
        error.value = errorText(e)
        if (!open.value) notifyFailure(error.value)
      }
    } finally {
      if (ticket === generation) reading.value = false
    }
  }
  function remove(index: number) {
    if (busy.value) return
    sources.value.splice(index, 1)
    error.value = ''
  }
  async function run(ensureConfig: () => Promise<void>) {
    if (busy.value || !sources.value.length) return
    working.value = true
    error.value = ''
    const ticket = ++generation,
      cancelled = () => ticket !== generation,
      requirements = instruction.value,
      started = Date.now()
    const batch = sources.value
    seconds.value = 0
    stopTimer()
    timer = setInterval(() => {
      seconds.value = Math.floor((Date.now() - started) / 1000)
    }, 1000)
    completion = (async () => {
      try {
        await ensureConfig()
        if (cancelled()) return
        const pending = batch.filter((s) => !s.result)
        for (const source of pending) source.error = undefined
        let cursor = 0
        async function worker() {
          while (!cancelled() && cursor < pending.length) {
            const source = pending[cursor++]!
            source.processing = true
            try {
              const scene = await native().RecognizeDiagram(source.data, requirements)
              if (!cancelled()) source.result = snapshot(scene)
            } catch (e) {
              if (!cancelled()) source.error = errorText(e)
            } finally {
              source.processing = false
            }
          }
        }
        await Promise.all(Array.from({ length: Math.min(3, pending.length) }, () => worker()))
        if (cancelled()) return
        if (batch.some((s) => !s.result))
          throw new Error(
            batch.length === 1 ? batch[0]!.error : '部分图片识别失败，请重试或移除失败图片',
          )
        await submit(
          batch.map((s) => ({ data: s.data, scene: snapshot(s.result!) })),
          cancelled,
        )
        if (!cancelled()) {
          retained.value = false
          open.value = false
        }
      } catch (e) {
        if (!cancelled()) {
          error.value = errorText(e)
          if (!open.value) notifyFailure(error.value)
        }
      } finally {
        if (!cancelled()) {
          working.value = false
          stopTimer()
        }
      }
    })()
    await completion
  }
  async function cancel() {
    generation++
    stopTimer()
    const requestActive = working.value
    working.value = false
    reading.value = false
    retained.value = false
    open.value = false
    if (!requestActive) return
    stopping.value = true
    const active = completion
    try {
      await native().CancelDiagramRecognition()
    } catch (e) {
      notifyFailure(errorText(e))
    } finally {
      // Wait for old workers to release backend slots before starting a new batch.
      await active
      stopping.value = false
    }
  }
  onBeforeUnmount(() => {
    void cancel()
  })
  return {
    open,
    sources,
    instruction,
    error,
    working,
    busy,
    retained,
    session,
    elapsed,
    processingCount,
    finishedCount,
    failed,
    prepare,
    add,
    remove,
    run,
    cancel,
    hide: () => {
      open.value = false
    },
    resume: () => {
      open.value = true
    },
  }
}
export type DiagramTask = ReturnType<typeof useDiagramTask>
