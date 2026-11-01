<script setup lang="ts">
import { useInkCuts } from '../../state/useInkCuts'
import { usePageMask } from '../../state/pageEraser'
import { computed, ref } from 'vue'
import type { LayoutSettings } from '../../types'
import type { TableAsset, TablePlacement } from '../../state/tableTypes'
import { useTableDrag } from '../../state/useTableDrag'
import { useFreeTextBoxEditor } from '../../state/useFreeTextBoxEditor'
import { cloneTable, setCellText } from '../../state/tableOperations'
import { readTable, writeTable } from '../../state/tableClipboard'
import { nativeObjectPath, nativeObjectStrokes } from '../../state/nativeObjectInk'
import { sum } from '../../state/tableGeometry'
import { fitTableFont } from '../../state/tableFontFit'
const props = defineProps<{
  asset: TableAsset
  preview?: TablePlacement
  settings: LayoutSettings
  scale: number
  selected: boolean
  interactive: boolean
  disabled: boolean
  startEditing: boolean
}>()
const emit = defineEmits<{
  select: [string]
  change: [TableAsset]
  remove: [string]
  editEnd: []
  notice: [string]
}>()
const clipInk = useInkCuts(() => props.asset.id)
const host = ref<HTMLElement>()
const measured = computed(() => ({
  ...props.asset,
  rowHeights:
    props.preview?.rowHeights.length === 1 ? props.preview.rowHeights : props.asset.rowHeights,
}))
const drag = useTableDrag(
  host,
  () => props.asset,
  () => measured.value,
  () => props.settings,
  () => props.scale,
  (asset) => emit('change', asset),
  () => props.preview,
)
const shown = computed(() => drag.draft.value ?? measured.value),
  fontSize = computed(() => fitTableFont(shown.value, props.preview).fontSize),
  width = computed(() => sum(shown.value.columnWidths)),
  height = computed(() => sum(shown.value.rowHeights)),
  path = computed(() => nativeObjectPath(nativeObjectStrokes(shown.value, props.preview, clipInk)))
const pageMask = usePageMask(() => ({ ...shown.value, width: width.value, height: height.value }))
function focus() {
  host.value?.focus({ preventScroll: true })
  emit('select', props.asset.id)
}
const editor = useFreeTextBoxEditor(
  props,
  host,
  focus,
  (asset) => emit('change', asset),
  () => emit('editEnd'),
)
const { input, editing, text, edit, save, key } = editor
function begin(e: PointerEvent, kind: 'move' | 'size' = 'move', wholeGroup = false) {
  if (props.disabled || !props.interactive || e.button !== 0 || editing.value) return
  e.preventDefault()
  focus()
  drag.begin(e, kind, 0, wholeGroup)
}
function align(e: Event) {
  const align = (e.target as HTMLSelectElement).value as TableAsset['align']
  emit('change', { ...cloneTable(props.asset), align })
}
function copy(e: ClipboardEvent, cut = false) {
  if (editing.value || props.disabled || !e.clipboardData) return
  e.preventDefault()
  e.stopPropagation()
  writeTable(e.clipboardData, props.asset)
  e.clipboardData.setData('text/plain', props.asset.cells[0]!.text)
  if (cut) emit('remove', props.asset.id)
}
function paste(e: ClipboardEvent) {
  const data = e.clipboardData
  if (editing.value || props.disabled || !data || readTable(data)) return
  if (
    data.types.some((type) => type.includes('bambuscribe') || type.startsWith('image/')) ||
    data.files.length
  )
    return
  e.preventDefault()
  e.stopPropagation()
  emit('change', setCellText(props.asset, 0, 0, data.getData('text/plain')))
}
</script>
<template>
  <div
    ref="host"
    class="manual-text-box free-text-box"
    :class="{
      'is-selected': selected && interactive,
      'is-editing': editing,
      'text-box-invalid': preview?.issues.length,
    }"
    :data-free-text-box="asset.id"
    :data-text-box="asset.id"
    tabindex="-1"
    :aria-label="`文本框：${asset.cells[0]?.text || '空白'}`"
    :style="{
      left: `${shown.x * scale + drag.offset.value.x}px`,
      top: `${shown.y * scale + drag.offset.value.y}px`,
      width: `${width * scale}px`,
      height: `${height * scale}px`,
      pointerEvents: interactive || editing ? 'auto' : 'none',
    }"
    @pointerdown.stop="begin($event)"
    @dblclick.stop="edit"
    @copy="copy($event)"
    @cut="copy($event, true)"
    @paste="paste"
  >
    <svg
      class="free-text-ink"
      :style="pageMask"
      :viewBox="`0 0 ${width} ${height}`"
      aria-hidden="true"
    >
      <path
        :d="path"
        fill="none"
        stroke="#1a1c1f"
        stroke-width="0.26"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </svg>
    <div
      v-if="selected && interactive && !disabled && !editing"
      class="free-text-controls"
      @pointerdown.stop
      @dblclick.stop
    >
      <button
        type="button"
        aria-label="移动文本框"
        title="仅移动文本框，可跨页"
        @pointerdown="begin($event)"
      >
        ⠿
      </button>
      <button
        v-if="asset.groupId"
        class="table-group-move"
        type="button"
        aria-label="整组移动"
        title="拖动表格、表名和说明，可跨页"
        @pointerdown="begin($event, 'move', true)"
      >
        整组移动
      </button>
      <label
        >自动字号（mm）<input
          type="number"
          :value="fontSize.toFixed(2)"
          readonly
          aria-label="文本框字号"
      /></label>
      <select :value="asset.align" aria-label="文本框对齐" @change="align">
        <option value="left">左对齐</option>
        <option value="center">居中</option>
        <option value="right">右对齐</option>
      </select>
      <button type="button" aria-label="删除文本框" @click.stop="emit('remove', asset.id)">
        ×
      </button>
    </div>
    <button
      v-if="selected && interactive && !disabled && !editing"
      class="text-box-resize"
      aria-label="调整文本框大小"
      @pointerdown.stop="begin($event, 'size')"
      @dblclick.stop
    />
    <textarea
      v-if="editing"
      ref="input"
      v-model="text"
      class="free-text-input"
      aria-label="文本框文字"
      wrap="off"
      maxlength="20000"
      :style="{
        fontSize: `${fontSize * scale * 0.85}px`,
        padding: `${shown.padding * scale}px`,
      }"
      @pointerdown.stop
      @dblclick.stop
      @keydown="key"
      @blur="save"
    />
  </div>
</template>
