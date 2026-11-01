<script setup lang="ts">
import type { DiagramTask } from '../../state/useDiagramTask'
defineProps<{ task: DiagramTask }>()
</script>
<template>
  <div
    v-if="!task.open.value && task.retained.value && (task.working.value || task.error.value)"
    class="diagram-background"
    aria-label="后台图像导入"
  >
    <progress v-if="task.working.value" aria-label="后台图像识别进度" />
    <button class="text-button" aria-label="查看图像导入任务" @click="task.resume">
      {{ task.working.value ? `图像识别中 · ${task.elapsed.value}` : '图像导入失败' }} · 查看
    </button>
    <button v-if="task.working.value" class="text-button cancel-diagram" @click="task.cancel">
      取消识别
    </button>
  </div>
</template>
<style scoped>
.diagram-background {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 24px;
  border-top: 1px solid var(--border);
  font-size: 12px;
}
.diagram-background progress {
  width: 64px;
  height: 6px;
}
.cancel-diagram {
  margin-left: auto;
}
</style>
