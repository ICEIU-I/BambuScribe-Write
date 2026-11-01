<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, useId, watch } from 'vue'
import { activeHint } from '../../state/activeHint'
defineProps<{ text: string }>()
const id = useId()
const trigger = ref<HTMLElement>()
const popup = ref<HTMLElement>()
const shown = computed(() => activeHint.value === id)
const position = ref({ left: '0px', top: '0px' })
let timer: ReturnType<typeof setTimeout> | undefined
function clearTimer() {
  clearTimeout(timer)
}
function hide() {
  clearTimer()
  if (shown.value) activeHint.value = null
}
function show(delay = 300) {
  clearTimer()
  timer = setTimeout(async () => {
    if (!trigger.value?.getClientRects().length) return
    activeHint.value = id
    await nextTick()
    if (!shown.value || !popup.value || !trigger.value) return
    const anchor = trigger.value.getBoundingClientRect()
    const hint = popup.value.getBoundingClientRect()
    const left =
      anchor.right + 10 + hint.width <= window.innerWidth - 8
        ? anchor.right + 10
        : Math.max(8, anchor.left - hint.width - 10)
    const top = Math.max(
      8,
      Math.min(
        window.innerHeight - hint.height - 8,
        anchor.top + (anchor.height - hint.height) / 2,
      ),
    )
    position.value = { left: `${left}px`, top: `${top}px` }
  }, delay)
}
function focus(event: FocusEvent) {
  if ((event.target as HTMLElement).matches(':focus-visible')) show(0)
}
function leave() {
  clearTimer()
  timer = setTimeout(hide, 100)
}
function dismiss(event: KeyboardEvent) {
  if (event.key === 'Escape') hide()
}
function keydown(event: KeyboardEvent) {
  if (['Escape', 'Enter', ' '].includes(event.key)) hide()
}
watch(shown, (visible) => {
  if (visible) {
    window.addEventListener('scroll', hide, true)
    window.addEventListener('resize', hide)
    window.addEventListener('blur', hide)
    window.addEventListener('keydown', dismiss)
  } else cleanup()
})
function cleanup() {
  window.removeEventListener('scroll', hide, true)
  window.removeEventListener('resize', hide)
  window.removeEventListener('blur', hide)
  window.removeEventListener('keydown', dismiss)
}
onBeforeUnmount(() => {
  hide()
  cleanup()
})
</script>
<template>
  <span
    ref="trigger"
    class="hint-trigger"
    @pointerenter="show()"
    @pointerleave="leave"
    @pointerdown="hide"
    @focusin="focus"
    @focusout="hide"
    @keydown="keydown"
  >
    <slot :described-by="shown ? id : undefined" />
    <Teleport to="body">
      <span
        v-if="shown"
        :id="id"
        ref="popup"
        role="tooltip"
        class="hover-hint"
        :style="position"
        @pointerenter="clearTimer"
        @pointerleave="hide"
        >{{ text }}</span
      >
    </Teleport>
  </span>
</template>
