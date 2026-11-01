<script setup lang="ts">
import { computed, reactive, ref, shallowRef } from 'vue'
import DocumentCanvas from '../../src/components/editor/DocumentCanvas.vue'
import { defaultLayout } from '../../src/state/defaults'
import type { Document, ImageAsset, ImagePlacement } from '../../src/types'
import '../../src/styles/base.css'
import '../../src/styles/paper-editor.css'

const canvas = ref<InstanceType<typeof DocumentCanvas>>()
const state = reactive({
  text: '正文保持不变',
  index: 2,
  zoom: 65,
  disabled: false,
  active: true,
  images: [
    {
      id: 'diagram',
      page: 2,
      data: 'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==',
      x: 35,
      y: 10,
      width: 70,
      height: 40,
      rotation: 0,
      mode: 'line',
      textRegions: [
        {
          id: 'erase',
          text: '',
          mode: 'erase',
          source: 'manual',
          x: 0.5,
          y: 0.5,
          width: 0.03,
          height: 0.03,
          erasePath: [
            { x: 0.5, y: 0.5 },
            { x: 0.53, y: 0.53 },
          ],
          eraseRadius: 0.02,
        },
        {
          id: 'label',
          text: '光源',
          mode: 'handwriting',
          source: 'manual',
          x: 0.7,
          y: 0.6,
          width: 0.2,
          height: 0.1,
        },
      ],
    },
  ] as ImageAsset[],
})
const held = shallowRef<Document | null>(null)
const preview = computed<Document>(() => ({
  id: 'image-fixture',
  settings: defaultLayout(),
  missing: [],
  characterCount: state.text.length,
  fontName: 'fixture',
  pages: [0, 1, 2].map((page) => ({
    number: page + 1,
    lineCount: 16,
    lineHeight: 15,
    missing: [],
    text: page === 0 ? state.text : '',
    cells: [],
    carets: [{ index: Math.min(page, state.text.length), x: 0, y: 0 }],
    textStrokes: [],
    strokes: [],
    images: state.images
      .filter((im) => im.page === page)
      .map((im): ImagePlacement => {
        const strokes = [
          [
            { x: im.x + im.width * 0.1, y: im.y + im.height * 0.2 },
            { x: im.x + im.width * 0.9, y: im.y + im.height * 0.2 },
          ],
        ]
        return { ...im, strokes, graphicStrokes: strokes, sourceStrokes: strokes }
      }),
  })),
}))
function jump(index: number) {
  state.index = index
  canvas.value?.jump(index)
}
Object.assign(window, {
  imageFixture: {
    state,
    freeze: () => {
      held.value = JSON.parse(JSON.stringify(preview.value))
    },
    thaw: () => {
      held.value = null
    },
    jump,
  },
})
</script>
<template>
  <div style="display: flex; flex-direction: column; height: 100vh">
    <div style="display: flex; gap: 8px; padding: 8px">
      <button @click="canvas?.undo()">撤销</button><button @click="canvas?.redo()">重做</button>
      <button v-for="index in [0, 1, 2]" :key="index" @click="jump(index)">
        第 {{ index + 1 }} 页
      </button>
      <input aria-label="其他输入" /><span data-current-page>{{ state.index + 1 }}</span>
    </div>
    <DocumentCanvas
      ref="canvas"
      v-model="state.text"
      v-model:images="state.images"
      v-model:index="state.index"
      v-model:zoom="state.zoom"
      :document="held ?? preview"
      :guides="false"
      :ruled="false"
      tool="select"
      erase-mode="free"
      :eraser-radius="0.02"
      :active="state.active"
      :disabled="state.disabled"
    />
  </div>
</template>
