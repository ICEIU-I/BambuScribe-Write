<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { errorText, native } from '../../services/bridge'
import type { Point } from '../../types'
const props = defineProps<{ text: string; fontId?: string }>()
const strokes = ref<Point[][]>([]),
  error = ref(''),
  loading = ref(false)
let generation = 0
watch(
  () => [props.text, props.fontId],
  async () => {
    const ticket = ++generation
    strokes.value = []
    error.value = ''
    loading.value = true
    try {
      const doc = await native().PreviewWritingRule(props.text, props.fontId ?? '')
      if (ticket === generation) strokes.value = doc.pages[0]?.strokes ?? []
    } catch (e) {
      if (ticket === generation) error.value = errorText(e)
    } finally {
      if (ticket === generation) loading.value = false
    }
  },
  { immediate: true },
)
onBeforeUnmount(() => generation++)
const viewBox = computed(() => {
  const points = strokes.value.flat()
  if (!points.length) return '0 0 100 30'
  const x = Math.min(...points.map((p) => p.x)),
    y = Math.min(...points.map((p) => p.y))
  const width = Math.max(...points.map((p) => p.x)) - x,
    height = Math.max(...points.map((p) => p.y)) - y
  const frameHeight = Math.max(16, height + 8)
  return `${x - 4} ${y - (frameHeight - height) / 2} ${Math.max(70, width + 8)} ${frameHeight}`
})
</script>
<template>
  <div class="rule-preview" :aria-busy="loading">
    <span v-if="loading" class="preview-status">正在排版…</span>
    <span v-else-if="error" class="preview-status" role="alert">{{ error }}</span>
    <svg
      v-else
      :viewBox="viewBox"
      preserveAspectRatio="xMinYMid meet"
      role="img"
      aria-label="示例实际手写预览"
    >
      <path
        v-for="(stroke, i) in strokes"
        :key="i"
        :d="stroke.map((p, j) => `${j ? 'L' : 'M'}${p.x},${p.y}`).join(' ')"
        fill="none"
        stroke="currentColor"
        stroke-width="0.26"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </svg>
  </div>
</template>
<style scoped>
.rule-preview {
  height: 120px;
  display: grid;
  align-items: center;
  background: var(--paper);
  color: var(--paper-ink);
  border-radius: 8px;
  padding: 6px 10px;
}
.rule-preview svg {
  width: 100%;
  height: 100%;
  min-height: 0;
}
.preview-status {
  font-size: 13px;
  color: #666;
  padding: 12px;
}
</style>
