<script setup lang="ts">
import { computed, nextTick, ref, watch, watchEffect } from 'vue'
import CalculationImportDialog from './components/import/CalculationImportDialog.vue'
import { useCalculationImport } from './state/useCalculationImport'
import TableImportDialog from './components/import/TableImportDialog.vue'
import DiagramImportDialog from './components/import/DiagramImportDialog.vue'
import DiagramImportStatus from './components/import/DiagramImportStatus.vue'
import { useDiagramImport } from './state/useDiagramImport'
import { useImagePaste } from './state/useImagePaste'
import { useTableImport } from './state/useTableImport'
import { useWorkspace } from './state/useWorkspace'
import { useToolShortcuts } from './state/useToolShortcuts'
import AppIcon from './components/AppIcon.vue'
import AppHeader from './components/layout/AppHeader.vue'
import WorkspaceSidebar from './components/layout/WorkspaceSidebar.vue'
import LayoutInspector from './components/editor/LayoutInspector.vue'
import DocumentCanvas from './components/editor/DocumentCanvas.vue'
import EditorToolbar from './components/editor/EditorToolbar.vue'
import DocumentWarnings from './components/editor/DocumentWarnings.vue'
import DevicePage from './components/device/DevicePage.vue'
import JobBar from './components/JobBar.vue'
import UploadDialog from './components/upload/UploadDialog.vue'
import UpdateDialog from './components/update/UpdateDialog.vue'
import { useUpdater } from './state/useUpdater'
import { onMenuAction, setWindowTitle } from './services/bridge'
import { t } from './i18n'
import { flushEditors } from './state/editorCommands'
import type { PageAction } from './state/documentPages'
import type { EraseMode, ImageEditTool, TableInsertOptions } from './types'
const w = useWorkspace()
const updater = useUpdater(w)
const {
  preferences,
  document,
  printer,
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
  page,
  canOutput,
  uploadOpen,
  outputReasons,
  projectPath,
  projectUnsaved,
} = w
// The native menu only sends an action name; every decision — including
// whether the action is available, which the menu cannot show — is made here.
function menuAction(action: string) {
  if (updater.installing.value) return
  if (action === 'new') void w.newProject()
  else if (action === 'save') void w.saveProject()
  else if (action === 'saveAs') void w.saveProjectAs()
  else if (action === 'open') void w.openProject()
  else if (action === 'reveal') void w.revealProject(projectPath.value)
  else if (action === 'exportPage') void w.exportPage(false)
  else if (action === 'exportAll') void w.exportPage(true)
  else if (action === 'upload') void w.upload()
  else if (action === 'ruled') ruled.value = !ruled.value
  else if (action === 'zoomIn') zoom.value = Math.min(150, zoom.value + 10)
  else if (action === 'zoomOut') zoom.value = Math.max(30, zoom.value - 10)
  else if (action === 'zoomReset') zoom.value = 100
  else if (action === 'tab:write') tab.value = 'write'
  else if (action === 'tab:device') tab.value = 'device'
  else if (action === 'theme:light') preferences.theme = 'light'
  else if (action === 'theme:dark') preferences.theme = 'dark'
  else if (action.startsWith('lang:'))
    preferences.language = action.slice(5) as typeof preferences.language
  else if (action === 'guide') void w.revealProject('https://github.com/ICEIU-I/BambuScribe-Write')
  else if (action === 'checkUpdate') void updater.check()
}
const editor = ref<InstanceType<typeof DocumentCanvas>>()
const tableImport = useTableImport(w, editor)
const calculationImport = useCalculationImport(w, editor)
const diagramImport = useDiagramImport(w, editor)
const imagePaste = useImagePaste(w, editor)
function insertTable(options: TableInsertOptions) {
  imageTool.value = 'select'
  editor.value?.insertTable(options)
}
// A project switch remounts the canvas to clear selections and undo history.
watch(
  () => w.projectOpened.value,
  async () => {
    const isNew = !projectPath.value
    if (isNew) imageTool.value = 'select'
    tableImport.open.value = false
    await nextTick()
    editor.value?.jump(0)
    if (isNew) editor.value?.focus()
  },
)
onMenuAction(menuAction)
const sidebarOpen = ref(true)
const settingsOpen = ref(false)
const projectTitle = computed(
  () =>
    projectPath.value
      .split('/')
      .pop()
      ?.replace(/\.ccf$/i, '') ||
    preferences.text.split('\n')[0]?.trim() ||
    t('app.untitled'),
)
// The same name appears in the native window and the page header. Reading t
// here also refreshes the native title when the language changes.
watchEffect(() => {
  setWindowTitle(`${projectTitle.value}${projectUnsaved.value ? ` — ${t('app.modified')}` : ''}`)
})
async function toggleSidebar(open: boolean) {
  sidebarOpen.value = open
  await nextTick()
  window.document
    .querySelector<HTMLButtonElement>(open ? '.sidebar-brand button' : '.sidebar-toggle')
    ?.focus()
}
async function jumpToPage(index: number) {
  tab.value = 'write'
  await nextTick()
  editor.value?.jump(index)
}
async function changePage(index: number, action: PageAction, destination?: number) {
  const previous = document.value
  flushEditors()
  await nextTick()
  if (
    !previous ||
    previous !== document.value ||
    !ready.value ||
    busy.value ||
    dirty.value ||
    loading.value
  ) {
    w.tell('排版正在更新，请稍后重试。')
    return
  }
  tab.value = 'write'
  try {
    if (editor.value?.changePage(index, action, destination)) {
      const target =
        action === 'move'
          ? destination!
          : action === 'delete'
            ? Math.max(0, Math.min(index, previous.pages.length - 2))
            : index + 1
      pageIndex.value = target
      const stop = watch(document, async (doc) => {
        if (!doc || doc === previous) return
        stop()
        await nextTick()
        editor.value?.jump(Math.min(target, doc.pages.length - 1))
      })
    }
  } catch (error) {
    w.tell(error instanceof Error ? error.message : String(error))
  }
}
const ruled = ref(true),
  zoom = ref(100),
  imageTool = ref<ImageEditTool>('select'),
  eraseMode = ref<EraseMode>('box'),
  eraserRadius = ref(0.018)
useToolShortcuts(
  imageTool,
  computed(() => tab.value === 'write' && ready.value),
  () => true,
)
// The menu mirrors a few checkable bits; push them whenever they change.
watch(
  () => [ruled.value, preferences.theme, tab.value, zoom.value] as const,
  () => {
    void w.syncMenu({ ruled: ruled.value, device: tab.value === 'device', zoom: zoom.value })
  },
)
</script>
<template>
  <div class="application-shell">
    <WorkspaceSidebar
      v-show="sidebarOpen"
      v-model:tab="tab"
      v-model:theme="preferences.theme"
      :pages="document?.pages ?? []"
      :index="pageIndex"
      :disabled="!ready || busy"
      :actions-disabled="!ready || busy || dirty || loading"
      :update-version="updater.availableVersion.value"
      @collapse="toggleSidebar(false)"
      @new="w.newProject"
      @open="w.openProject"
      @save-as="w.saveProjectAs"
      @check-update="updater.check"
      @page="jumpToPage"
      @delete-page="changePage($event, 'delete')"
      @copy-page="changePage($event, 'copy')"
      @blank-page="changePage($event, 'blank')"
      @move-page="(from, to) => changePage(from, 'move', to)"
    />
    <div class="workspace-main">
      <UpdateDialog :updater="updater" />
      <AppHeader
        :tab="tab"
        :title="projectTitle"
        :modified="projectUnsaved"
        :sidebar-open="sidebarOpen"
        :can-output="canOutput"
        :output-reasons="outputReasons"
        :printer="printer"
        :pages="document?.pages.length ?? 0"
        :characters="document?.characterCount ?? 0"
        @sidebar="toggleSidebar(true)"
        @upload="w.upload"
        @export="w.exportPage($event)"
      />
      <CalculationImportDialog :task="calculationImport" />
      <TableImportDialog
        :open="tableImport.open.value"
        :tables="preferences.tables"
        :anchor="tableImport.anchor"
        :preview="tableImport.preview"
        :submit="tableImport.submit"
        @close="tableImport.open.value = false"
      />
      <DiagramImportDialog
        :task="diagramImport.task"
        :replacing="
          !!diagramImport.replacement.value &&
          diagramImport.task.sources.value.length === 1 &&
          diagramImport.task.sources.value[0]?.data === diagramImport.replacement.value.data
        "
      />
      <UploadDialog
        :open="uploadOpen"
        :text="preferences.text"
        :document="document"
        :writing="preferences.writing"
        :index="pageIndex"
        :ready="ready"
        :loading="loading"
        :dirty="dirty"
        :busy="busy || w.selectingDevice.value"
        :state="printer"
        :notice="notice"
        :submit="w.uploadSelected"
        :cancel="() => w.cancelUpload(w.uploadDeviceId.value)"
        :devices="w.devices.value"
        :device-id="w.selectedDeviceId.value"
        @device="w.selectDevice"
        @close="uploadOpen = false"
      />
      <div
        v-if="notice"
        class="notice"
        :class="noticeKind"
        :role="noticeKind === 'error' ? 'alert' : 'status'"
      >
        <AppIcon :name="noticeKind === 'error' ? 'alert' : 'check'" :size="17" /><span
          class="notice-text"
          >{{ notice
          }}<button
            v-if="noticePath"
            class="path-link"
            :title="`在访达中显示 ${noticePath}`"
            @click="w.revealProject(noticePath)"
          >
            {{ noticePath }}
          </button></span
        ><button class="icon-button" aria-label="关闭提示" @click="notice = ''">
          <AppIcon name="close" :size="15" />
        </button>
      </div>
      <main
        id="write-panel"
        v-show="tab === 'write'"
        class="writing-workspace"
        role="tabpanel"
        aria-labelledby="write-tab"
      >
        <EditorToolbar
          :settings="preferences.layout"
          :settings-open="settingsOpen"
          :disabled="!ready"
          v-model:tool="imageTool"
          v-model:erase-mode="eraseMode"
          v-model:eraser-radius="eraserRadius"
          v-model:ruled="ruled"
          v-model:zoom="zoom"
          @insert-table="insertTable"
          @import-table="tableImport.open.value = true"
          @import-diagram="diagramImport.show"
          @import-calculation="calculationImport.show()"
          @settings="settingsOpen = !settingsOpen"
          @undo="editor?.undo()"
          @redo="editor?.redo()"
        />
        <div v-if="page?.missing.length" class="document-alert" role="alert">
          <AppIcon name="alert" :size="16" />第 {{ pageIndex + 1 }} 页缺字：{{
            [...new Set(page.missing.map((item) => item.char))].slice(0, 12).join(' ')
          }}。请修改标记的文字后导出。
        </div>
        <div v-if="page?.issues?.length" class="document-alert" role="alert">
          <AppIcon name="alert" :size="16" />{{ page.issues[0] }}
        </div>
        <DocumentWarnings
          :key="w.projectOpened.value"
          :warnings="page?.issues?.length ? [] : (page?.warnings ?? [])"
          :images="
            preferences.images.filter((image) =>
              page?.images?.some((placed) => placed.id === image.id),
            )
          "
        />
        <div class="editor-body">
          <DocumentCanvas
            :key="w.projectOpened.value"
            ref="editor"
            v-model="preferences.text"
            v-model:page-layouts="preferences.layout.pageLayouts"
            v-model:images="preferences.images"
            v-model:tables="preferences.tables"
            v-model:erasures="preferences.erasures"
            v-model:calculations="preferences.calculations"
            @edit-calculation="calculationImport.show($event)"
            v-model:index="pageIndex"
            :document="document"
            v-model:zoom="zoom"
            :guides="false"
            :ruled="ruled"
            :tool="imageTool"
            :erase-mode="eraseMode"
            :eraser-radius="eraserRadius"
            :active="tab === 'write'"
            :disabled="!ready"
            @notice="w.tell($event, 'success')"
            @import-image="imagePaste.paste"
          />
          <LayoutInspector
            v-if="settingsOpen"
            :settings="preferences.layout"
            @change="preferences.layout = $event"
            @close="settingsOpen = false"
          />
        </div>
      </main>
      <div
        id="device-panel"
        v-show="tab === 'device'"
        role="tabpanel"
        aria-labelledby="device-tab"
        class="device-panel-wrap"
      >
        <DevicePage :workspace="w" />
      </div>
      <DiagramImportStatus :task="diagramImport.task" />
      <JobBar
        :state="printer"
        :save-status="saved"
        :project-path="projectPath"
        :device-name="w.selectedDevice.value?.name"
        @cancel="w.cancelUpload()"
        @device="tab = 'device'"
      />
    </div>
  </div>
</template>
