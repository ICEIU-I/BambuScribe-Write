<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
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
  void task.add(files)
}
function fileChange(e: Event, index?: number) {
  const input = e.target as HTMLInputElement
  void task.add([...(input.files ?? [])], index)
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
      <span v-if="!task.sources.value.length">粘贴或拖入多张示意图、线稿图片</span>
      <div
        v-for="(source, index) in task.sources.value"
        :key="source.id"
        class="import-source diagram-source-item"
        :class="{ single: task.sources.value.length === 1 }"
        :data-diagram-source="index"
      >
        <img
          :src="source.data"
          :alt="task.sources.value.length === 1 ? '待识别原图' : `待识别原图 ${index + 1}`"
        />
        <span>
          {{ index + 1 }}. {{ source.name }}
          <small :class="{ 'import-error': source.error }">{{
            source.processing ? '识别中…' : source.result ? '已识别' : source.error || '待识别'
          }}</small>
        </span>
        <label class="button"
          >更换图片<input
            type="file"
            accept="image/png,image/jpeg"
            :aria-label="`更换图像 ${index + 1}`"
            :disabled="task.busy.value"
            @change="fileChange($event, index)"
        /></label>
        <button
          class="icon-button"
          :aria-label="`移除图像 ${index + 1}`"
          :disabled="task.busy.value"
          @click="task.remove(index)"
        >
          ×
        </button>
      </div>
      <label class="button"
        >{{ task.sources.value.length ? '添加图片' : '选择图片'
        }}<input
          type="file"
          multiple
          accept="image/png,image/jpeg"
          aria-label="图像文件"
          :disabled="task.busy.value"
          @change="fileChange($event)"
      /></label>
    </div>
    <p class="import-warning">最多同时识别 3 张 · 每批最多 10 张 · 文字自动放大 1.5–2 倍</p>
    <label class="import-field"
      >补充要求（可选）<input
        v-model="task.instruction.value"
        maxlength="2000"
        aria-label="图像补充要求"
        :disabled="task.busy.value"
    /></label>
    <ImportModelConfig ref="model" :disabled="task.busy.value" @error="task.error.value = $event" />
    <p v-if="task.error.value" class="import-error" role="alert">{{ task.error.value }}</p>
    <div v-if="task.working.value" class="diagram-progress" role="status">
      <progress aria-label="图像识别进度" />
      <span>
        {{ task.processingCount.value ? '正在重建图像…' : '正在准备导入…' }}
        {{ task.elapsed.value }}
        <small>
          正在识别 {{ task.processingCount.value }} 张 · 已完成 {{ task.finishedCount.value }} /
          {{ task.sources.value.length }} 张
        </small>
      </span>
    </div>
    <div class="import-actions">
      <button class="button" @click="task.cancel">
        {{ task.working.value ? '取消识别' : '取消' }}
      </button>
      <button
        class="button primary"
        :disabled="task.busy.value || !task.sources.value.length"
        @click="run"
      >
        {{
          task.working.value
            ? '识别中…'
            : task.failed.value && task.sources.value.length > 1
              ? '重试失败图片'
              : replacing
                ? '重新识别'
                : '识别并插入'
        }}
      </button>
    </div>
  </dialog>
</template>
<style scoped>
.diagram-import-dialog {
  max-width: 680px;
}
.diagram-source {
  max-height: 360px;
}
.diagram-source-item.single {
  flex-wrap: wrap;
}
.diagram-source-item.single img {
  width: 100%;
  height: auto;
  max-height: 230px;
}
.diagram-source-item .button {
  padding: 6px 10px;
  font-size: 12px;
}
.diagram-source input[type='file'] {
  display: none;
}
.diagram-progress {
  display: flex;
  gap: 12px;
  align-items: center;
  padding: 12px 0;
}
.diagram-progress small {
  display: block;
  margin-top: 4px;
  color: var(--text-soft);
}
.import-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 16px;
}
</style>
