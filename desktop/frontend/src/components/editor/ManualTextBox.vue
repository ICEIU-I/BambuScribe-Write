<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import type { ImageTextRegion, Point } from '../../types'
import { transformTextBox } from '../../state/textBox'
const props = defineProps<{
  region: ImageTextRegion
  selected: boolean
  interactive: boolean
  disabled: boolean
  startEditing: boolean
  locate: (event: { clientX: number; clientY: number }) => Point
}>()
const emit = defineEmits<{
  select: []
  draft: [ImageTextRegion | null]
  change: [ImageTextRegion]
  remove: []
  editEnd: []
}>()
const host = ref<HTMLElement>(),
  input = ref<HTMLTextAreaElement>()
const editing = ref(false),
  text = ref('')
const style = computed(() => ({
  left: `${props.region.x * 100}%`,
  top: `${props.region.y * 100}%`,
  width: `${props.region.width * 100}%`,
  height: `${props.region.height * 100}%`,
  pointerEvents: props.interactive || editing.value ? ('auto' as const) : ('none' as const),
}))
let gesture:
  | {
      region: ImageTextRegion
      start: Point
      resize: boolean
      pointer: number
      changed?: ImageTextRegion
    }
  | undefined
function begin(e: PointerEvent, resize = false) {
  if (props.disabled || !props.interactive || e.button !== 0 || editing.value) return
  e.stopPropagation()
  host.value?.focus({ preventScroll: true })
  emit('select')
  gesture = { region: { ...props.region }, start: props.locate(e), resize, pointer: e.pointerId }
  host.value?.setPointerCapture(e.pointerId)
  window.addEventListener('pointermove', move)
  window.addEventListener('pointerup', finish)
  window.addEventListener('pointercancel', cancel)
}
function move(e: PointerEvent) {
  const g = gesture
  if (!g || e.pointerId !== g.pointer) return
  const p = props.locate(e),
    delta = { x: p.x - g.start.x, y: p.y - g.start.y }
  if (Math.hypot(delta.x, delta.y) < 0.001) return
  e.preventDefault()
  g.changed = transformTextBox(g.region, delta, g.resize)
  emit('draft', g.changed)
}
function cleanup() {
  const g = gesture
  gesture = undefined
  if (g && host.value?.hasPointerCapture(g.pointer)) host.value.releasePointerCapture(g.pointer)
  window.removeEventListener('pointermove', move)
  window.removeEventListener('pointerup', finish)
  window.removeEventListener('pointercancel', cancel)
  emit('draft', null)
}
function finish(e: PointerEvent) {
  if (!gesture || gesture.pointer !== e.pointerId) return
  const value = gesture.changed
  cleanup()
  if (value) emit('change', value)
}
function cancel() {
  cleanup()
}
async function edit() {
  if (props.disabled) return
  emit('select')
  editing.value = true
  text.value = props.region.text
  await nextTick()
  input.value?.focus({ preventScroll: true })
  input.value?.select()
}
function save(focus = true) {
  if (!editing.value) return
  editing.value = false
  emit('editEnd')
  const value = text.value.trim()
  if (value !== props.region.text) emit('change', { ...props.region, text: value })
  if (focus) host.value?.focus({ preventScroll: true })
}
function cancelEdit() {
  editing.value = false
  emit('editEnd')
  host.value?.focus({ preventScroll: true })
}
function key(e: KeyboardEvent) {
  if (e.isComposing) return
  if (e.key === 'Enter') {
    e.preventDefault()
    e.stopPropagation()
    save()
  } else if (e.key === 'Escape') {
    e.preventDefault()
    e.stopPropagation()
    cancelEdit()
  }
}
watch(
  () => props.startEditing,
  (open) => {
    if (open) edit()
  },
  { immediate: true },
)
onBeforeUnmount(cleanup)
</script>
<template>
  <div
    ref="host"
    class="manual-text-box"
    :class="{ 'is-selected': selected && interactive, 'is-editing': editing }"
    :style="style"
    :data-text-box="region.id"
    tabindex="-1"
    :aria-label="`文本框：${region.text || '空白'}`"
    @pointerdown="begin($event)"
    @dblclick.stop="edit"
  >
    <template v-if="selected && interactive && !editing">
      <button
        type="button"
        class="text-box-delete"
        aria-label="删除文本框"
        @pointerdown.stop
        @dblclick.stop
        @click.stop="emit('remove')"
      >
        ×
      </button>
      <button
        type="button"
        class="text-box-resize"
        aria-label="调整文本框大小"
        @pointerdown.stop="begin($event, true)"
        @dblclick.stop
      ></button>
    </template>
    <textarea
      v-if="editing"
      ref="input"
      v-model="text"
      class="image-inline-editor"
      aria-label="输入图片文字"
      rows="1"
      maxlength="500"
      @pointerdown.stop
      @click.stop
      @dblclick.stop
      @keydown="key"
      @blur="save(false)"
    />
  </div>
</template>
