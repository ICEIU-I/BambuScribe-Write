<script setup lang="ts">
import type { ImageTextRegion } from '../../types'
import { eraserPath, eraserRadius } from '../../state/imageEraser'
defineProps<{ id: string; regions: ImageTextRegion[]; width: number; height: number }>()
</script>
<template>
  <mask
    :id="id"
    maskUnits="userSpaceOnUse"
    maskContentUnits="userSpaceOnUse"
    x="0"
    y="0"
    width="1"
    height="1"
    style="mask-type: luminance"
  >
    <rect width="1" height="1" fill="white" />
    <template v-for="region in regions" :key="region.id">
      <!-- A free stroke's bounding box is metadata, never an erasing shape. -->
      <g v-if="region.erasePath?.length" :transform="`scale(${1 / width} ${1 / height})`">
        <path
          :d="eraserPath(region, width, height)"
          fill="none"
          stroke="black"
          :stroke-width="2 * eraserRadius(region) * Math.min(width, height)"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
        <circle
          :cx="region.erasePath[0].x * width"
          :cy="region.erasePath[0].y * height"
          :r="eraserRadius(region) * Math.min(width, height)"
          fill="black"
        />
      </g>
      <rect
        v-else
        :x="region.x"
        :y="region.y"
        :width="region.width"
        :height="region.height"
        fill="black"
      />
    </template>
  </mask>
</template>
