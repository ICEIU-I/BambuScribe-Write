<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { readImageFile } from '../../services/imageFiles'
import type { DiagramTask } from '../../state/useDiagramTask'
import ImportModelConfig from './ImportModelConfig.vue'
import './table-import.css'
const props = defineProps<{ task: DiagramTask; replacing: boolean }>()
const task = props.task
const dialog = ref<HTMLDialogElement>(),
  model = ref<InstanceType<typeof ImportModelConfig>>()
watch(task.open, async (open) => {
  if (!open) {
    dialog.value?.close()
    return
  }
  await nextTick()
  if (task.open.value && !dialog.value?.open) dialog.value?.showModal()
})
// Opening a hidden task only restores its window. Load configuration once per
// newly prepared import, never on every reopen.
watch(task.session, async () => {
  await nextTick()
  await model.value?.load()
})
function choose(files: File[]) {
  if (!files.length) return
  void task.read(async () => {
    if (files.length !== 1) throw new Error('每次导入一张图像，请分次添加')
    return readImageFile(files[0]!)
  })
}
function fileChange(e: Event) {
  const input = e.target as HTMLInputElement
  choose([...(input.files ?? [])])
  input.value = ''
}
function paste(e: ClipboardEvent) {
  const files = [...(e.clipboardData?.files ?? [])].filter((f) => f.type.startsWith('image/'))
  if (!files.length)
    for (const item of e.clipboardData?.items ?? []) {
      if (item.type.startsWith('image/')) {
        const file = item.getAsFile()
        if (file) files.push(file)
      }
    }
  if (files.length) {
    e.preventDefault()
    e.stopPropagation()
    choose(files)
  }
}
function run() {
  return task.run(async () => {
    await model.value?.ensure()
  })
}
</script>
<template>
  <dialog
    ref="dialog"
    class="table-import-dialog diagram-import-dialog"
    aria-label="智能导入图像"
    @cancel.prevent="task.hide"
    @paste="paste"
  >
    <div class="import-heading">
      <h2>{{ replacing ? '重新识别图像' : '智能导入图像' }}</h2>
      <button
        class="icon-button"
        aria-label="关闭图像导入"
        :title="task.working.value ? '收起窗口，后台继续识别' : '关闭窗口'"
        @click="task.hide"
      >
        ×
      </button>
    </div>
    <div
      class="import-image diagram-source"
      tabindex="0"
      aria-label="粘贴或拖入图像"
      @dragover.prevent
      @drop.prevent="choose([...($event.dataTransfer?.files ?? [])])"
    >
      <img v-if="task.data.value" :src="task.data.value" alt="待识别原图" />
      <span v-else>粘贴或拖入示意图、线稿图片</span>
      <label class="button"
        >{{ task.data.value ? '更换图片' : '选择图片'
        }}<input
          type="file"
          accept="image/png,image/jpeg"
          aria-label="图像文件"
          :disabled="task.busy.value"
          @change="fileChange"
      /></label>
    </div>
    <label class="import-field"
      >补充要求（可选）<input
        v-model="task.instruction.value"
        maxlength="2000"
        aria-label="图像补充要求"
        :disabled="task.working.value"
    /></label>
    <ImportModelConfig ref="model" :disabled="task.busy.value" @error="task.error.value = $event" />
    <p v-if="task.error.value" class="import-error" role="alert">{{ task.error.value }}</p>
    <div v-if="task.working.value" class="diagram-progress" role="status">
      <progress aria-label="图像识别进度" />正在重建图像… {{ task.elapsed.value }}
    </div>
    <div class="import-actions">
      <button class="button" @click="task.cancel">
        {{ task.working.value ? '取消识别' : '取消' }}
      </button>
      <button class="button primary" :disabled="task.busy.value || !task.data.value" @click="run">
        {{ task.working.value ? '识别中…' : replacing ? '重新识别' : '识别并插入' }}
      </button>
    </div>
  </dialog>
</template>
<style scoped>
.diagram-import-dialog {
  max-width: 680px;
}
.diagram-source {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}
.diagram-source img {
  max-width: 100%;
  max-height: 230px;
  object-fit: contain;
  background: white;
}
.diagram-progress {
  display: flex;
  gap: 12px;
  align-items: center;
  padding: 12px 0;
}
.import-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 16px;
}
</style>
