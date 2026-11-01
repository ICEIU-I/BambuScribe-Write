<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import AppIcon from '../AppIcon.vue'
import PageSettings from './PageSettings.vue'
import type { LayoutSettings } from '../../types'
defineProps<{ settings: LayoutSettings }>()
const emit = defineEmits<{ change: [LayoutSettings]; close: [] }>()
const closeButton = ref<HTMLButtonElement>()
let previousFocus: HTMLElement | null = null
onMounted(() => {
  previousFocus = document.getElementById('layout-settings-toggle')
  closeButton.value?.focus({ preventScroll: true })
})
onBeforeUnmount(() => {
  if (previousFocus?.isConnected) previousFocus.focus({ preventScroll: true })
})
</script>
<template>
  <aside
    id="layout-inspector"
    class="layout-inspector"
    role="region"
    aria-label="排版设置"
    @keydown.esc.stop="emit('close')"
  >
    <div class="inspector-heading">
      <div>
        <AppIcon name="settings" :size="16" />
        <h2>排版设置</h2>
      </div>
      <button
        ref="closeButton"
        class="icon-button"
        aria-label="关闭排版设置"
        @click="emit('close')"
      >
        <AppIcon name="close" :size="17" />
      </button>
    </div>
    <PageSettings :settings="settings" @change="emit('change', $event)" @close="emit('close')" />
  </aside>
</template>
