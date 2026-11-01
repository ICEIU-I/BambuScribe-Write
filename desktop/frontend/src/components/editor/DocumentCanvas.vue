<script setup lang="ts">
import type { CalculationRecord } from '../../services/calculationTypes'
import FormulaEditDialog from './FormulaEditDialog.vue'
import { calculationAt, formulaRanges } from '../../state/calculationText'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type {
  Document,
  PageErasure,
  PageLayout,
  EraseMode,
  ImageAsset,
  ImageEditTool,
  TableAsset,
  TableSelection,
  TableInsertOptions,
} from '../../types'
import { usePaperInput } from '../../state/usePaperInput'
import PaperPage from './PaperPage.vue'
import TableSettings from '../tables/TableSettings.vue'
import { useTableClipboard } from '../../state/useTableClipboard'
import { useCanvasZoom } from '../../state/useCanvasZoom'
import { useDocumentEditing } from '../../state/useDocumentEditing'
import { paperPosition, type PaperPosition } from '../../state/paperGeometry'
import { createFreeTextBox } from '../../state/freeTextBox'
import { flushEditors } from '../../state/editorCommands'
import { placedObjects, keepObjectAnchor } from '../../state/objectLayout'
import { usePaperMarquee } from '../../state/usePaperMarquee'
import { useSelectionDrag } from '../../state/useSelectionDrag'
import { useSelectionClipboard } from '../../state/useSelectionClipboard'
import { provideImportedTableDrag } from '../../state/useImportedTableDrag'
import type { PageAction } from '../../state/documentPages'
const props = withDefaults(
  defineProps<{
    calculations?: CalculationRecord[]
    modelValue: string
    pageLayouts?: PageLayout[]
    tables?: TableAsset[]
    erasures?: PageErasure[]
    images: ImageAsset[]
    document: Document | null
    index: number
    zoom: number
    guides: boolean
    ruled: boolean
    tool: ImageEditTool
    eraseMode: EraseMode
    eraserRadius: number
    active: boolean
    disabled: boolean
  }>(),
  { tables: () => [], erasures: () => [], calculations: () => [] },
)
const emit = defineEmits<{
  'update:modelValue': [string]
  'update:calculations': [CalculationRecord[] | undefined]
  editCalculation: [number]
  'update:pageLayouts': [PageLayout[] | undefined]
  'update:images': [ImageAsset[]]
  'update:tables': [TableAsset[]]
  'update:erasures': [PageErasure[]]
  'update:index': [number]
  'update:zoom': [number]
  notice: [string]
  importImage: [File]
}>()
const input = ref<HTMLTextAreaElement>(),
  viewport = ref<HTMLDivElement>(),
  stack = ref<HTMLDivElement>(),
  caretElement = ref<HTMLDivElement>()
let imagePastePosition: PaperPosition | undefined
let importedTarget: { id: string; page: number } | undefined
watch(
  () => props.document?.id,
  async () => {
    if (!importedTarget || !props.document) return
    const page = props.document.pages.findIndex(
      (p) =>
        p.tables?.some((t) => t.id === importedTarget?.id) ||
        p.images?.some((i) => i.id === importedTarget?.id),
    )
    if (page < 0) return
    const target = { ...importedTarget, page }
    importedTarget = undefined
    await nextTick()
    stack.value?.querySelector(`[data-paper="${target.page}"]`)?.scrollIntoView({ block: 'start' })
    emit('update:index', target.page)
  },
)
const editing = useDocumentEditing(
  props,
  input,
  (text, images, tables, erasures, pageLayouts, calculations) => {
    emit('update:calculations', calculations)
    emit('update:modelValue', text)
    emit('update:images', images)
    emit('update:tables', tables)
    emit('update:erasures', erasures)
    emit('update:pageLayouts', pageLayouts)
  },
  (a, b) => editor.select(a, b),
  (message) => emit('notice', message),
  {
    location: () =>
      imagePastePosition ??
      (focused.value && caret.value.page === props.index ? caret.value : { page: props.index }),
    pasted: (image) => {
      imagePastePosition = { page: image.page, x: image.x + 2, y: image.y + 2 }
      emit('update:index', image.page)
    },
  },
)
const { imageId: selectedImageId, regionId: selectedTextId, tableId: selectedTableId } = editing
const tableSelection = ref<TableSelection | null>(null),
  tableSettingsOpen = ref(false),
  newTextBoxId = ref('')
const placedTables = computed(() => placedObjects(props.tables, props.document))
provideImportedTableDrag(
  () => props.tables,
  () => props.document,
  (table) => editing.changeTable(keepObjectAnchor(table, props.tables, placedTables.value), true),
)
const placedImages = computed(() => placedObjects(props.images, props.document))
const changeTable = (table: TableAsset) =>
  editing.changeTable(keepObjectAnchor(table, props.tables, placedTables.value))
const changeImage = (image: ImageAsset) =>
  editing.changeImage(keepObjectAnchor(image, props.images, placedImages.value))
const selectedTable = computed(() => {
  return placedTables.value.find((t) => t.id === selectedTableId.value)
})
const selectedTablePreview = computed(() =>
  props.document?.pages.flatMap((p) => p.tables ?? []).find((t) => t.id === selectedTableId.value),
)
watch(selectedTableId, () => {
  if (selectedTableId.value !== newTextBoxId.value) newTextBoxId.value = ''
  tableSelection.value = null
  tableSettingsOpen.value = false
})
const tableClipboard = useTableClipboard(
  props,
  () => input.value,
  () =>
    imagePastePosition ??
    (focused.value
      ? caret.value
      : selectedTable.value
        ? {
            page: selectedTable.value.page,
            x: selectedTable.value.x + 2,
            y: selectedTable.value.y + 2,
          }
        : { page: props.index }),
  (table) => {
    editing.addTable(table)
    imagePastePosition = { page: table.page, x: table.x + 2, y: table.y + 2 }
    emit('update:index', table.page)
    input.value?.blur()
  },
  (message) => emit('notice', message),
)
const editor = usePaperInput(
  input,
  () => props.modelValue,
  () => props.document,
  editing.changeText,
)
const { start, end, caret, focused, composition } = editor
const marquee = usePaperMarquee(props, stack, viewport, (event) => {
  if (stack.value) editor.placeCaret(event, stack.value)
})
const selectionDrag = useSelectionDrag(
  props,
  stack,
  viewport,
  marquee.selection,
  editing.changeObjects,
  (page) => emit('update:index', page),
)
const selectionClipboard = useSelectionClipboard(
  props,
  () => marquee.selection.value,
  () => input.value,
  () => imagePastePosition ?? { page: props.index },
  editing.addObjects,
  (position) => {
    marquee.clear()
    imagePastePosition = position
    emit('update:index', position.page)
  },
  (message) => emit('notice', message),
)
function copy(event: ClipboardEvent) {
  if ((event.target as Element | null)?.closest?.('dialog,[role="dialog"]')) return
  if (!selectionClipboard.copy(event)) editing.copy(event)
}
function cut(event: ClipboardEvent) {
  if ((event.target as Element | null)?.closest?.('dialog,[role="dialog"]')) return
  editing.cut(event)
}
function keydown(event: KeyboardEvent) {
  if (event.key === 'Escape') marquee.clear()
  editing.globalKeydown(event)
}
function capturePointerDown(event: PointerEvent) {
  if (selectionDrag.begin(event)) return
  insertTextBox(event)
}
function lostCapture() {
  marquee.lostCapture()
  selectionDrag.lostCapture()
}
useCanvasZoom(
  viewport,
  () => props.zoom,
  (zoom) => emit('update:zoom', zoom),
)
const scale = computed(() => ((96 / 25.4) * props.zoom) / 100)
const width = computed(() => (props.document?.settings.paperWidth ?? 163) * scale.value)
const pageHeight = computed(() => (props.document?.settings.paperHeight ?? 240) * scale.value)
const imagePreviews = computed(
  () => props.document?.pages.flatMap((page) => page.images ?? []) ?? [],
)
watch(
  () => props.index,
  (index) => {
    if (imagePastePosition?.page !== index) imagePastePosition = undefined
  },
)
watch(
  () => placedTables.value.find((table) => table.id === selectedTableId.value)?.page,
  (page) => {
    if (page !== undefined) emit('update:index', page)
  },
)
watch(
  () => placedImages.value.find((image) => image.id === selectedImageId.value)?.page,
  (page) => {
    if (page !== undefined) emit('update:index', page)
  },
)
const caretStyle = computed(() => ({
  left: `calc(50% - ${width.value / 2}px + ${caret.value.x * scale.value}px)`,
  top: `${32 + caret.value.page * (pageHeight.value + 30) + caret.value.y * scale.value}px`,
  height: `${
    Math.min(
      props.document?.settings.fontSize ?? 6,
      props.document?.pages[caret.value.page]?.lineHeight ?? 6,
    ) * scale.value
  }px`,
}))
watch(
  () => [caret.value.index, caret.value.page, caret.value.y, props.document?.id],
  async () => {
    if (!focused.value) return
    emit('update:index', caret.value.page)
    await nextTick()
    const rect = caretElement.value?.getBoundingClientRect(),
      view = viewport.value?.getBoundingClientRect()
    if (rect && view && (rect.top < view.top + 20 || rect.bottom > view.bottom - 20))
      caretElement.value?.scrollIntoView({ block: 'nearest' })
  },
)
function jump(page: number) {
  stack.value
    ?.querySelector(`[data-paper="${page}"]`)
    ?.scrollIntoView({ block: 'start', behavior: 'smooth' })
  emit('update:index', page)
}
function pointerDown(event: PointerEvent) {
  if (event.button !== 0 || props.disabled || !props.active || props.tool !== 'select') return
  flushEditors()
  editing.deselect()
  if (stack.value && props.document)
    imagePastePosition = paperPosition(
      stack.value,
      { x: event.clientX, y: event.clientY },
      props.document.settings.paperWidth,
    )
  if (marquee.begin(event)) return
  if (stack.value) editor.pointerDown(event, stack.value)
}
function insertTextBox(event: PointerEvent) {
  if (
    props.tool !== 'text' ||
    props.disabled ||
    !props.active ||
    !props.document ||
    event.button !== 0 ||
    !stack.value
  )
    return
  const target = event.target as HTMLElement
  if (!target.closest('[data-paper]') || target.closest('input,textarea,button,.manual-text-box'))
    return
  const position = paperPosition(
    stack.value,
    { x: event.clientX, y: event.clientY },
    props.document.settings.paperWidth,
  )
  if (!position) return
  event.preventDefault()
  event.stopPropagation()
  if (props.tables.length >= 50) {
    emit('notice', '表格和文本框共最多 50 个，请拆分文档。')
    return
  }
  flushEditors()
  const box = createFreeTextBox(props.document.settings, position)
  editing.addTable(box)
  newTextBoxId.value = box.id
  imagePastePosition = { page: box.page, x: box.x + 2, y: box.y + 2 }
  emit('update:index', box.page)
}
function selectTable(id: string) {
  marquee.clear()
  flushEditors()
  imagePastePosition = undefined
  editing.selectTable(id)
}
function selectImage(id: string, point?: { x: number; y: number }) {
  marquee.clear()
  flushEditors()
  imagePastePosition = undefined
  editing.select(id, point)
}
function paste(event: ClipboardEvent) {
  if ((event.target as Element | null)?.closest?.('dialog,[role="dialog"]')) return
  if (selectionClipboard.paste(event)) return
  if (tableClipboard.paste(event)) return
  if (editing.paste(event)) return
  if (props.disabled || !props.active || event.defaultPrevented) return
  const file =
    [...(event.clipboardData?.items ?? [])]
      .find((item) => item.type.startsWith('image/'))
      ?.getAsFile() ??
    [...(event.clipboardData?.files ?? [])].find((item) => item.type.startsWith('image/'))
  if (file) {
    event.preventDefault()
    emit('importImage', file)
    return
  }
  // Plain text lands in the paper's textarea, which repairs the spacing a
  // rich-text source flattened on its way to the clipboard.
  if (event.target === input.value) editing.notePaste()
}
let scrollFrame = 0
function scroll() {
  cancelAnimationFrame(scrollFrame)
  scrollFrame = requestAnimationFrame(() => {
    if (!viewport.value || !props.document) return
    const page = Math.max(
      0,
      Math.min(
        props.document.pages.length - 1,
        Math.floor(
          (viewport.value.scrollTop + viewport.value.clientHeight * 0.35) / (pageHeight.value + 30),
        ),
      ),
    )
    emit('update:index', page)
  })
}
onMounted(() => {
  window.addEventListener('copy', copy)
  window.addEventListener('cut', cut)
  window.addEventListener('paste', paste)
  window.addEventListener('keydown', keydown, true)
})
onBeforeUnmount(() => {
  cancelAnimationFrame(scrollFrame)
  window.removeEventListener('copy', copy)
  window.removeEventListener('cut', cut)
  window.removeEventListener('paste', paste)
  window.removeEventListener('keydown', keydown, true)
})
const formulaEdit = ref<{ source: string; start: number; end: number; original: string } | null>(
  null,
)
function saveFormula(source: string) {
  const current = formulaEdit.value
  if (current && current.original === props.modelValue) {
    const opening = current.original.slice(current.start, current.start + 2)
    const closing = current.original.slice(current.end - 2, current.end)
    editing.replaceFormula(current.start, current.end, `${opening}${source}${closing}`)
    formulaEdit.value = null
  } else {
    formulaEdit.value = null
    emit('notice', '正文已修改，请重新打开公式')
  }
}
function editFormula(event: MouseEvent) {
  if (props.disabled || props.tool !== 'select' || !stack.value || !props.document) return
  const pos = paperPosition(
    stack.value,
    { x: event.clientX, y: event.clientY },
    props.document.settings.paperWidth,
  )
  if (!pos) return
  const formula = props.document.pages[pos.page]?.formulas?.find(
    (f) => pos.x >= f.x && pos.x <= f.x + f.width && pos.y >= f.y && pos.y <= f.y + f.height,
  )
  if (!formula) return
  if (calculationAt(props.modelValue, props.calculations ?? [], formula.start)) {
    input.value?.blur()
    emit('editCalculation', formula.start)
  } else {
    const range = formulaRanges(props.modelValue).find(
      (r) => formula.start >= r.start && formula.start < r.end,
    )
    if (range) {
      input.value?.blur()
      formulaEdit.value = { ...range, original: props.modelValue }
    }
  }
}
defineExpose({
  textAnchor: () => {
    const page = props.document?.pages[props.index]
    const from =
      caret.value.page === props.index
        ? start.value
        : (page?.carets[0]?.index ?? props.modelValue.length)
    return { from, to: caret.value.page === props.index ? end.value : from }
  },
  replaceGenerated: editing.replaceGenerated,
  jump,
  changePage: (index: number, action: PageAction, destination?: number) => {
    if (!editing.changePage(index, action, destination)) return false
    marquee.clear()
    tableSelection.value = null
    tableSettingsOpen.value = false
    imagePastePosition = undefined
    importedTarget = undefined
    input.value?.blur()
    return true
  },
  imageImportAnchor: () => ({ ...caret.value }),
  selectedImage: () => props.images.find((i) => i.id === selectedImageId.value),
  applyImages: (images: ImageAsset[]) => {
    const last = images.at(-1)
    importedTarget = last ? { id: last.id, page: last.page } : undefined
    input.value?.blur()
    editing.importImages(images)
    if (last) emit('update:index', last.page)
  },
  applyImage: (image: ImageAsset, replace: boolean) => {
    importedTarget = { id: image.id, page: image.page }
    input.value?.blur()
    if (replace) editing.changeImage(image)
    else editing.addImage(image)
    emit('update:index', image.page)
  },
  importAnchor: () => ({
    page: props.index + 1,
    y: caret.value.page === props.index ? caret.value.y : 0,
    afterId: selectedTableId.value || undefined,
  }),
  addImportedTables: (objects: TableAsset[]) => {
    const last = objects.at(-1)
    importedTarget = last ? { id: last.id, page: last.page } : undefined
    input.value?.blur()
    editing.addTables(objects)
    emit('update:index', objects.at(-1)?.page ?? props.index)
  },
  insertTable: (options: TableInsertOptions) => tableClipboard.insert(options),
  undo: editing.undo,
  redo: editing.redo,
  deselect: () => {
    marquee.clear()
    editing.deselect()
  },
  focus: () => {
    marquee.clear()
    editor.focus()
  },
})
</script>
<template>
  <div
    ref="viewport"
    class="document-viewport"
    tabindex="0"
    @scroll.passive="scroll"
    @paste="paste"
  >
    <FormulaEditDialog
      v-if="document"
      :source="formulaEdit?.source ?? null"
      :settings="document.settings"
      :disabled="disabled"
      @close="formulaEdit = null"
      @save="saveFormula"
    />
    <TableSettings
      v-if="tableSettingsOpen && selectedTable && document"
      :table="selectedTable"
      :preview="selectedTablePreview"
      :selection="tableSelection"
      :settings="document.settings"
      :pages="document.pages.length"
      :disabled="disabled"
      @change="changeTable"
      @remove="editing.removeTable(selectedTable.id)"
      @close="tableSettingsOpen = false"
    />
    <div
      ref="stack"
      class="document-stack"
      @dblclick="editFormula"
      :style="{ width: `${width + 112}px` }"
      @pointerdown="pointerDown"
      @pointerdown.capture="capturePointerDown"
      @pointermove="stack && editor.pointerMove($event, stack)"
      @pointerup="editor.pointerUp"
      @pointercancel="editor.pointerUp"
      @lostpointercapture="lostCapture"
    >
      <template v-if="document"
        ><PaperPage
          v-for="page in document.pages"
          :key="page.number"
          :page="page"
          :settings="document.settings"
          :scale="scale"
          :guides="guides"
          :ruled="ruled"
          :selection-start="start"
          :selection-end="end"
          :marquee="selectionDrag.selection.value"
          :tables="placedTables"
          :selected-table-id="selectedTableId"
          :new-text-box-id="newTextBoxId"
          :images="placedImages"
          :image-previews="imagePreviews"
          :selected-image-id="selectedImageId"
          :selected-text-id="selectedTextId"
          :tool="tool"
          :erasures="erasures"
          @add-erasure="editing.addErasure"
          @erase-start="
            (page) => {
              flushEditors()
              editing.deselect()
              input?.blur()
              emit('update:index', page)
            }
          "
          :erase-mode="eraseMode"
          :eraser-radius="eraserRadius"
          :disabled="disabled"
          @select-table="selectTable"
          @table-selection="tableSelection = $event"
          @change-table="changeTable"
          @remove-table="editing.removeTable"
          @text-box-edit-end="newTextBoxId = ''"
          @table-settings="tableSettingsOpen = true"
          @notice="emit('notice', $event)"
          @undo="editing.undo"
          @redo="editing.redo"
          @select-image="selectImage"
          @select-text="editing.selectRegion"
          @remove-text="editing.removeRegion"
          @change-image="changeImage"
          @remove-image="editing.removeImage"
          @edit-image-text="editing.editRegionText"
          @add-image-text="editing.addRegion"
      /></template>
      <div
        v-else
        class="paper-page loading-paper"
        :style="{ width: `${width}px`, height: `${pageHeight}px` }"
      >
        正在加载字库…
      </div>
      <div ref="caretElement" class="caret-anchor" :style="caretStyle">
        <div
          v-if="tool !== 'erase' && focused && start === end && !composition"
          class="paper-caret"
        ></div>
        <span
          v-if="tool !== 'erase' && composition"
          class="composition-text"
          :style="{ fontSize: `${(document?.settings.fontSize ?? 6) * scale * 0.8}px` }"
          >{{ composition }}</span
        >
        <textarea
          ref="input"
          class="paper-input"
          aria-label="书写内容"
          :disabled="disabled"
          spellcheck="false"
          autocapitalize="off"
          autocomplete="off"
          autocorrect="off"
          wrap="off"
          @input="editor.changed"
          @select="editor.readSelection"
          @keydown="editor.keydown"
          @beforeinput="editing.beforeInput"
          @keyup="editor.readSelection"
          @focus="focused = true"
          @blur="focused = false"
          @compositionstart="editor.compositionStart"
          @compositionupdate="editor.compositionUpdate"
          @compositionend="editor.compositionEnd"
        ></textarea>
      </div>
    </div>
  </div>
</template>
