<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import type { EraseMode, PageErasure, Point } from '../../types'
import { erasureBounds, paperBrushRadius } from '../../state/pageEraser'
import EraserCursor from './EraserCursor.vue'
const props = defineProps<{
  page: number
  width: number
  height: number
  scale: number
  mode: EraseMode
  radius: number
  disabled: boolean
}>()
const emit = defineEmits<{ start: []; draft: [PageErasure | null]; add: [PageErasure] }>()
const host = ref<HTMLElement>(),
  point = ref<Point | null>(null),
  draft = ref<PageErasure | null>(null)
let pointer: number | undefined
let start: Point | undefined
function locate(e: PointerEvent): Point {
  const r = host.value!.getBoundingClientRect()
  return {
    x: Math.max(0, Math.min(props.width, ((e.clientX - r.left) * props.width) / r.width)),
    y: Math.max(0, Math.min(props.height, ((e.clientY - r.top) * props.height) / r.height)),
  }
}
function update(value: PageErasure | null) {
  draft.value = value
  emit('draft', value)
}
function begin(e: PointerEvent) {
  if (props.disabled || e.button !== 0 || pointer !== undefined) return
  e.preventDefault()
  emit('start')
  pointer = e.pointerId
  start = locate(e)
  point.value = start
  host.value?.setPointerCapture(pointer)
  update({
    id: 'draft',
    page: props.page,
    x: start.x,
    y: start.y,
    width: 0,
    height: 0,
    ...(props.mode === 'free'
      ? { path: [start, start], radius: paperBrushRadius(props.radius) }
      : {}),
  })
}
function move(e: PointerEvent) {
  point.value = locate(e)
  if (e.pointerId !== pointer || !start || !draft.value) return
  e.preventDefault()
  const p = point.value
  if (props.mode === 'free') {
    const path = [...draft.value.path!]
    const last = path.at(-1)!
    if (Math.hypot(last.x - p.x, last.y - p.y) < 0.15) return
    path.push(p)
    update({ ...draft.value, ...erasureBounds(path), path })
  } else update({ ...draft.value, ...erasureBounds([start, p]) })
}
function clear() {
  const id = pointer
  pointer = undefined
  start = undefined
  if (id !== undefined && host.value?.hasPointerCapture(id)) host.value.releasePointerCapture(id)
  update(null)
}
function finish(e: PointerEvent) {
  if (pointer !== e.pointerId) return
  move(e)
  const value = draft.value
  if (value && (value.path?.length || (value.width > 0.1 && value.height > 0.1)))
    emit('add', { ...value, id: `erase-${crypto.randomUUID()}` })
  clear()
}
watch(() => [props.mode, props.radius, props.disabled], clear)
onBeforeUnmount(clear)
</script>
<template>
  <div
    ref="host"
    class="paper-eraser"
    :class="{ free: mode === 'free' }"
    aria-label="纸面擦除"
    @pointerdown.stop="begin"
    @pointermove.stop="move"
    @pointerup.stop="finish"
    @pointercancel.stop="clear"
    @lostpointercapture="clear"
    @pointerleave="point = null"
  >
    <div
      v-if="draft && !draft.path"
      class="paper-erase-box"
      :style="{
        left: `${draft.x * scale}px`,
        top: `${draft.y * scale}px`,
        width: `${draft.width * scale}px`,
        height: `${draft.height * scale}px`,
      }"
    />
    <EraserCursor
      v-if="mode === 'free' && point"
      :point="{ x: point.x / width, y: point.y / height }"
      :width="width * scale"
      :height="height * scale"
      :diameter="paperBrushRadius(radius) * 2 * scale"
    />
  </div>
</template>
<style scoped>
.paper-eraser {
  position: absolute;
  inset: 0;
  z-index: 20;
  touch-action: none;
  cursor: crosshair;
}
.paper-eraser.free,
.paper-eraser.free * {
  cursor: none;
}
.paper-erase-box {
  position: absolute;
  border: 1px dashed #fa423e;
  background: rgb(250 66 62 / 8%);
  pointer-events: none;
}
</style>
