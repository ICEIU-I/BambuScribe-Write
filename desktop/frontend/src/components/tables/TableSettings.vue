<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import type { LayoutSettings } from '../../types'
import type {
  TableAsset,
  TablePlacement,
  TableSelection,
  TableAlign,
  TableVerticalAlign,
  TableStyle,
} from '../../state/tableTypes'
import {
  cloneTable,
  changeAxis,
  mergeCells,
  resizeTable,
  setBorders,
  setCellOptions,
  splitCells,
} from '../../state/tableOperations'
import { selectionBounds, sum } from '../../state/tableGeometry'
import { fitTableCellFont, fitTableFont } from '../../state/tableFontFit'
const props = defineProps<{
  table: TableAsset
  preview?: TablePlacement
  selection: TableSelection | null
  settings: LayoutSettings
  pages: number
  disabled: boolean
}>()
const emit = defineEmits<{ change: [TableAsset]; remove: []; close: [] }>()
const error = ref(''),
  panel = ref<HTMLElement>()
const measured = computed(() => ({
  ...props.table,
  columnWidths: props.preview?.columnWidths ?? props.table.columnWidths,
  padding: props.preview?.padding ?? props.table.padding,
  fontSize: fitTableFont(props.table, props.preview).fontSize,
  rowHeights:
    props.preview?.rowHeights.length === props.table.rowHeights.length
      ? props.preview.rowHeights
      : props.table.rowHeights,
}))
const fontRange = computed(() => {
  const sizes = props.table.cells
    .filter((cell) => cell.text)
    .map((cell) => {
      const old = props.preview?.cells.find((c) => c.row === cell.row && c.col === cell.col)
      return fitTableCellFont(
        measured.value,
        cell,
        old,
        props.preview?.fontSize ?? props.table.fontSize,
        props.preview?.maxFontSize,
      )
    })
  if (!sizes.length) return measured.value.fontSize.toFixed(2)
  const min = Math.min(...sizes).toFixed(2),
    max = Math.max(...sizes).toFixed(2)
  return min === max ? min : `${min}–${max}`
})
const bounds = computed(() =>
  props.selection
    ? selectionBounds(props.table, props.selection)
    : {
        r0: 0,
        c0: 0,
        r1: props.table.rowHeights.length - 1,
        c1: props.table.columnWidths.length - 1,
      },
)
function action(work: () => TableAsset) {
  try {
    const t = work()
    error.value = ''
    emit('change', t)
  } catch (e) {
    error.value = (e as Error).message
  }
}
function value(e: Event) {
  return Number((e.target as HTMLInputElement).value)
}
function property(key: 'padding' | 'x' | 'y' | 'page', e: Event) {
  const v = value(e),
    max =
      key === 'padding'
        ? 10
        : key === 'x'
          ? props.settings.paperWidth
          : key === 'y'
            ? props.settings.paperHeight
            : 20001,
    min = key === 'page' ? 1 : 0
  if (!Number.isFinite(v) || v < min || v > max) {
    error.value = '数值超出范围，请检查输入。'
    return
  }
  action(() => ({ ...cloneTable(props.table), [key]: key === 'page' ? Math.floor(v) - 1 : v }))
}
function dimension(axis: 'width' | 'height', e: Event) {
  const n = value(e)
  if (
    !Number.isFinite(n) ||
    n <= 0 ||
    n > (axis === 'width' ? props.settings.paperWidth : props.settings.paperHeight)
  ) {
    error.value = '尺寸须为正数且不超过纸张。'
    return
  }
  action(() => {
    const resized = fitTableFont(
      resizeTable(
        measured.value,
        axis === 'width' ? n : sum(measured.value.columnWidths),
        axis === 'height' ? n : sum(measured.value.rowHeights),
      ),
      props.preview,
    )
    return resized
  })
}
function sizes(axis: 'row' | 'col', e: Event) {
  const n = value(e)
  if (!Number.isFinite(n) || n < 3 || n > 1000) {
    error.value = '行高和列宽须为 3–1000 mm。'
    return
  }
  action(() => {
    const t = cloneTable(measured.value),
      array = axis === 'row' ? t.rowHeights : t.columnWidths,
      b = bounds.value
    for (let i = axis === 'row' ? b.r0 : b.c0; i <= (axis === 'row' ? b.r1 : b.c1); i++)
      array[i] = n
    const fitted = fitTableFont(t, props.preview)
    return fitted
  })
}
function average(axis: 'row' | 'col') {
  action(() => {
    const t = cloneTable(measured.value),
      array = axis === 'row' ? t.rowHeights : t.columnWidths,
      total = sum(array)
    array.fill(total / array.length)
    const fitted = fitTableFont(t, props.preview)
    return fitted
  })
}
function style(e: Event) {
  action(() => {
    const t = cloneTable(props.table)
    t.style = (e.target as HTMLSelectElement).value as TableStyle
    for (const c of t.cells) c.borders = undefined
    return t
  })
}
function align(e: Event, vertical = false) {
  action(() =>
    setCellOptions(
      props.table,
      bounds.value,
      vertical
        ? { verticalAlign: (e.target as HTMLSelectElement).value as TableVerticalAlign }
        : { align: (e.target as HTMLSelectElement).value as TableAlign },
    ),
  )
}
function shortcut(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    e.preventDefault()
    emit('close')
  }
}
onMounted(() => {
  panel.value?.focus()
  window.addEventListener('keydown', shortcut)
})
onBeforeUnmount(() => window.removeEventListener('keydown', shortcut))
</script>
<template>
  <Teleport to="body"
    ><section
      ref="panel"
      class="table-settings-panel"
      role="dialog"
      aria-label="表格设置"
      tabindex="-1"
      @pointerdown.stop
    >
      <div class="table-panel-heading">
        <strong>表格设置</strong
        ><button class="icon-button" aria-label="关闭表格设置" @click="emit('close')">×</button>
      </div>
      <div class="table-form" :class="{ 'is-disabled': disabled }">
        <label
          >样式<select :value="table.style" @change="style">
            <option value="grid">普通网格表</option>
            <option value="three-line">三线表</option>
            <option value="horizontal">仅横线表</option>
            <option value="none">无边框</option>
          </select></label
        >
        <div class="table-form-row">
          <label
            >宽度（mm）<input
              type="number"
              :value="sum(table.columnWidths).toFixed(1)"
              min="3"
              :max="settings.paperWidth"
              step="0.5"
              aria-label="设置表格宽度"
              @change="dimension('width', $event)" /></label
          ><label
            >高度（mm）<input
              type="number"
              :value="sum(measured.rowHeights).toFixed(1)"
              min="3"
              :max="settings.paperHeight"
              step="0.5"
              aria-label="设置表格高度"
              @change="dimension('height', $event)"
          /></label>
        </div>
        <div class="table-form-row">
          <label
            >自动字号（mm）<input
              type="text"
              :value="fontRange"
              readonly
              aria-label="表格字号" /></label
          ><label
            >内边距（mm）<input
              type="number"
              :value="table.padding"
              min="0"
              max="10"
              step="0.5"
              aria-label="表格内边距"
              @change="property('padding', $event)"
          /></label>
        </div>
        <div class="table-form-row">
          <label
            >X（mm）<input
              type="number"
              :value="table.x.toFixed(1)"
              min="0"
              :max="settings.paperWidth"
              step="0.5"
              aria-label="表格位置 X"
              @change="property('x', $event)" /></label
          ><label
            >Y（mm）<input
              type="number"
              :value="table.y.toFixed(1)"
              min="0"
              :max="settings.paperHeight"
              step="0.5"
              aria-label="表格位置 Y"
              @change="property('y', $event)" /></label
          ><label
            >页码<input
              type="number"
              :value="table.page + 1"
              min="1"
              :max="Math.max(pages, table.page + 1) + 1"
              step="1"
              aria-label="表格页码"
              @change="property('page', $event)"
          /></label>
        </div>
        <div class="table-panel-section">
          <strong>{{
            selection
              ? `选中：第 ${bounds.r0 + 1}–${bounds.r1 + 1} 行，第 ${bounds.c0 + 1}–${bounds.c1 + 1} 列`
              : '整张表格'
          }}</strong>
          <div class="table-form-row">
            <label
              >列宽（mm）<input
                type="number"
                :value="table.columnWidths[bounds.c0]?.toFixed(1)"
                min="3"
                max="1000"
                step="0.5"
                aria-label="选中列宽"
                @change="sizes('col', $event)" /></label
            ><label
              >行高（mm）<input
                type="number"
                :value="table.rowHeights[bounds.r0]?.toFixed(1)"
                min="3"
                max="1000"
                step="0.5"
                aria-label="选中行高"
                @change="sizes('row', $event)"
            /></label>
          </div>
          <div class="table-form-row">
            <button class="button" @click="average('col')">平均列宽</button
            ><button class="button" @click="average('row')">平均行高</button>
          </div>
          <div class="table-form-row">
            <label
              >水平对齐<select
                aria-label="单元格水平对齐"
                :value="
                  table.cells.find((c) => c.row === bounds.r0 && c.col === bounds.c0)?.align ??
                  table.align
                "
                @change="align($event)"
              >
                <option value="left">左对齐</option>
                <option value="center">居中</option>
                <option value="right">右对齐</option>
              </select></label
            ><label
              >垂直对齐<select
                aria-label="单元格垂直对齐"
                :value="
                  table.cells.find((c) => c.row === bounds.r0 && c.col === bounds.c0)
                    ?.verticalAlign ?? table.verticalAlign
                "
                @change="align($event, true)"
              >
                <option value="top">顶部</option>
                <option value="middle">居中</option>
                <option value="bottom">底部</option>
              </select></label
            >
          </div>
          <div class="table-form-row">
            <button
              class="button"
              :disabled="!selection"
              @click="action(() => mergeCells(table, bounds))"
            >
              合并单元格</button
            ><button
              class="button"
              :disabled="!selection"
              @click="action(() => splitCells(table, bounds))"
            >
              拆分单元格
            </button>
          </div>
          <label
            >边框<select
              aria-label="单元格边框"
              value=""
              @change="
                action(() =>
                  setBorders(
                    table,
                    bounds,
                    ($event.target as HTMLSelectElement).value as
                      'all' | 'outer' | 'none' | 'reset',
                  ),
                )
              "
            >
              <option value="" disabled>选择边框</option>
              <option value="all">全部边框</option>
              <option value="outer">外边框</option>
              <option value="none">无边框</option>
              <option value="reset">跟随表格样式</option>
            </select></label
          >
        </div>
        <div class="table-panel-section">
          <strong>行列</strong>
          <div class="table-form-row">
            <button
              class="button"
              @click="
                action(() =>
                  changeAxis(table, 'row', selection ? bounds.r1 + 1 : table.rowHeights.length),
                )
              "
            >
              下方插入行</button
            ><button
              class="button"
              @click="
                action(() =>
                  changeAxis(table, 'col', selection ? bounds.c1 + 1 : table.columnWidths.length),
                )
              "
            >
              右侧插入列
            </button>
          </div>
          <div class="table-form-row">
            <button
              class="button"
              :disabled="table.rowHeights.length === 1"
              @click="action(() => changeAxis(table, 'row', bounds.r0, true))"
            >
              删除当前行</button
            ><button
              class="button"
              :disabled="table.columnWidths.length === 1"
              @click="action(() => changeAxis(table, 'col', bounds.c0, true))"
            >
              删除当前列
            </button>
          </div>
        </div>
        <p v-if="error" class="table-error" role="alert">{{ error }}</p>
        <button class="button" @click="emit('remove')">删除表格</button>
      </div>
    </section></Teleport
  >
</template>
