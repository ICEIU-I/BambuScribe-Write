<script setup lang="ts">
import type { ObjectSelection } from '../../state/objectSelection'
defineProps<{ selection: ObjectSelection; scale: number }>()
</script>
<template>
  <div
    class="paper-marquee-layer"
    aria-label="框选对象"
    :data-selected-count="selection.objects.length"
  >
    <div
      v-for="object in selection.objects"
      :key="object.id"
      class="marquee-object"
      :data-marquee-object="object.id"
      :style="{
        left: `${object.x * scale}px`,
        top: `${object.y * scale}px`,
        width: `${object.width * scale}px`,
        height: `${object.height * scale}px`,
      }"
    />
    <div
      class="paper-marquee"
      :style="{
        left: `${selection.x * scale}px`,
        top: `${selection.y * scale}px`,
        width: `${selection.width * scale}px`,
        height: `${selection.height * scale}px`,
      }"
    />
  </div>
</template>
<style scoped>
.paper-marquee-layer {
  position: absolute;
  inset: 0;
  pointer-events: none;
  z-index: 25;
}
.paper-marquee {
  position: absolute;
  border: 1px dashed var(--info);
  background: rgb(51 156 255 / 12%);
}
.marquee-object {
  position: absolute;
  outline: 2px solid var(--info);
  background: rgb(51 156 255 / 10%);
}
</style>
