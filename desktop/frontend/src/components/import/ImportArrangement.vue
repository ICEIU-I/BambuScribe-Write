<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import type { ImportPosition, ImportPreview } from '../../services/tableImportTypes'
import { nativeObjectPath } from '../../state/nativeObjectInk'
const props = defineProps<{
  plan: ImportPreview
  positions: ImportPosition[]
  selected: number
  disabled: boolean
}>()
const emit = defineEmits<{ select: [index: number]; change: [positions: ImportPosition[]] }>()
const draft = ref<ImportPosition[]>()
const items = computed(() => draft.value ?? props.positions)
const pageNumbers = computed(() =>
  [...new Set(items.value.map((p) => p.page))].sort((a, b) => a - b),
)
const selectedPosition = computed(() => items.value[props.selected])
let drag:
  | {
      pointer: number
      target: SVGElement
      startX: number
      startY: number
      x: number
      y: number
      scale: number
      index: number
    }
  | undefined
function update(index: number, key: keyof ImportPosition, value: number) {
  const positions = props.positions.map((p) => ({ ...p }))
  positions[index] = { ...positions[index]!, [key]: value }
  emit('change', positions)
}
function begin(e: PointerEvent, index: number) {
  if (props.disabled || e.button !== 0) return
  e.preventDefault()
  emit('select', index)
  const target = e.currentTarget as SVGElement,
    svg = target.closest('svg')!,
    p = props.positions[index]!
  draft.value = props.positions.map((p) => ({ ...p }))
  drag = {
    pointer: e.pointerId,
    target,
    index,
    startX: e.clientX,
    startY: e.clientY,
    x: p.x,
    y: p.y,
    scale: svg.getBoundingClientRect().width / props.plan.settings.paperWidth,
  }
  target.setPointerCapture(e.pointerId)
}
function move(e: PointerEvent) {
  if (!drag || e.pointerId !== drag.pointer || !draft.value) return
  const p = draft.value[drag.index]!,
    s = props.plan.settings,
    group = props.plan.groups[drag.index]!
  p.x = Math.max(
    s.marginLeft,
    Math.min(
      s.paperWidth - s.marginRight - p.width,
      drag.x + (e.clientX - drag.startX) / drag.scale,
    ),
  )
  p.y = Math.max(
    s.marginTop,
    Math.min(
      s.paperHeight - s.marginBottom - group.height,
      drag.y + (e.clientY - drag.startY) / drag.scale,
    ),
  )
}
function finish(e: PointerEvent) {
  if (!drag || drag.pointer !== e.pointerId) return
  move(e)
  const positions = draft.value!
  cancel()
  emit('change', positions)
}
function cancel() {
  if (drag?.target.hasPointerCapture(drag.pointer)) drag.target.releasePointerCapture(drag.pointer)
  drag = undefined
  draft.value = undefined
}
onBeforeUnmount(cancel)
function objects(index: number) {
  const ids = props.plan.groups[index]!.objectIds
  return props.plan.preview.filter((t) => ids.includes(t.id))
}
</script>
<template>
  <section class="import-arrangement" aria-label="导入排版预览">
    <div class="import-position" v-if="selectedPosition">
      <label v-for="field in ['page', 'x', 'y', 'width'] as const" :key="field">
        {{ { page: '页码', x: 'X (mm)', y: 'Y (mm)', width: '宽度 (mm)' }[field] }}
        <input
          type="number"
          :aria-label="`预览${{ page: '页码', x: 'X', y: 'Y', width: '宽度' }[field]}`"
          :value="Number(selectedPosition[field].toFixed(2))"
          :min="field === 'page' ? 1 : 0"
          :step="field === 'page' ? 1 : 1"
          :disabled="disabled"
          @change="update(selected, field, Number(($event.target as HTMLInputElement).value))"
        />
      </label>
    </div>
    <div class="import-papers">
      <div v-for="page in pageNumbers" :key="page" class="import-paper-wrap">
        <small>第 {{ page }} 页</small>
        <svg
          class="import-paper"
          :data-import-page="page"
          :viewBox="`0 0 ${plan.settings.paperWidth} ${plan.settings.paperHeight}`"
          @keydown.esc="cancel"
        >
          <rect
            v-for="(r, i) in plan.occupied.filter((r) => r.page === page)"
            :key="`existing-${i}`"
            :x="r.x"
            :y="r.y"
            :width="r.width"
            :height="r.height"
            fill="#dedede"
          />
          <template v-for="(p, i) in items" :key="i">
            <g
              v-if="p.page === page"
              :data-import-group="i"
              :transform="`translate(${p.x} ${p.y})`"
              tabindex="0"
              :aria-label="`拖动表格 ${i + 1}`"
              role="button"
              @pointerdown="begin($event, i)"
              @pointermove="move"
              @pointerup="finish"
              @pointercancel="cancel"
              @focus="emit('select', i)"
            >
              <rect
                class="import-group-frame"
                :class="{ selected: i === selected }"
                :width="p.width"
                :height="plan.groups[i]!.height"
              />
              <g :transform="`scale(${p.width / plan.groups[i]!.width} 1)`" pointer-events="none">
                <path
                  v-for="o in objects(i)"
                  :key="o.id"
                  :d="nativeObjectPath(o.strokes)"
                  :transform="`translate(${o.x - plan.groups[i]!.x} ${o.y - plan.groups[i]!.y})`"
                  fill="none"
                  stroke="#222"
                  stroke-width="0.25"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </g>
            </g>
          </template>
        </svg>
      </div>
    </div>
  </section>
</template>
