<script setup lang="ts">
import { useInkCuts } from '../../state/useInkCuts'
import { usePageMask } from '../../state/pageEraser'
import { computed, onBeforeUnmount, ref } from 'vue'
import type {
  EraseMode,
  ImageAsset,
  ImagePlacement,
  ImageTextRegion,
  ImageEditTool,
  LayoutSettings,
  Point,
} from '../../types'
import ManualTextBox from './ManualTextBox.vue'
import EraserCursor from './EraserCursor.vue'
import { imageMasks, isTextBox } from '../../state/textBox'
import ImageEraserMask from './ImageEraserMask.vue'
import { freeEraseRegion } from '../../state/imageEraser'
import { displayRegions } from '../../state/imageEditing'
import { useImageDrag } from '../../state/useImageDrag'
const props = defineProps<{
  asset: ImageAsset
  preview?: ImagePlacement
  scale: number
  selected: boolean
  selectedTextId?: string
  tool: ImageEditTool
  eraseMode: EraseMode
  eraserRadius: number
  settings: LayoutSettings
  disabled: boolean
}>()
const emit = defineEmits<{
  select: [string, Point?]
  selectText: [string, string]
  removeText: [string, string]
  change: [ImageAsset]
  editText: [string, ImageTextRegion]
  add: [string, ImageTextRegion]
  remove: [string]
}>()
const clipInk = useInkCuts(() => props.asset.id)
const host = ref<HTMLElement>()
const ink = ref<SVGSVGElement>()
const editingRegionId = ref('')
const placement = useImageDrag(
  host,
  () => props.asset,
  () => props.settings,
  () => props.scale,
  (image) => emit('change', image),
)
const regionDraft = ref<ImageTextRegion | null>(null)
const frame = computed(() => placement.draft.value ?? props.asset)
const pageMask = usePageMask(() => frame.value)
const textDraft = ref<ImageTextRegion | null>(null)
const regions = computed(() =>
  displayRegions(props.asset, props.preview).map((r) =>
    textDraft.value?.id === r.id ? textDraft.value : r,
  ),
)
const eraseRegions = computed(() => {
  const result = imageMasks(regions.value)
  if (regionDraft.value?.mode === 'erase') result.push(regionDraft.value)
  return result
})
const maskId = computed(() => `image-erase-${props.asset.id.replace(/[^a-zA-Z0-9_-]/g, '-')}`)
const clamp = (v: number, min: number, max: number) => Math.max(min, Math.min(max, v))
const isFreeEraser = computed(() => props.tool === 'erase' && props.eraseMode === 'free')
const brushPoint = ref<Point | null>(null)
const brushDiameterMm = computed(
  () => props.eraserRadius * 2 * Math.min(frame.value.width, frame.value.height),
)
function newRegionId() {
  return typeof crypto !== 'undefined' && crypto.randomUUID
    ? crypto.randomUUID()
    : `manual-text-${Date.now()}-${Math.random().toString(16).slice(2)}`
}
function localPoint(p: Point): Point {
  const im = props.preview ?? props.asset,
    angle = -((im.rotation * Math.PI) / 180)
  const dx = p.x - im.x - im.width / 2,
    dy = p.y - im.y - im.height / 2
  return {
    x: (dx * Math.cos(angle) - dy * Math.sin(angle) + im.width / 2) / im.width,
    y: (dx * Math.sin(angle) + dy * Math.cos(angle) + im.height / 2) / im.height,
  }
}
function path(strokes: Point[][]) {
  return strokes
    .map((s) =>
      s
        .map((p, i) => {
          const q = localPoint(p)
          return `${i ? 'L' : 'M'}${q.x} ${q.y}`
        })
        .join(' '),
    )
    .join(' ')
}
const graphicPath = computed(() =>
  path(
    clipInk(
      props.preview?.sourceStrokes ?? props.preview?.graphicStrokes ?? props.preview?.strokes ?? [],
      'graphic',
    ),
  ),
)
const textPaths = computed(() =>
  (props.preview?.textRegions ?? []).map((r) => {
    const index = regions.value.findIndex((region) => region.id === r.id)
    const erasures = imageMasks(regions.value.slice(index + 1))
    if (regionDraft.value?.mode === 'erase') erasures.push(regionDraft.value)
    const frame = regions.value[index] ?? r
    const k = Math.min(frame.width / r.width, frame.height / r.height)
    const transform = `translate(${frame.x - r.x * k} ${frame.y + (frame.height - r.height * k) / 2 - r.y * k}) scale(${k})`
    return {
      id: r.id,
      d: path(clipInk(r.sourceStrokes ?? r.strokes ?? [], `region:${r.id}`)),
      erasures,
      transform,
    }
  }),
)
let gesture:
  | {
      x: number
      y: number
      start?: Point
      target: HTMLElement
      pointer: number
      moved: boolean
    }
  | undefined
function pointerPoint(e: { clientX: number; clientY: number }): Point {
  // Use the exact same transform as the ink, including scroll, zoom and rotation.
  const matrix = ink.value?.getScreenCTM()
  if (!matrix) return { x: 0, y: 0 }
  const point = new DOMPoint(e.clientX, e.clientY).matrixTransform(matrix.inverse())
  return { x: clamp(point.x, 0, 1), y: clamp(point.y, 0, 1) }
}
function trackPointer(event: PointerEvent) {
  if (isFreeEraser.value) brushPoint.value = pointerPoint(event)
}
function begin(e: PointerEvent, kind: 'move' | 'size' | 'draw') {
  if (props.disabled || e.button !== 0) return
  e.stopPropagation()
  if (isFreeEraser.value) brushPoint.value = pointerPoint(e)
  // Keep click/dblclick native events intact; start capture only after a drag.
  host.value?.focus({ preventScroll: true })
  emit('select', props.asset.id, pointerPoint(e))
  if (kind !== 'draw') {
    placement.begin(e, kind)
    return
  }
  gesture = {
    x: e.clientX,
    y: e.clientY,
    target: e.currentTarget as HTMLElement,
    pointer: e.pointerId,
    moved: false,
    start: pointerPoint(e),
  }
  if (isFreeEraser.value) {
    const p = gesture.start!
    regionDraft.value = freeEraseRegion([p, p], props.eraserRadius)
  }
  window.addEventListener('pointermove', move)
  window.addEventListener('pointerup', finish)
  window.addEventListener('pointercancel', cancel)
}
function move(e: PointerEvent) {
  const g = gesture
  if (!g || g.pointer !== e.pointerId) return
  if (isFreeEraser.value) brushPoint.value = pointerPoint(e)
  const dx = e.clientX - g.x,
    dy = e.clientY - g.y
  if (!g.moved && Math.hypot(dx, dy) < 3) return
  if (!g.moved) {
    g.moved = true
    g.target.setPointerCapture(e.pointerId)
  }
  e.preventDefault()
  const p = pointerPoint(e),
    a = g.start!
  if (isFreeEraser.value) {
    const points = [...(regionDraft.value?.erasePath ?? [a]), p]
    regionDraft.value = freeEraseRegion(points, props.eraserRadius)
  } else {
    regionDraft.value = {
      id: 'draft',
      text: '',
      source: 'manual',
      mode: props.tool === 'erase' ? 'erase' : 'handwriting',
      x: Math.min(a.x, p.x),
      y: Math.min(a.y, p.y),
      width: Math.abs(a.x - p.x),
      height: Math.abs(a.y - p.y),
    }
  }
}
function cleanup() {
  const g = gesture
  gesture = undefined
  if (g?.target.hasPointerCapture(g.pointer)) g.target.releasePointerCapture(g.pointer)
  window.removeEventListener('pointermove', move)
  window.removeEventListener('pointerup', finish)
  window.removeEventListener('pointercancel', cancel)
  regionDraft.value = null
}
function addTextRegion(region: ImageTextRegion) {
  emit('add', props.asset.id, region)
  editingRegionId.value = region.id
}

function finish(e: PointerEvent) {
  const g = gesture
  if (!g || g.pointer !== e.pointerId) return
  if (g.moved) {
    if (regionDraft.value) {
      const valid = isFreeEraser.value
        ? (regionDraft.value.erasePath?.length ?? 0) > 1
        : regionDraft.value.width > 0.005 && regionDraft.value.height > 0.005
      if (valid) {
        const region = { ...regionDraft.value, id: newRegionId() }
        if (region.mode === 'handwriting') addTextRegion(region)
        else emit('add', props.asset.id, region)
      }
    }
  } else if (isFreeEraser.value && regionDraft.value) {
    emit('add', props.asset.id, { ...regionDraft.value, id: newRegionId() })
  } else if (props.tool === 'text' && g.start) {
    addTextRegion({
      id: newRegionId(),
      text: '',
      x: clamp(g.start.x, 0, 0.82),
      y: clamp(g.start.y, 0, 0.9),
      width: 0.18,
      height: 0.1,
      mode: 'handwriting',
      source: 'manual',
    })
  }
  cleanup()
}
function cancel() {
  cleanup()
}
function add(e: MouseEvent) {
  if (props.disabled || props.tool !== 'select' || e.target !== e.currentTarget) return
  const p = pointerPoint(e)
  addTextRegion({
    id: newRegionId(),
    text: '',
    x: Math.min(0.82, p.x),
    y: Math.min(0.9, p.y),
    width: 0.18,
    height: 0.1,
    mode: 'handwriting',
    source: 'manual',
  })
}
onBeforeUnmount(cleanup)
</script>
<template>
  <div
    ref="host"
    class="image-selection"
    tabindex="-1"
    :class="{
      selected,
      'eraser-tool': tool === 'erase',
      'free-eraser-tool': isFreeEraser,
      'text-tool': tool === 'text',
    }"
    :data-image-id="asset.id"
    :style="{
      left: `${frame.x * scale + placement.offset.value.x}px`,
      top: `${frame.y * scale + placement.offset.value.y}px`,
      width: `${frame.width * scale}px`,
      height: `${frame.height * scale}px`,
      transform: `rotate(${frame.rotation}deg)`,
    }"
    @pointerdown="begin($event, tool === 'select' ? 'move' : 'draw')"
    @pointermove="trackPointer"
    @pointerleave="brushPoint = null"
    @dblclick.stop="add"
  >
    <svg
      ref="ink"
      class="image-ink"
      :style="pageMask"
      viewBox="0 0 1 1"
      preserveAspectRatio="none"
      aria-hidden="true"
    >
      <defs>
        <ImageEraserMask
          :id="maskId"
          :regions="eraseRegions"
          :width="frame.width"
          :height="frame.height"
        />
        <ImageEraserMask
          v-for="p in textPaths"
          :key="p.id"
          :id="`${maskId}-${p.id}`"
          :regions="p.erasures"
          :width="frame.width"
          :height="frame.height"
        />
      </defs>
      <g
        fill="none"
        stroke="#1a1c1f"
        :stroke-width="0.26 * scale"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <path :d="graphicPath" :mask="`url(#${maskId})`" vector-effect="non-scaling-stroke" />
        <template v-for="p in textPaths" :key="p.id">
          <g v-if="regions.some((r) => r.id === p.id)" :mask="`url(#${maskId}-${p.id})`">
            <path :d="p.d" :transform="p.transform" vector-effect="non-scaling-stroke" />
          </g>
        </template>
      </g>
    </svg>
    <EraserCursor
      v-if="isFreeEraser && brushPoint"
      :point="brushPoint"
      :width="frame.width * scale"
      :height="frame.height * scale"
      :diameter="brushDiameterMm * scale"
    />
    <ManualTextBox
      v-for="r in regions.filter(isTextBox)"
      :key="r.id"
      :region="r"
      :selected="selected && selectedTextId === r.id"
      :interactive="tool === 'select' || tool === 'text'"
      :disabled="disabled"
      :start-editing="editingRegionId === r.id"
      :locate="pointerPoint"
      @select="emit('selectText', asset.id, r.id)"
      @draft="textDraft = $event"
      @change="emit('editText', asset.id, $event)"
      @remove="emit('removeText', asset.id, r.id)"
      @edit-end="editingRegionId = ''"
    />
    <div
      v-if="regionDraft?.id === 'draft' && !isFreeEraser"
      class="image-text-region show-region image-text-draft"
      :class="{ erasing: regionDraft.mode === 'erase' }"
      :style="{
        left: `${regionDraft.x * 100}%`,
        top: `${regionDraft.y * 100}%`,
        width: `${regionDraft.width * 100}%`,
        height: `${regionDraft.height * 100}%`,
      }"
    ></div>
    <button
      v-if="selected && tool === 'select'"
      class="image-delete-handle"
      type="button"
      aria-label="删除图片"
      @pointerdown.stop
      @dblclick.stop
      @click.stop="emit('remove', asset.id)"
    >
      ×
    </button>
    <button
      v-if="selected && tool === 'select'"
      class="image-resize-handle"
      type="button"
      aria-label="调整图片大小"
      @pointerdown.stop="begin($event, 'size')"
      @dblclick.stop
    ></button>
  </div>
</template>
