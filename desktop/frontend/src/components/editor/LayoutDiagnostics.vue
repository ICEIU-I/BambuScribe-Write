<script setup lang="ts">
import { computed } from 'vue'
import type { LayoutDiagnostic } from '../../types'

const props = defineProps<{
  diagnostics: LayoutDiagnostic[]
  width: number
  height: number
}>()

const markers = computed(() =>
  props.diagnostics.map((diagnostic) => {
    const left = Math.max(0, diagnostic.x)
    const right = Math.min(props.width, diagnostic.x + diagnostic.width)
    const y = Math.min(props.height - 0.5, diagnostic.y + diagnostic.height + 0.5)
    let d = `M${left} ${y}`
    let direction = -1
    for (let x = left; x < right; x += 0.75) {
      const end = Math.min(right, x + 0.75)
      d += `Q${(x + end) / 2} ${y + direction * 0.6} ${end} ${y}`
      direction *= -1
    }
    return { ...diagnostic, d }
  }),
)
</script>

<template>
  <svg
    v-if="markers.length"
    class="layout-diagnostics"
    :viewBox="`0 0 ${width} ${height}`"
    preserveAspectRatio="none"
    role="img"
    aria-label="排版问题位置"
  >
    <path
      v-for="(marker, index) in markers"
      :key="index"
      :data-layout-issue="marker.code"
      :d="marker.d"
      vector-effect="non-scaling-stroke"
    >
      <title>{{ marker.message }}</title>
    </path>
  </svg>
</template>

<style scoped>
.layout-diagnostics {
  position: absolute;
  inset: 0;
  z-index: 2;
  width: 100%;
  height: 100%;
  overflow: hidden;
  pointer-events: none;
  fill: none;
  stroke: #e33e38;
  stroke-width: 1.5px;
  stroke-linecap: round;
  stroke-linejoin: round;
}
</style>
