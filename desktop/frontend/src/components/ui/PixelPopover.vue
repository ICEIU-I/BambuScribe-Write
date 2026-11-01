<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import AppIcon from '../AppIcon.vue'
const props = withDefaults(
  defineProps<{ open: boolean; anchor?: HTMLElement; label: string; width?: number }>(),
  { width: 380 },
)
const emit = defineEmits<{ close: [] }>()
const dialog = ref<HTMLDialogElement>()
function position() {
  const el = dialog.value
  if (!el?.open) return
  const anchor = props.anchor?.getBoundingClientRect()
  const width = Math.min(props.width, window.innerWidth - 32)
  el.style.width = `${width}px`
  const height = el.getBoundingClientRect().height
  el.style.left = `${Math.max(16, Math.min((anchor?.right ?? width + 16) - width, window.innerWidth - width - 16))}px`
  el.style.top = `${Math.max(16, Math.min((anchor?.bottom ?? 16) + 12, window.innerHeight - height - 16))}px`
}
function outside(event: MouseEvent) {
  const box = dialog.value?.getBoundingClientRect()
  if (
    box &&
    (event.clientX < box.left ||
      event.clientX > box.right ||
      event.clientY < box.top ||
      event.clientY > box.bottom)
  )
    emit('close')
}
watch(
  () => props.open,
  async (open) => {
    await nextTick()
    if (open) {
      dialog.value?.showModal()
      position()
      window.addEventListener('resize', position)
    } else {
      dialog.value?.close()
      window.removeEventListener('resize', position)
    }
  },
  { immediate: true },
)
onBeforeUnmount(() => window.removeEventListener('resize', position))
</script>
<template>
  <Teleport to="body">
    <dialog
      ref="dialog"
      class="pixel-popover"
      :aria-label="label"
      @cancel.prevent="emit('close')"
      @click="outside"
    >
      <div class="popup-heading">
        <strong>{{ label }}</strong
        ><button class="icon-button" :aria-label="`关闭${label}`" @click="emit('close')">
          <AppIcon name="close" :size="16" />
        </button>
      </div>
      <slot />
    </dialog>
  </Teleport>
</template>
