import { nextTick, onBeforeUnmount, onMounted, watch, type Ref } from 'vue'
import { errorText, native, snapshot } from '../services/bridge'
import type { Preferences } from '../types'
import { flushEditors } from './editorCommands'

export type SaveStatus = '' | 'saving' | 'saved' | 'error'

interface AutosaveWorkspace {
  preferences: Preferences
  ready: Ref<boolean>
  projectPath: Ref<string>
  projectOpened: Ref<number>
  contentVersion: Ref<number>
  saved: Ref<SaveStatus>
  markSaved(version: number, path: string, opened: number): void
  tell(text: string): void
}

export function useWorkspaceAutosave(w: AutosaveWorkspace) {
  let timer: ReturnType<typeof setTimeout> | undefined
  let running: Promise<void> | undefined
  let pending = false,
    paused = 0,
    closing = false,
    disposed = false
  let stopClose = () => {}

  function schedule() {
    clearTimeout(timer)
    if (pending && !paused && !disposed && w.ready.value)
      timer = setTimeout(() => void write().catch(() => {}), 650)
  }
  function write(): Promise<void> {
    if (running) return running
    if (!pending || !w.ready.value || disposed) return Promise.resolve()
    pending = false
    const path = w.projectPath.value,
      opened = w.projectOpened.value,
      version = w.contentVersion.value,
      preferences = snapshot(w.preferences)
    w.saved.value = 'saving'
    running = (async () => {
      try {
        await native().SavePreferences(preferences, path)
        if (path === w.projectPath.value && opened === w.projectOpened.value) {
          if (path) w.markSaved(version, path, opened)
          w.saved.value = pending ? 'saving' : 'saved'
        }
      } catch (error) {
        pending = true
        w.saved.value = 'error'
        w.tell(`${errorText(error)}；请重试保存或另存为。`)
        throw error
      } finally {
        running = undefined
        // A failure waits for another edit or an explicit retry; successful
        // writes serialize a newer snapshot instead of racing the old one.
        if (w.saved.value !== 'error') schedule()
      }
    })()
    return running
  }
  watch(
    w.preferences,
    () => {
      if (!w.ready.value) return
      pending = true
      w.saved.value = 'saving'
      schedule()
    },
    { deep: true },
  )

  async function flush() {
    flushEditors()
    await nextTick()
    clearTimeout(timer)
    // An edit made during the previous write still needs its own snapshot.
    if (running) await running
    while (pending) {
      await write()
      clearTimeout(timer)
    }
  }
  async function withPaused<T>(work: () => Promise<T>): Promise<T> {
    paused++
    clearTimeout(timer)
    try {
      // Manual save/open/new cannot change the destination of an older write.
      if (running) await running.catch(() => {})
      return await work()
    } finally {
      paused--
      schedule()
    }
  }
  async function close() {
    if (closing) return
    closing = true
    try {
      await flush()
      await native().CompleteClose()
    } catch {
      // write() already explains the failure. Keep the document editable.
    } finally {
      closing = false
    }
  }
  onMounted(() => {
    stopClose = window.runtime?.EventsOn('workspace:close', close) ?? (() => {})
  })
  onBeforeUnmount(() => {
    disposed = true
    clearTimeout(timer)
    stopClose()
  })
  return { flush, withPaused }
}
