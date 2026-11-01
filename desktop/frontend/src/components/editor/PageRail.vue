<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import HoverHint from '../ui/HoverHint.vue'
import PageContextMenu from './PageContextMenu.vue'
import type { Page } from '../../types'
import { usePageReorder } from '../../state/usePageReorder'
const props = defineProps<{ pages: Page[]; index: number; actionsDisabled?: boolean }>()
const emit = defineEmits<{
  page: [number]
  deletePage: [number]
  copyPage: [number]
  blankPage: [number]
  movePage: [number, number]
}>()
const context = ref<{ index: number; x: number; y: number }>()
const characters = (page: Page) => Array.from(page.text.replace(/\f/g, '')).length
function openContext(event: MouseEvent | KeyboardEvent, index: number) {
  event.preventDefault()
  const button = event.currentTarget as HTMLButtonElement
  button.focus({ preventScroll: true })
  const rect = button.getBoundingClientRect()
  context.value = {
    index,
    x: event instanceof MouseEvent ? event.clientX : rect.right,
    y: event instanceof MouseEvent ? event.clientY : rect.top,
  }
}
function contextKey(event: KeyboardEvent, index: number) {
  if (event.key === 'ContextMenu' || (event.shiftKey && event.key === 'F10'))
    openContext(event, index)
}
function deletePage() {
  if (!context.value || props.actionsDisabled) return
  const index = context.value.index
  context.value = undefined
  emit('deletePage', index)
}
function copyPage() {
  if (!context.value || props.actionsDisabled) return
  const index = context.value.index
  context.value = undefined
  emit('copyPage', index)
}
function blankPage() {
  if (!context.value || props.actionsDisabled) return
  const index = context.value.index
  context.value = undefined
  emit('blankPage', index)
}
watch(
  () => props.pages,
  () => {
    context.value = undefined
  },
)
const rail = ref<HTMLElement>()
const reorder = usePageReorder(
  rail,
  () => !!props.actionsDisabled,
  () => props.pages,
  (from, to) => emit('movePage', from, to),
)
const { source, insertion, preview } = reorder
function selectPage(event: MouseEvent, index: number) {
  if (!reorder.click(event)) emit('page', index)
}
function reveal() {
  const button = rail.value?.querySelector<HTMLButtonElement>('[aria-current="page"]')
  if (!button || !rail.value) return
  const top = button.offsetTop
  if (
    top < rail.value.scrollTop ||
    top + button.offsetHeight > rail.value.scrollTop + rail.value.clientHeight
  )
    rail.value.scrollTop = top - rail.value.clientHeight / 2 + button.offsetHeight / 2
}
let resizeObserver: ResizeObserver | undefined
onMounted(() => {
  resizeObserver = new ResizeObserver(reveal)
  if (rail.value) resizeObserver.observe(rail.value)
  reveal()
})
onBeforeUnmount(() => resizeObserver?.disconnect())
watch(
  () => [props.index, props.pages.length],
  async () => {
    await nextTick()
    reveal()
  },
)
async function keydown(event: KeyboardEvent) {
  if (event.defaultPrevented) return
  const last = props.pages.length - 1
  if (last < 0) return
  const index = {
    ArrowUp: Math.max(0, props.index - 1),
    ArrowDown: Math.min(last, props.index + 1),
    ArrowLeft: Math.max(0, props.index - 1),
    ArrowRight: Math.min(last, props.index + 1),
    Home: 0,
    End: last,
  }[event.key]
  if (index === undefined) return
  event.preventDefault()
  emit('page', index)
  await nextTick()
  rail.value?.querySelectorAll<HTMLButtonElement>('button')[index]?.focus({ preventScroll: true })
}
</script>
<template>
  <nav class="page-navigation" aria-label="文档页面">
    <div
      ref="rail"
      class="page-rail"
      :class="{ 'is-reordering': source !== null }"
      @keydown="keydown"
    >
      <HoverHint
        v-for="(page, i) in pages"
        :key="page.number"
        v-slot="hint"
        :text="`第 ${page.number} 页 · ${characters(page)} 字${page.missing.length ? ' · 缺字' : ''}\n${page.text.trim().replace(/\s+/g, ' ').slice(0, 48) || (page.strokes.length ? '图表与手写内容' : '空白页')}`"
      >
        <button
          :class="{
            selected: i === index,
            'has-missing': page.missing.length,
            'page-dragging': i === source,
            'page-drop-before': insertion === i,
            'page-drop-after': insertion === pages.length && i === pages.length - 1,
          }"
          :data-page-index="i"
          :data-reorder-enabled="!actionsDisabled && pages.length > 1"
          :aria-current="i === index ? 'page' : undefined"
          :tabindex="i === index ? 0 : -1"
          :aria-label="`第 ${page.number} 页，${characters(page)} 字${page.missing.length ? '，有缺字' : ''}`"
          :aria-describedby="hint.describedBy"
          @click="selectPage($event, i)"
          @pointerdown="reorder.pointerDown($event, i)"
          @pointermove="reorder.pointerMove"
          @pointerup="reorder.pointerUp"
          @pointercancel="reorder.cancel"
          @lostpointercapture="reorder.cancel"
          @dragstart.prevent
          @contextmenu="openContext($event, i)"
          @keydown="contextKey($event, i)"
        >
          <span class="page-miniature" aria-hidden="true">{{ page.number }}</span>
          <i v-if="page.missing.length" class="page-missing-dot" aria-hidden="true"></i>
        </button>
      </HoverHint>
    </div>
    <Teleport to="body">
      <div
        v-if="preview"
        class="page-drag-preview"
        aria-hidden="true"
        :style="{ left: `${preview.x}px`, top: `${preview.y}px` }"
      >
        <span class="page-miniature">{{ preview.number }}</span>
      </div>
    </Teleport>
    <PageContextMenu
      v-if="context"
      :key="`${context.index}:${context.x}:${context.y}`"
      :page="context.index + 1"
      :x="context.x"
      :y="context.y"
      :disabled="!!actionsDisabled"
      @close="context = undefined"
      @delete="deletePage"
      @copy="copyPage"
      @blank="blankPage"
    />
  </nav>
</template>
