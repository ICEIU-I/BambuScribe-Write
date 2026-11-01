<script setup lang="ts">
import { useInkCuts } from '../../state/useInkCuts'
import { usePageMask } from '../../state/pageEraser'
import { computed, ref } from 'vue'
import type { LayoutSettings } from '../../types'
import type { TableAsset, TableCell, TablePlacement, TableSelection } from '../../state/tableTypes'
import { cellBounds, offsets, sum } from '../../state/tableGeometry'
import { nativeObjectPath, nativeObjectStrokes } from '../../state/nativeObjectInk'
import { useTableCells } from '../../state/useTableCells'
import { useTableDrag } from '../../state/useTableDrag'
const props = defineProps<{
  asset: TableAsset
  preview?: TablePlacement
  scale: number
  settings: LayoutSettings
  selected: boolean
  interactive: boolean
  disabled: boolean
}>()
const emit = defineEmits<{
  select: [string]
  selection: [TableSelection | null]
  change: [TableAsset]
  remove: [string]
  settings: []
  notice: [string]
  undo: []
  redo: []
}>()
const clipInk = useInkCuts(() => props.asset.id)
const host = ref<HTMLElement>()
const resolved = computed<TableAsset>(() => ({
  ...props.asset,
  columnWidths:
    props.preview?.columnWidths.length === props.asset.columnWidths.length
      ? props.preview.columnWidths
      : props.asset.columnWidths,
  padding: props.preview?.padding ?? props.asset.padding,
  rowHeights:
    props.preview?.rowHeights.length === props.asset.rowHeights.length
      ? props.preview.rowHeights
      : props.asset.rowHeights,
}))
const drag = useTableDrag(
  host,
  () => props.asset,
  () => resolved.value,
  () => props.settings,
  () => props.scale,
  (t) => emit('change', t),
  () => props.preview,
)
const shown = computed(() => drag.draft.value ?? resolved.value),
  width = computed(() => sum(shown.value.columnWidths)),
  height = computed(() => sum(shown.value.rowHeights)),
  xs = computed(() => offsets(shown.value.columnWidths)),
  ys = computed(() => offsets(shown.value.rowHeights))
const pageMask = usePageMask(() => ({ ...shown.value, width: width.value, height: height.value }))
const strokePath = computed(() =>
  nativeObjectPath(nativeObjectStrokes(shown.value, props.preview, clipInk)),
)
const style = computed(() => ({
  left: `${shown.value.x * props.scale + drag.offset.value.x}px`,
  top: `${shown.value.y * props.scale + drag.offset.value.y}px`,
  width: `${width.value * props.scale}px`,
  height: `${height.value * props.scale}px`,
  pointerEvents: props.interactive ? ('auto' as const) : ('none' as const),
}))
const cellStyle = (c: TableCell) => {
  const b = cellBounds(shown.value, c)
  return {
    left: `${b.x * props.scale}px`,
    top: `${b.y * props.scale}px`,
    width: `${b.width * props.scale}px`,
    height: `${b.height * props.scale}px`,
  }
}
const cells = useTableCells(
  props,
  () => shown.value,
  host,
  () => props.scale,
  {
    select: (id) => emit('select', id),
    selection: (value) => emit('selection', value),
    change: (value) => emit('change', value),
    remove: (id) => emit('remove', id),
    settings: () => emit('settings'),
    notice: (message) => emit('notice', message),
    undo: () => emit('undo'),
    redo: () => emit('redo'),
  },
)
const {
  selection,
  editing,
  selected,
  setInput,
  openSettings,
  range,
  startCell,
  save,
  edit,
  key,
  copy,
  paste,
  merge,
  split,
} = cells
function begin(
  e: PointerEvent,
  kind: 'move' | 'size' | 'row' | 'col',
  index = 0,
  wholeGroup = false,
) {
  if (props.disabled) return
  e.preventDefault()
  save()
  cells.focus()
  if (kind === 'move') range(null)
  drag.begin(e, kind, index, wholeGroup)
}
</script>
<template>
  <div
    ref="host"
    class="table-overlay"
    :class="{
      'is-selected': props.selected && interactive,
      'table-invalid': preview?.issues.length,
    }"
    :style="style"
    :data-table="asset.id"
    tabindex="-1"
    aria-label="可编辑表格"
    role="grid"
    @pointerdown.stop
    @keydown="key"
    @copy="copy($event)"
    @cut="copy($event, true)"
    @paste="paste"
    @contextmenu.prevent="openSettings"
  >
    <svg class="table-ink" :style="pageMask" :viewBox="`0 0 ${width} ${height}`" aria-hidden="true">
      <path
        :d="strokePath"
        fill="none"
        stroke="#1a1c1f"
        stroke-width="0.26"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </svg>
    <div
      v-for="c in shown.cells"
      :key="`${c.row}:${c.col}`"
      class="table-cell"
      :class="{ 'is-selected': selected.includes(c) && interactive && props.selected }"
      :style="cellStyle(c)"
      :data-table-cell="`${c.row}:${c.col}`"
      role="gridcell"
      :aria-label="`第 ${c.row + 1} 行第 ${c.col + 1} 列：${c.text || '空白'}`"
      :aria-selected="selected.includes(c) && props.selected"
      @pointerdown.stop="startCell($event, c)"
      @dblclick.stop="edit(c)"
    >
      <textarea
        v-if="editing?.row === c.row && editing.col === c.col"
        :ref="setInput"
        v-model="editing.text"
        class="table-cell-input"
        aria-label="单元格文字"
        wrap="off"
        maxlength="20000"
        :style="{
          fontSize: `${(preview?.cells.find((cell) => cell.row === c.row && cell.col === c.col)?.fontSize ?? preview?.fontSize ?? shown.fontSize) * scale * 0.85}px`,
          padding: `${shown.padding * scale}px`,
        }"
        @pointerdown.stop
        @dblclick.stop
        @blur="save"
      />
    </div>
    <template v-if="props.selected && interactive && !disabled">
      <div class="table-object-controls" @pointerdown.stop>
        <button
          class="table-move"
          aria-label="移动表格"
          title="仅移动表格，可跨页"
          @pointerdown="begin($event, 'move')"
        >
          ⠿
        </button>
        <button
          v-if="asset.groupId"
          class="table-group-move"
          aria-label="整组移动"
          title="拖动表格、表名和说明，可跨页"
          @pointerdown="begin($event, 'move', 0, true)"
        >
          整组移动
        </button>
        <button v-if="selection" :disabled="selected.length < 2" @click="merge">合并</button
        ><button
          v-if="selection"
          :disabled="!selected.some((c) => c.rowSpan > 1 || c.colSpan > 1)"
          @click="split"
        >
          拆分
        </button>
        <button aria-label="表格设置" @click="emit('settings')">设置</button
        ><button aria-label="删除表格" @click="emit('remove', asset.id)">×</button>
      </div>
      <button
        v-for="(x, i) in xs.slice(1, -1)"
        :key="`c${i}`"
        class="table-column-handle"
        :style="{ left: `${x * scale}px` }"
        :aria-label="`调整第 ${i + 1} 列宽度`"
        @pointerdown.stop="begin($event, 'col', i + 1)"
      />
      <button
        v-for="(y, i) in ys.slice(1, -1)"
        :key="`r${i}`"
        class="table-row-handle"
        :style="{ top: `${y * scale}px` }"
        :aria-label="`调整第 ${i + 1} 行高度`"
        @pointerdown.stop="begin($event, 'row', i + 1)"
      />
      <button
        class="table-size-handle"
        aria-label="调整表格大小"
        @pointerdown.stop="begin($event, 'size')"
      />
    </template>
  </div>
</template>
