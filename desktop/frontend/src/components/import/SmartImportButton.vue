<script setup lang="ts">
import { ref } from 'vue'
import AppIcon from '../AppIcon.vue'
import PixelPopover from '../ui/PixelPopover.vue'
defineProps<{ disabled: boolean }>()
const emit = defineEmits<{ table: []; diagram: [] }>()
const open = ref(false),
  button = ref<HTMLButtonElement>()
function choose(kind: 'table' | 'diagram') {
  open.value = false
  if (kind === 'table') emit('table')
  else emit('diagram')
}
</script>
<template>
  <button
    ref="button"
    class="button tool-button import-table-button"
    :disabled="disabled"
    :aria-expanded="open"
    aria-haspopup="dialog"
    @mousedown.prevent
    @click="open = true"
  >
    <AppIcon name="image" :size="16" /><span>智能导入</span>
  </button>
  <PixelPopover :open="open" :anchor="button" :width="220" label="智能导入" @close="open = false">
    <div class="import-choices">
      <button class="button" @click="choose('table')">
        <AppIcon name="table" :size="16" />导入表格
      </button>
      <button class="button" @click="choose('diagram')">
        <AppIcon name="image" :size="16" />导入图像
      </button>
    </div>
  </PixelPopover>
</template>
<style scoped>
.import-choices {
  display: grid;
  gap: 8px;
  padding: 8px 0;
}
.import-choices button {
  justify-content: flex-start;
  gap: 10px;
}
</style>
