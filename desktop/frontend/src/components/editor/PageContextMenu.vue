<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import AppIcon from '../AppIcon.vue'
const props = defineProps<{ page: number; x: number; y: number; disabled: boolean }>()
const emit = defineEmits<{ close: []; delete: []; copy: []; blank: [] }>()
const menu = ref<HTMLDialogElement>()
function position() {
  const el = menu.value
  if (!el) return
  const rect = el.getBoundingClientRect()
  el.style.left = `${Math.max(8, Math.min(props.x, window.innerWidth - rect.width - 8))}px`
  el.style.top = `${Math.max(8, Math.min(props.y, window.innerHeight - rect.height - 8))}px`
}
function outside(event: MouseEvent) {
  if (event.target === menu.value) emit('close')
}
function keydown(event: KeyboardEvent) {
  if (!['ArrowUp', 'ArrowDown', 'Home', 'End'].includes(event.key)) return
  event.preventDefault()
  const items = [
    ...(menu.value?.querySelectorAll<HTMLButtonElement>('button:not(:disabled)') ?? []),
  ]
  const index = items.indexOf(document.activeElement as HTMLButtonElement)
  const target =
    event.key === 'Home'
      ? 0
      : event.key === 'End'
        ? items.length - 1
        : (index + (event.key === 'ArrowUp' ? -1 : 1) + items.length) % items.length
  items[target]?.focus()
}
onMounted(async () => {
  menu.value?.showModal()
  await nextTick()
  position()
  window.addEventListener('resize', position)
})
onBeforeUnmount(() => {
  menu.value?.close()
  window.removeEventListener('resize', position)
})
</script>
<template>
  <Teleport to="body">
    <dialog
      ref="menu"
      class="page-context-menu"
      role="menu"
      :aria-label="`第 ${page} 页操作`"
      @cancel.prevent="emit('close')"
      @click="outside"
      @contextmenu.prevent="emit('close')"
      @keydown="keydown"
    >
      <button role="menuitem" :disabled="disabled" @click="emit('copy')">
        <AppIcon name="copy" :size="16" />复制整页
      </button>
      <button role="menuitem" :disabled="disabled" @click="emit('blank')">
        <AppIcon name="file-plus" :size="16" />插入空白页
      </button>
      <button class="delete-page" role="menuitem" :disabled="disabled" @click="emit('delete')">
        <AppIcon name="trash" :size="16" />删除整页
      </button>
    </dialog>
  </Teleport>
</template>
<style scoped>
.page-context-menu {
  position: fixed;
  margin: 0;
  padding: 4px;
  min-width: 152px;
  border: 1px solid var(--border-heavy);
  border-radius: 8px;
  color: var(--text);
  background: var(--elevated);
  box-shadow: var(--shadow);
}
.page-context-menu::backdrop {
  background: transparent;
}
.page-context-menu button {
  display: flex;
  align-items: center;
  gap: 9px;
  width: 100%;
  padding: 8px 10px;
  border: 0;
  border-radius: 4px;
  color: var(--text);
  background: transparent;
  font-size: 13px;
  text-align: left;
}
.page-context-menu button:hover {
  background: var(--surface-hover);
}
.page-context-menu .delete-page {
  color: var(--danger-text);
}
</style>
