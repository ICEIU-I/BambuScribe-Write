<script setup lang="ts">
import { ref } from 'vue'
import type { EraseMode, ImageEditTool, LayoutSettings } from '../../types'
import AppIcon from '../AppIcon.vue'
import ZoomControl from './ZoomControl.vue'
import PixelPopover from '../ui/PixelPopover.vue'
import TableInsert from '../tables/TableInsert.vue'
import SmartImportButton from '../import/SmartImportButton.vue'
import type { TableInsertOptions } from '../../state/tableTypes'
const props = defineProps<{
  settings: LayoutSettings
  settingsOpen: boolean
  zoom: number
  ruled: boolean
  tool: ImageEditTool
  eraseMode: EraseMode
  eraserRadius: number
  disabled: boolean
}>()
const emit = defineEmits<{
  importTable: []
  importDiagram: []
  insertTable: [TableInsertOptions]
  settings: []
  undo: []
  redo: []
  'update:zoom': [number]
  'update:ruled': [boolean]
  'update:tool': [ImageEditTool]
  'update:eraseMode': [EraseMode]
  'update:eraserRadius': [number]
}>()
const tools: {
  value: ImageEditTool
  label: string
  icon: string
  hint: string
  key: string
}[] = [
  {
    value: 'select',
    label: '选择',
    icon: 'frame',
    hint: '选择、移动和缩放图片或文本框',
    key: '⌘1',
  },
  {
    value: 'text',
    label: '文本框',
    icon: 'text',
    hint: '单击纸面任意位置，添加手写文本框',
    key: '⌘2',
  },
  {
    value: 'erase',
    label: '擦除',
    icon: 'eraser',
    hint: '移除纸面任意位置的笔画',
    key: '⌘3',
  },
]
const eraseModes: { value: EraseMode; label: string; title: string }[] = [
  { value: 'box', label: '框选', title: '拖动选择矩形，移除矩形内部的笔画' },
  { value: 'free', label: '自由', title: '沿笔画拖动，只擦除笔刷经过的位置' },
]
function insertTable(options: TableInsertOptions) {
  emit('insertTable', options)
  tableOpen.value = false
}
const tableOpen = ref(false),
  tableButton = ref<HTMLButtonElement>()
</script>
<template>
  <section class="editor-toolbar" aria-label="书写工具栏">
    <div class="toolbar-group history-controls" aria-label="历史">
      <button
        class="icon-button"
        title="撤销 ⌘Z"
        aria-label="撤销"
        :disabled="disabled"
        @mousedown.prevent
        @click="emit('undo')"
      >
        <AppIcon name="undo" :size="17" />
      </button>
      <button
        class="icon-button"
        title="重做 ⇧⌘Z"
        aria-label="重做"
        :disabled="disabled"
        @mousedown.prevent
        @click="emit('redo')"
      >
        <AppIcon name="redo" :size="17" />
      </button>
    </div>
    <div class="toolbar-group image-tools" role="group" aria-label="书写工具">
      <button
        v-for="item in tools"
        :key="item.value"
        class="button tool-button"
        :class="{ active: tool === item.value }"
        :disabled="disabled"
        type="button"
        :title="`${item.hint} · ${item.key}`"
        :aria-pressed="tool === item.value"
        @mousedown.prevent
        @click="emit('update:tool', item.value)"
      >
        <AppIcon :name="item.icon" :size="16" /><span>{{ item.label }}</span>
      </button>
    </div>
    <button
      ref="tableButton"
      class="button tool-button table-insert-button"
      :disabled="disabled"
      :aria-expanded="tableOpen"
      aria-haspopup="dialog"
      @click="tableOpen = true"
    >
      <AppIcon name="table" :size="15" />表格
    </button>
    <PixelPopover
      :open="tableOpen"
      :anchor="tableButton"
      :width="280"
      label="插入表格"
      @close="tableOpen = false"
      ><TableInsert :settings="settings" @insert="insertTable"
    /></PixelPopover>
    <SmartImportButton
      :disabled="disabled"
      @table="emit('importTable')"
      @diagram="emit('importDiagram')"
    />
    <div class="tool-options">
      <div v-if="tool === 'erase'" class="erase-modes" role="group" aria-label="擦除方式">
        <button
          v-for="mode in eraseModes"
          :key="mode.value"
          class="erase-mode-button"
          :class="{ active: eraseMode === mode.value }"
          type="button"
          :title="mode.title"
          :aria-pressed="eraseMode === mode.value"
          @mousedown.prevent
          @click="emit('update:eraseMode', mode.value)"
        >
          {{ mode.label }}
        </button>
      </div>
      <label v-if="tool === 'erase' && eraseMode === 'free'" class="tool-select">
        粗细
        <select
          :value="eraserRadius"
          aria-label="自由擦除笔刷粗细"
          :title="`笔刷直径：${Number((eraserRadius * 200).toFixed(1))} mm`"
          @change="emit('update:eraserRadius', Number(($event.target as HTMLSelectElement).value))"
        >
          <option :value="0.008" title="直径 1.6 mm">细</option>
          <option :value="0.018" title="直径 3.6 mm">中</option>
          <option :value="0.032" title="直径 6.4 mm">粗</option>
          <option :value="0.05" title="直径 10 mm">特粗</option>
        </select>
      </label>
    </div>
    <div class="toolbar-group canvas-controls" aria-label="画布视图">
      <label class="line-toggle"
        ><input
          type="checkbox"
          :checked="ruled"
          @change="emit('update:ruled', ($event.target as HTMLInputElement).checked)"
        />横线</label
      >
      <ZoomControl :model-value="zoom" @update:model-value="emit('update:zoom', $event)" />
    </div>
    <button
      id="layout-settings-toggle"
      class="button tool-button layout-toggle"
      :class="{ active: settingsOpen }"
      :disabled="disabled"
      :aria-expanded="settingsOpen"
      aria-controls="layout-inspector"
      @click="emit('settings')"
    >
      <AppIcon name="settings" :size="16" />排版设置
    </button>
  </section>
</template>
