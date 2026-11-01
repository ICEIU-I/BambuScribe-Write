<script setup lang="ts">
import { pageInkKey } from '../../state/useInkCuts'
import { cutInk } from '../../state/inkCuts'
import { computed, nextTick, onBeforeUnmount, onMounted, provide, ref, watch } from 'vue'
import type {
  TableAsset,
  PageErasure,
  TableSelection,
  EraseMode,
  ImageAsset,
  ImagePlacement,
  ImageEditTool,
  ImageTextRegion,
  LayoutSettings,
  Page,
  Point,
} from '../../types'
import ImageOverlay from './ImageOverlay.vue'
import PaperEraser from './PaperEraser.vue'
import { pageErasuresKey, paperMask } from '../../state/pageEraser'
import { targetErasure } from '../../state/erasureTargets'
import TableOverlay from '../tables/TableOverlay.vue'
import FreeTextBox from './FreeTextBox.vue'
import PaperMarquee from './PaperMarquee.vue'
import type { ObjectSelection } from '../../state/objectSelection'
import { isFreeTextBox } from '../../state/freeTextBox'
const props = withDefaults(
  defineProps<{
    page: Page
    settings: LayoutSettings
    scale: number
    guides: boolean
    ruled: boolean
    selectionStart: number
    selectionEnd: number
    tables?: TableAsset[]
    erasures?: PageErasure[]
    selectedTableId?: string
    newTextBoxId?: string
    images: ImageAsset[]
    imagePreviews: ImagePlacement[]
    selectedImageId: string
    selectedTextId: string
    tool: ImageEditTool
    eraseMode: EraseMode
    eraserRadius: number
    disabled: boolean
    marquee?: ObjectSelection | null
  }>(),
  { erasures: () => [], tables: () => [], selectedTableId: '', newTextBoxId: '' },
)
const emit = defineEmits<{
  addErasure: [PageErasure]
  eraseStart: [number]
  selectTable: [string]
  tableSelection: [TableSelection | null]
  changeTable: [TableAsset]
  removeTable: [string]
  tableSettings: []
  textBoxEditEnd: []
  notice: [string]
  undo: []
  redo: []
  selectImage: [string, Point?]
  selectText: [string, string]
  removeText: [string, string]
  changeImage: [ImageAsset]
  removeImage: [string]
  editImageText: [string, ImageTextRegion]
  addImageText: [string, ImageTextRegion]
}>()
const eraseDraft = ref<PageErasure | null>(null)
const erasures = computed(() => [
  ...props.erasures,
  ...(eraseDraft.value ? [eraseDraft.value] : []),
])
provide(pageErasuresKey, erasures)
provide(
  pageInkKey,
  computed(() => props.page.ink ?? []),
)
const textMask = computed(() =>
  paperMask(erasures.value, {
    page: props.page.number - 1,
    x: 0,
    y: 0,
    width: props.settings.paperWidth,
    height: props.settings.paperHeight,
  }),
)
const canvas = ref<HTMLCanvasElement>(),
  inkCanvas = ref<HTMLCanvasElement>(),
  surface = ref<HTMLDivElement>(),
  visible = ref(false)
let observer: IntersectionObserver | undefined
const selected = computed(() =>
  props.marquee || props.tool === 'erase' || props.selectionStart === props.selectionEnd
    ? []
    : props.page.cells.filter((c) => c.end > props.selectionStart && c.start < props.selectionEnd),
)
const pageAssets = computed(() => props.images.filter((i) => i.page === props.page.number - 1))
async function draw() {
  await nextTick()
  const el = canvas.value,
    ink = inkCanvas.value
  if (!el || !ink) return
  if (!visible.value) {
    el.width = ink.width = 1
    el.height = ink.height = 1
    return
  }
  const s = props.settings,
    dpr = window.devicePixelRatio || 1
  el.width = Math.max(1, Math.round(s.paperWidth * props.scale * dpr))
  el.height = Math.max(1, Math.round(s.paperHeight * props.scale * dpr))
  ink.width = el.width
  ink.height = el.height
  let ctx = el.getContext('2d')
  if (!ctx) return
  ctx.scale(props.scale * dpr, props.scale * dpr)
  ctx.fillStyle = '#fff'
  ctx.fillRect(0, 0, s.paperWidth, s.paperHeight)
  if (props.ruled) {
    ctx.strokeStyle = '#afafaf'
    ctx.lineWidth = Math.max(0.32, 0.8 / props.scale)
    ctx.beginPath()
    for (
      let y = s.marginTop + Math.min(s.fontSize, props.page.lineHeight);
      y <= s.paperHeight - s.marginBottom;
      y += props.page.lineHeight
    ) {
      // Paper guides stop at native tables; only the table's own rules remain.
      const covered = props.tables
        .filter((t) => t.page === props.page.number - 1 && !isFreeTextBox(t))
        .map((t) => {
          const preview = props.page.tables?.find((p) => p.id === t.id)
          return {
            x: t.x,
            y: t.y,
            width: t.columnWidths.reduce((a, b) => a + b, 0),
            height: (preview?.rowHeights ?? t.rowHeights).reduce((a, b) => a + b, 0),
          }
        })
        .filter((t) => y >= t.y && y <= t.y + t.height)
        .sort((a, b) => a.x - b.x)
      let x = s.marginLeft
      for (const t of covered) {
        if (t.x > x) {
          ctx.moveTo(x, y)
          ctx.lineTo(Math.min(t.x, s.paperWidth - s.marginRight), y)
        }
        x = Math.max(x, t.x + t.width)
      }
      if (x < s.paperWidth - s.marginRight) {
        ctx.moveTo(x, y)
        ctx.lineTo(s.paperWidth - s.marginRight, y)
      }
    }
    ctx.stroke()
  }
  if (props.guides) {
    ctx.strokeStyle = '#dfdfdf'
    ctx.lineWidth = 0.2
    ctx.setLineDash([1.5, 1.5])
    ctx.strokeRect(
      s.marginLeft,
      s.marginTop,
      s.paperWidth - s.marginLeft - s.marginRight,
      s.paperHeight - s.marginTop - s.marginBottom,
    )
    ctx.setLineDash([])
  }
  ctx = ink.getContext('2d')
  if (!ctx) return
  ctx.scale(props.scale * dpr, props.scale * dpr)
  ctx.strokeStyle = '#1a1c1f'
  ctx.lineWidth = 0.26
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'
  ctx.beginPath()
  const body =
    props.page.ink
      ?.filter((l) => !l.objectId)
      .flatMap((l) => cutInk(l.strokes, l, props.erasures)) ??
    props.page.textStrokes ??
    props.page.strokes
  for (const stroke of body)
    stroke.forEach((p, i) => (i ? ctx.lineTo(p.x, p.y) : ctx.moveTo(p.x, p.y)))
  ctx.stroke()
  for (const glyph of props.page.missing) {
    ctx.strokeStyle = '#fa423e'
    ctx.lineWidth = 0.3
    ctx.strokeRect(glyph.x, glyph.y, glyph.size, glyph.size)
    ctx.fillStyle = '#fa423e'
    ctx.font = `${glyph.size * 0.65}px sans-serif`
    ctx.textAlign = 'center'
    ctx.fillText('?', glyph.x + glyph.size / 2, glyph.y + glyph.size * 0.75)
  }
}
watch(
  () => [
    props.page,
    props.erasures,
    props.tables,
    props.scale,
    props.guides,
    props.ruled,
    visible.value,
  ],
  draw,
)
onMounted(() => {
  observer = new IntersectionObserver(
    (es) => {
      visible.value = es.some((e) => e.isIntersecting)
    },
    { rootMargin: '700px' },
  )
  if (surface.value) observer.observe(surface.value)
})
onBeforeUnmount(() => observer?.disconnect())
</script>
<template>
  <div
    ref="surface"
    class="paper-page"
    :data-paper="page.number - 1"
    :style="{
      width: `${settings.paperWidth * scale}px`,
      height: `${settings.paperHeight * scale}px`,
    }"
  >
    <canvas ref="canvas" aria-hidden="true" />
    <canvas
      ref="inkCanvas"
      class="paper-text-ink"
      :style="textMask"
      :aria-label="`第 ${page.number} 页手写内容`"
    />
    <div class="selection-layer" aria-hidden="true">
      <div
        v-for="cell in selected"
        :key="cell.start"
        class="text-selection"
        :style="{
          left: `${cell.x * scale}px`,
          top: `${cell.y * scale}px`,
          width: `${cell.width * scale}px`,
          height: `${cell.height * scale}px`,
        }"
      ></div>
    </div>
    <TableOverlay
      v-for="table in tables.filter((t) => t.page === page.number - 1 && !isFreeTextBox(t))"
      :key="table.id"
      :asset="table"
      :preview="page.tables?.find((t) => t.id === table.id)"
      :scale="scale"
      :settings="settings"
      :selected="selectedTableId === table.id"
      :interactive="tool === 'select'"
      :disabled="disabled"
      @select="emit('selectTable', $event)"
      @selection="emit('tableSelection', $event)"
      @change="emit('changeTable', $event)"
      @remove="emit('removeTable', $event)"
      @settings="emit('tableSettings')"
      @notice="emit('notice', $event)"
      @undo="emit('undo')"
      @redo="emit('redo')"
    />
    <FreeTextBox
      v-for="box in tables.filter((t) => t.page === page.number - 1 && isFreeTextBox(t))"
      :key="box.id"
      :asset="box"
      :preview="page.tables?.find((t) => t.id === box.id)"
      :settings="settings"
      :scale="scale"
      :selected="selectedTableId === box.id"
      :interactive="tool !== 'erase'"
      :disabled="disabled"
      :start-editing="newTextBoxId === box.id"
      @select="emit('selectTable', $event)"
      @change="emit('changeTable', $event)"
      @remove="emit('removeTable', $event)"
      @edit-end="emit('textBoxEditEnd')"
      @notice="emit('notice', $event)"
    />
    <ImageOverlay
      v-for="asset in pageAssets"
      :key="asset.id"
      :asset="asset"
      :preview="imagePreviews.find((i) => i.id === asset.id)"
      :scale="scale"
      :settings="settings"
      :selected="selectedImageId === asset.id"
      :selected-text-id="selectedImageId === asset.id ? selectedTextId : ''"
      :tool="tool"
      :erase-mode="eraseMode"
      :eraser-radius="eraserRadius"
      :disabled="disabled"
      @select="(id, p) => emit('selectImage', id, p)"
      @select-text="(id, r) => emit('selectText', id, r)"
      @remove-text="(id, r) => emit('removeText', id, r)"
      @change="emit('changeImage', $event)"
      @edit-text="(id, r) => emit('editImageText', id, r)"
      @add="(id, r) => emit('addImageText', id, r)"
      @remove="emit('removeImage', $event)"
    />
    <PaperEraser
      v-if="tool === 'erase'"
      :page="page.number - 1"
      :width="settings.paperWidth"
      :height="settings.paperHeight"
      :scale="scale"
      :mode="eraseMode"
      :radius="eraserRadius"
      :disabled="disabled"
      @start="emit('eraseStart', page.number - 1)"
      @draft="eraseDraft = $event"
      @add="emit('addErasure', targetErasure($event, page))"
    />
    <span class="paper-page-number" aria-hidden="true">{{ page.number }}</span>
    <PaperMarquee v-if="marquee?.page === page.number - 1" :selection="marquee" :scale="scale" />
  </div>
</template>
