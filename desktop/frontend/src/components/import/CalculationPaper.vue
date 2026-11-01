<script setup lang="ts">
import type { Document } from '../../types'
defineProps<{ document: Document }>()
</script>
<template>
  <div class="calculation-paper-list" aria-label="数据处理预览">
    <svg
      v-for="page in document.pages"
      :key="page.number"
      class="calculation-paper"
      :viewBox="`0 0 ${document.settings.paperWidth} ${document.settings.paperHeight}`"
      :aria-label="`预览第${page.number}页`"
      role="img"
    >
      <rect width="100%" height="100%" fill="white" />
      <path
        v-for="(stroke, i) in page.strokes"
        :key="i"
        :d="stroke.map((p, j) => `${j ? 'L' : 'M'}${p.x},${p.y}`).join(' ')"
        fill="none"
        stroke="#1a1c1f"
        stroke-width="0.26"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </svg>
  </div>
</template>
