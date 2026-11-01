import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { errorText, native, snapshot } from '../services/bridge'
import type { DiagramScene } from '../services/diagramTypes'

// Visibility is independent of the request lifetime. Only explicit cancellation,
// a project switch, or application teardown invalidates an in-flight result.
export function useDiagramTask(
  submit: (data: string, scene: DiagramScene, cancelled: () => boolean) => Promise<void>,
  notifyFailure: (message: string) => void,
) {
  const open = ref(false),
    data = ref(''),
    instruction = ref(''),
    error = ref(''),
    working = ref(false),
    reading = ref(false),
    stopping = ref(false),
    retained = ref(false),
    session = ref(0),
    seconds = ref(0)
  const busy = computed(() => working.value || reading.value || stopping.value)
  const elapsed = computed(
    () => `${Math.floor(seconds.value / 60)}:${String(seconds.value % 60).padStart(2, '0')}`,
  )
  let generation = 0,
    result: DiagramScene | undefined,
    timer: ReturnType<typeof setInterval> | undefined
  function stopTimer() {
    clearInterval(timer)
    timer = undefined
  }
  watch(
    [data, instruction],
    () => {
      result = undefined
    },
    { flush: 'sync' },
  )
  function prepare(source: string) {
    generation++
    data.value = source
    error.value = ''
    result = undefined
    seconds.value = 0
    retained.value = true
    session.value++
    open.value = true
  }
  async function read(readFile: () => Promise<string>) {
    if (busy.value) return
    reading.value = true
    error.value = ''
    const ticket = generation
    try {
      const source = await readFile()
      if (ticket === generation) data.value = source
    } catch (e) {
      if (ticket === generation) {
        error.value = errorText(e)
        if (!open.value) notifyFailure(error.value)
      }
    } finally {
      if (ticket === generation) reading.value = false
    }
  }
  async function run(ensureConfig: () => Promise<void>) {
    if (busy.value || !data.value) return
    working.value = true
    error.value = ''
    const ticket = ++generation,
      cancelled = () => ticket !== generation,
      source = data.value,
      requirements = instruction.value,
      started = Date.now()
    seconds.value = 0
    stopTimer()
    timer = setInterval(() => {
      seconds.value = Math.floor((Date.now() - started) / 1000)
    }, 1000)
    try {
      await ensureConfig()
      if (cancelled()) return
      const recognized = result ?? (await native().RecognizeDiagram(source, requirements))
      if (cancelled()) return
      result = recognized
      await submit(source, snapshot(recognized), cancelled)
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
  }
  async function cancel() {
    generation++
    stopTimer()
    const requestActive = working.value
    working.value = false
    reading.value = false
    retained.value = false
    open.value = false
    result = undefined
    if (!requestActive) return
    stopping.value = true
    try {
      await native().CancelDiagramRecognition()
    } catch (e) {
      notifyFailure(errorText(e))
    } finally {
      stopping.value = false
    }
  }
  onBeforeUnmount(() => {
    void cancel()
  })
  return {
    open,
    data,
    instruction,
    error,
    working,
    busy,
    retained,
    session,
    elapsed,
    prepare,
    read,
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
