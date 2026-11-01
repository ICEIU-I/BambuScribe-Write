<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import type { SourceImage } from '../../services/calculationTypes'
const props = defineProps<{ image?: SourceImage }>()
const emit = defineEmits<{ close: [] }>()
const dialog = ref<HTMLDialogElement>(),
  originalSize = ref(false)
watch(
  () => props.image,
  async (image) => {
    originalSize.value = false
    await nextTick()
    if (image) dialog.value?.showModal()
    else dialog.value?.close()
  },
)
</script>
<template>
  <dialog
    ref="dialog"
    class="calculation-image-viewer"
    aria-label="查看原图"
    @cancel.stop.prevent="emit('close')"
  >
    <div class="calculation-heading">
      <strong>{{ image?.name }}</strong>
      <button class="button compact" @click="originalSize = !originalSize">
        {{ originalSize ? '适应窗口' : '原始大小' }}
      </button>
      <button class="icon-button" aria-label="关闭原图" @click="emit('close')">×</button>
    </div>
    <div class="calculation-image-scroll">
      <img v-if="image" :class="{ original: originalSize }" :src="image.data" :alt="image.name" />
    </div>
  </dialog>
</template>
<style scoped>
.calculation-image-viewer {
  max-width: 95vw;
  max-height: 95vh;
  padding: 18px;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--surface);
  color: var(--text);
}
.calculation-image-viewer::backdrop {
  background: rgb(0 0 0 / 40%);
}
.calculation-heading strong {
  margin-right: auto;
}
.calculation-image-scroll {
  overflow: auto;
  max-width: 88vw;
  max-height: 78vh;
}
img {
  display: block;
  max-width: 88vw;
  max-height: 78vh;
  object-fit: contain;
}
img.original {
  max-width: none;
  max-height: none;
}
</style>
