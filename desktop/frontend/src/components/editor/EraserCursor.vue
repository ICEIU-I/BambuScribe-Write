<script setup lang="ts">
import type { Point } from '../../types'

defineProps<{ point: Point; width: number; height: number; diameter: number }>()
</script>

<template>
  <!-- Keep hover repaints outside the masked ink SVG, especially in WKWebView. -->
  <div
    class="eraser-cursor-dot"
    aria-hidden="true"
    :style="{
      width: `${diameter}px`,
      height: `${diameter}px`,
      transform: `translate3d(${point.x * width}px, ${point.y * height}px, 0) translate(-50%, -50%)`,
    }"
  ></div>
</template>

<style scoped>
.eraser-cursor-dot {
  position: absolute;
  left: 0;
  top: 0;
  z-index: 2;
  box-sizing: border-box;
  border: 1.5px solid #fa423e;
  border-radius: 50%;
  background: rgb(250 66 62 / 12%);
  pointer-events: none;
  will-change: transform;
}
</style>
