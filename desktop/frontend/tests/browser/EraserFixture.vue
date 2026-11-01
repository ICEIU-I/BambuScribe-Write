<script setup lang="ts">
import { computed, reactive, ref, shallowRef, onMounted, onBeforeUnmount } from 'vue'
import ImageOverlay from '../../src/components/editor/ImageOverlay.vue'
import { defaultLayout } from '../../src/state/defaults'
import { useDocumentEditing } from '../../src/state/useDocumentEditing'
import type { CalculationRecord } from '../../src/services/calculationTypes'
import type { PageErasure, PageLayout, TableAsset } from '../../src/types'
import type { EraseMode, ImageAsset, ImageEditTool, ImagePlacement, Point } from '../../src/types'
import '../../src/styles/base.css'
import '../../src/styles/paper-editor.css'
const settings = defaultLayout()
const tool = ref<ImageEditTool>('erase')
const eraseMode = ref<EraseMode>('free')
const scale = ref(3),
  radius = ref(0.02)
const state = reactive({
  modelValue: '',
  calculations: [] as CalculationRecord[],
  pageLayouts: undefined as PageLayout[] | undefined,
  erasures: [] as PageErasure[],
  tables: [] as TableAsset[],
  images: [] as ImageAsset[],
  document: null,
  disabled: false,
  active: true,
})
const input = ref<HTMLTextAreaElement>()
const editing = useDocumentEditing(
  state,
  input,
  (text, images, tables, erasures, pageLayouts, calculations) => {
    state.modelValue = text
    state.images = images
    state.tables = tables
    state.erasures = erasures
    state.pageLayouts = pageLayouts
    state.calculations = calculations ?? []
  },
  () => {},
)
const heldPreview = shallowRef<ImagePlacement | null>(null)
const asset = computed(() => state.images[0])
const preview = computed<ImagePlacement>(() => {
  const im = asset.value
  const strokes = [0.2, 0.4, 0.6, 0.8].map((y) => [
    { x: im.x, y: im.y + y * im.height },
    { x: im.x + im.width, y: im.y + y * im.height },
  ])
  strokes.push([
    { x: im.x + im.width * 0.6, y: im.y },
    { x: im.x + im.width * 0.6, y: im.y + im.height },
  ])
  const radians = (im.rotation * Math.PI) / 180,
    cx = im.x + im.width / 2,
    cy = im.y + im.height / 2
  const rotate = (strokes: Point[][]) =>
    strokes.map((s) =>
      s.map((p) => ({
        x: cx + (p.x - cx) * Math.cos(radians) - (p.y - cy) * Math.sin(radians),
        y: cy + (p.x - cx) * Math.sin(radians) + (p.y - cy) * Math.cos(radians),
      })),
    )
  const rotated = rotate(strokes)
  const textRegions = (im.textRegions ?? [])
    .filter((r) => r.mode === 'handwriting')
    .map((r) => {
      const raw = r.text
        ? rotate([
            [
              { x: im.x + r.x * im.width, y: im.y + r.y * im.height },
              { x: im.x + (r.x + r.width) * im.width, y: im.y + (r.y + r.height) * im.height },
            ],
          ])
        : []
      return { ...r, strokes: raw, sourceStrokes: raw }
    })
  return { ...im, strokes: rotated, graphicStrokes: [], sourceStrokes: rotated, textRegions }
})
function setup(width = 160, height = 80, zoom = 3, rotation = 0) {
  state.images = [
    {
      id: 'fixture',
      page: 0,
      data: '',
      x: 10,
      y: 10,
      width,
      height,
      rotation,
      mode: 'line',
      textRegions: [],
    },
  ]
  heldPreview.value = null
  scale.value = zoom
  radius.value = 0.02
}
setup()
onMounted(() => {
  window.addEventListener('copy', editing.copy)
  window.addEventListener('paste', editing.paste)
  window.addEventListener('keydown', editing.globalKeydown, true)
})
onBeforeUnmount(() => {
  window.removeEventListener('copy', editing.copy)
  window.removeEventListener('paste', editing.paste)
  window.removeEventListener('keydown', editing.globalKeydown, true)
})
Object.assign(window, {
  eraserFixture: {
    setup,
    state,
    tool,
    eraseMode,
    radius,
    freeze: () => {
      heldPreview.value = JSON.parse(JSON.stringify(preview.value))
    },
  },
})
</script>
<template>
  <button @click="editing.undo()">撤销</button><button @click="editing.redo()">重做</button>
  <select v-model="tool" aria-label="工具">
    <option value="erase">擦除</option>
    <option value="select">选择</option>
    <option value="text">文本框</option>
  </select>
  <select v-model="eraseMode" aria-label="擦除方式">
    <option value="free">自由</option>
    <option value="box">框选</option>
  </select>
  <div style="height: 100px"></div>
  <div style="position: relative; margin: 80px; height: 1500px; background: white">
    <ImageOverlay
      :asset="asset"
      :preview="heldPreview ?? preview"
      :scale="scale"
      :selected="true"
      :selected-text-id="editing.regionId.value"
      :tool="tool"
      :erase-mode="eraseMode"
      :eraser-radius="radius"
      :settings="settings"
      :disabled="false"
      @select="editing.select"
      @select-text="editing.selectRegion"
      @edit-text="editing.editRegionText"
      @remove-text="editing.removeRegion"
      @add="editing.addRegion"
    />
  </div>
</template>
