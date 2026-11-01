import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch, type Ref } from 'vue'
import { errorText, native, snapshot } from '../services/bridge'
import type { Preferences } from '../types'
import { flushEditors } from './editorCommands'
import { projectPayload, projectPreferences } from './projectFile'
import type { SaveStatus } from './useWorkspaceAutosave'

interface ProjectWorkspace {
  preferences: Preferences
  ready: Ref<boolean>
  saved: Ref<SaveStatus>
  withPaused<T>(work: () => Promise<T>): Promise<T>
  flushPersistence(): Promise<void>
  action<T>(work: () => Promise<T>): Promise<T | undefined>
  tell(text: string, kind?: 'error' | 'success', path?: string): void
  replaced(): void
}

export function useProjectFiles(w: ProjectWorkspace) {
  const projectPath = ref(''),
    projectOpened = ref(0)
  const contentVersion = ref(0),
    savedVersion = ref(0)
  const projectUnsaved = computed(() => contentVersion.value !== savedVersion.value)
  const p = w.preferences
  watch(
    () => [p.text, p.layout, p.writing, p.images, p.tables, p.erasures, p.calculations],
    () => {
      if (w.ready.value) contentVersion.value++
    },
    { deep: true },
  )

  async function flush() {
    flushEditors()
    await nextTick()
  }
  async function save(chooseLocation: boolean): Promise<boolean> {
    await flush()
    const version = contentVersion.value
    const payload = projectPayload(snapshot(p))
    const path = chooseLocation
      ? await native().SaveProjectAs(payload)
      : await native().SaveProject(payload)
    if (!path) return false
    projectPath.value = path
    savedVersion.value = version
    w.saved.value = 'saved'
    w.tell('已保存到', 'success', path)
    return true
  }
  async function canReplace(): Promise<boolean> {
    await flush()
    if (projectPath.value) {
      await w.flushPersistence()
      return true
    }
    // A restored draft has no .ccf path, even before the first edit this session.
    const unsavedDraft =
      !projectPath.value &&
      (p.text.length > 0 || p.images.length > 0 || p.tables.length > 0 || p.erasures.length > 0)
    if (!projectUnsaved.value && !unsavedDraft) return true
    const choice = await native().ConfirmDiscardProject()
    if (choice === 'discard') return true
    return choice === 'save' && (await save(false))
  }
  async function replaced(path: string) {
    projectPath.value = path
    w.replaced()
    await nextTick()
    savedVersion.value = contentVersion.value
    projectOpened.value++
  }
  async function saveProjectAs() {
    if (!w.ready.value) return
    return w.action(() => w.withPaused(() => save(true)))
  }
  async function saveProject() {
    if (!w.ready.value) return
    return w.action(() => w.withPaused(() => save(false)))
  }
  async function openProject() {
    if (!w.ready.value) return
    await w.action(() =>
      w.withPaused(async () => {
        if (!(await canReplace())) return
        const file = await native().OpenProject()
        if (!file) return
        Object.assign(p, projectPreferences(p, file.project))
        await replaced(file.path)
        w.tell('已打开', 'success', file.path)
      }),
    )
  }
  async function newProject() {
    if (!w.ready.value) return
    await w.action(() =>
      w.withPaused(async () => {
        if (!(await canReplace())) return
        // Reset the backend path too: saving this document must never reuse the old file.
        await native().NewProject()
        Object.assign(p, { text: '', images: [], tables: [], erasures: [], calculations: [] })
        p.layout.pageLayouts = undefined
        await replaced('')
        w.tell('', 'success')
      }),
    )
  }
  async function revealProject(path: string) {
    try {
      await native().RevealProject(path)
    } catch (error) {
      w.tell(errorText(error))
    }
  }
  function shortcut(event: KeyboardEvent) {
    if (!(event.metaKey || event.ctrlKey) || event.altKey || event.repeat) return
    const key = event.key.toLowerCase()
    if (key === 's') {
      event.preventDefault()
      void (event.shiftKey ? saveProjectAs() : saveProject())
    } else if (!event.shiftKey && (key === 'o' || key === 'n')) {
      event.preventDefault()
      void (key === 'n' ? newProject() : openProject())
    }
  }
  onMounted(() => window.addEventListener('keydown', shortcut))
  onBeforeUnmount(() => window.removeEventListener('keydown', shortcut))
  return {
    projectPath,
    projectOpened,
    contentVersion,
    projectUnsaved,
    markSaved(version: number, path: string, opened: number) {
      if (projectPath.value === path && projectOpened.value === opened) savedVersion.value = version
    },
    saveProject,
    saveProjectAs,
    openProject,
    newProject,
    revealProject,
  }
}
