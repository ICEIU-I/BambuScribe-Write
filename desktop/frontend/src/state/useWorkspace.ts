import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  reactive,
  ref,
  shallowRef,
  watch,
} from 'vue'
import { errorText, native, onPrinterState, snapshot } from '../services/bridge'
import { defaultPreferences, defaultPrinter } from './defaults'
import { currentLanguage, setLanguage } from '../i18n'
import type { MenuState } from '../types'
import { outputAvailability } from './outputAvailability'
import { normalizePreferences } from './normalizePreferences'
import { usePrinterConnection } from './usePrinterConnection'
import { useProjectFiles } from './useProjectFiles'
import { useWorkspaceAutosave, type SaveStatus } from './useWorkspaceAutosave'
import { flushEditors } from './editorCommands'
import type { Document, Preferences, PrinterSettings } from '../types'

export function useWorkspace() {
  const preferences = reactive<Preferences>(defaultPreferences())
  const document = shallowRef<Document | null>(null)
  const printer = ref(defaultPrinter())
  const connection = ref<PrinterSettings>({ ip: '', serial: '', hasAccessCode: false })
  const ready = ref(false),
    loading = ref(false),
    dirty = ref(true),
    busy = ref(false)
  const pageIndex = ref(0),
    tab = ref<'write' | 'device'>('write')
  const notice = ref(''),
    noticeKind = ref<'error' | 'success'>('error'),
    noticePath = ref(''),
    saved = ref<SaveStatus>('')
  const page = computed(() => document.value?.pages[pageIndex.value] ?? null)
  const outputReasons = computed(() =>
    outputAvailability({
      document: document.value,
      index: pageIndex.value,
      ready: ready.value,
      loading: loading.value,
      dirty: dirty.value,
      busy: busy.value,
    }),
  )
  const uploadOpen = ref(false)
  const uploadDeviceId = ref('')
  const canOutput = computed(
    () =>
      ready.value &&
      !busy.value &&
      !loading.value &&
      !dirty.value &&
      !!document.value?.pages.length,
  )
  let previewTimer: ReturnType<typeof setTimeout> | undefined
  let poll: ReturnType<typeof setInterval> | undefined
  let sequence = 0,
    stopEvents = () => {}

  watch(
    () => preferences.theme,
    (theme) => {
      window.document.documentElement.dataset.theme = theme
    },
    { immediate: true },
  )

  function tell(text: string, kind: 'error' | 'success' = 'error', path = '') {
    notice.value = text
    noticeKind.value = kind
    noticePath.value = path
  }
  const printerConnection = usePrinterConnection(connection, printer, tell)
  async function action<T>(work: () => Promise<T>): Promise<T | undefined> {
    if (busy.value) return
    busy.value = true
    notice.value = ''
    try {
      return await work()
    } catch (e) {
      tell(errorText(e))
    } finally {
      busy.value = false
    }
  }
  const projectFiles = useProjectFiles({
    preferences,
    ready,
    saved,
    withPaused: (work) => autosave.withPaused(work),
    flushPersistence: () => autosave.flush(),
    action,
    tell,
    replaced() {
      sequence++
      clearTimeout(previewTimer)
      document.value = null
      dirty.value = true
      pageIndex.value = 0
      tab.value = 'write'
      uploadOpen.value = false
    },
  })
  const autosave = useWorkspaceAutosave({
    preferences,
    ready,
    saved,
    tell,
    ...projectFiles,
  })
  async function preview(version: number) {
    loading.value = true
    try {
      const api = native()
      const result = preferences.erasures.length
        ? await api.PreviewWithErasures(
            preferences.text,
            snapshot(preferences.layout),
            snapshot(preferences.images),
            snapshot(preferences.tables),
            snapshot(preferences.erasures),
          )
        : preferences.tables.length
          ? await api.PreviewWithObjects(
              preferences.text,
              snapshot(preferences.layout),
              snapshot(preferences.images),
              snapshot(preferences.tables),
            )
          : await api.PreviewWithImages(
              preferences.text,
              snapshot(preferences.layout),
              snapshot(preferences.images),
            )
      if (version !== sequence) return
      document.value = result
      if (result.erasures && preferences.erasures.some((e) => !e.cuts))
        preferences.erasures = result.erasures
      dirty.value = false
      pageIndex.value = Math.min(pageIndex.value, Math.max(0, result.pages.length - 1))
    } catch (e) {
      if (version === sequence) tell(errorText(e))
    } finally {
      if (version === sequence) loading.value = false
    }
  }
  watch(
    () => [
      preferences.text,
      preferences.layout,
      preferences.images,
      preferences.tables,
      preferences.erasures,
    ],
    () => {
      if (!ready.value) return
      dirty.value = true
      const version = ++sequence
      clearTimeout(previewTimer)
      previewTimer = setTimeout(() => preview(version), 65)
    },
    { deep: true },
  )
  // The webview resolves "system" against the OS locale and tells the backend,
  // which renders its own messages and the menu bar in that language.
  watch(
    () => preferences.language,
    () => {
      if (ready.value) applyLanguage()
    },
  )
  // The native menu mirrors a few checkable bits of the editor; pushing them
  // rebuilds the menu so its checkmarks match what the user sees.
  async function syncMenu(state: Pick<MenuState, 'ruled' | 'device' | 'zoom'>) {
    await native().SetMenuState({
      ...state,
      dark: preferences.theme === 'dark',
      lang: preferences.language,
    })
  }
  function applyLanguage() {
    setLanguage(preferences.language)
    void native().SetLanguage(preferences.language, currentLanguage())
  }
  onMounted(async () => {
    try {
      const boot = await native().Bootstrap()
      Object.assign(preferences, normalizePreferences(boot.preferences))
      projectFiles.projectPath.value = boot.projectPath || ''
      if (projectFiles.projectPath.value) saved.value = 'saved'
      applyLanguage()
      printerConnection.loadPrinters(boot)
      stopEvents = onPrinterState((state) => {
        printerConnection.observePrinter(state)
      })
      if (boot.warning) tell(boot.warning)
      // Flush watchers before enabling persistence, so opening doesn't rewrite the draft.
      await Promise.resolve()
      ready.value = true
      // Connection runs alongside preview and never holds the editor's busy flag.
      void printerConnection.autoConnectSaved()
      await preview(++sequence)
      poll = setInterval(async () => {
        try {
          await printerConnection.refreshPrinters()
        } catch {
          /* Native events remain primary. */
        }
      }, 2000)
    } catch (e) {
      tell(errorText(e))
    }
  })
  onBeforeUnmount(() => {
    clearTimeout(previewTimer)
    clearInterval(poll)
    stopEvents()
  })

  async function exportPage(all = false) {
    flushEditors()
    await nextTick()
    const reason = all ? outputReasons.value.all : outputReasons.value.current
    if (reason) {
      tell(reason)
      return
    }
    if (!document.value) return
    await action(async () => {
      const path = all
        ? await native().ExportAll(document.value!.id, snapshot(preferences.writing))
        : await native().ExportPage(
            document.value!.id,
            pageIndex.value,
            snapshot(preferences.writing),
          )
      if (path) tell('书写文件已导出。', 'success')
    })
  }
  async function upload() {
    flushEditors()
    await nextTick()
    if (!canOutput.value) {
      tell(outputReasons.value.all)
      return
    }
    notice.value = ''
    uploadOpen.value = true
  }
  async function uploadSelected(prefix: string, index: number | null): Promise<string | undefined> {
    if (printerConnection.selectingDevice.value) {
      tell('请等待实例切换完成。')
      return
    }
    const reasons = outputAvailability({
      document: document.value,
      index: index ?? pageIndex.value,
      ready: ready.value,
      loading: loading.value,
      dirty: dirty.value,
      busy: busy.value,
    })
    const reason = index === null ? reasons.all : reasons.current
    if (reason) {
      tell(reason)
      return
    }
    if (!document.value) return
    if (!printer.value.connected) {
      tab.value = 'device'
      tell('请先连接打印机。')
      return
    }
    const writing = snapshot(preferences.writing)
    uploadDeviceId.value = printerConnection.selectedDeviceId.value
    const id = await action(() =>
      native().UploadDevicePages(
        uploadDeviceId.value,
        document.value!.id,
        prefix,
        index === null ? [] : [index],
        writing,
      ),
    )
    return id
  }
  async function cancelUpload(deviceId = printerConnection.selectedDeviceId.value) {
    try {
      await native().CancelDeviceUpload(deviceId)
    } catch (e) {
      tell(errorText(e))
    }
  }
  return {
    ...projectFiles,
    preferences,
    document,
    printer,
    connection,
    ...printerConnection,
    ready,
    loading,
    dirty,
    busy,
    pageIndex,
    tab,
    notice,
    noticeKind,
    noticePath,
    saved,
    syncMenu,
    page,
    canOutput,
    uploadOpen,
    uploadDeviceId,
    outputReasons,
    tell,
    action,
    exportPage,
    upload,
    uploadSelected,
    cancelUpload,
  }
}
