<script setup lang="ts">
import type { SourceImage } from '../../services/calculationTypes'
const props = defineProps<{ title: string; images: SourceImage[]; disabled: boolean }>()
const emit = defineEmits<{ add: [File[]]; change: [SourceImage[]]; view: [SourceImage] }>()
function paste(e: ClipboardEvent) {
  const files = [...(e.clipboardData?.files ?? [])].filter((f) => f.type.startsWith('image/'))
  if (files.length && !props.disabled) {
    e.preventDefault()
    e.stopPropagation()
    emit('add', files)
  }
}
function choose(e: Event) {
  const target = e.target as HTMLInputElement
  emit('add', [...(target.files ?? [])])
  target.value = ''
}
function move(index: number, direction: number) {
  const next = [...props.images]
  const item = next.splice(index, 1)[0]
  next.splice(index + direction, 0, item)
  emit('change', next)
}
</script>
<template>
  <section
    class="calculation-images"
    tabindex="0"
    :aria-label="title"
    @paste="paste"
    @dragover.prevent
    @drop.prevent="!disabled && emit('add', [...($event.dataTransfer?.files ?? [])])"
  >
    <div class="calculation-section-heading">
      <h3>{{ title }}</h3>
      <label class="button compact"
        >添加图片<input
          type="file"
          accept="image/png,image/jpeg"
          multiple
          :aria-label="`添加${title}`"
          :disabled="disabled"
          @change="choose"
      /></label>
    </div>
    <span v-if="!images.length" class="calculation-drop-hint">点击此处后粘贴，或拖入多张图片</span>
    <div class="calculation-thumbnails">
      <div v-for="(image, i) in images" :key="image.id" class="calculation-thumbnail">
        <button
          class="calculation-image-open"
          :aria-label="`查看${title}第${i + 1}张`"
          @click="emit('view', image)"
        >
          <img :src="image.data" :alt="`${title}第${i + 1}张`" />
        </button>
        <span>{{ i + 1 }}. {{ image.name }}</span>
        <div>
          <button
            class="icon-button"
            :aria-label="`前移${title}第${i + 1}张`"
            :disabled="disabled || i === 0"
            @click="move(i, -1)"
          >
            ←</button
          ><button
            class="icon-button"
            :aria-label="`后移${title}第${i + 1}张`"
            :disabled="disabled || i === images.length - 1"
            @click="move(i, 1)"
          >
            →</button
          ><button
            class="icon-button"
            :aria-label="`移除${title}第${i + 1}张`"
            :disabled="disabled"
            @click="
              emit(
                'change',
                images.filter((v) => v.id !== image.id),
              )
            "
          >
            ×
          </button>
        </div>
      </div>
    </div>
  </section>
</template>
