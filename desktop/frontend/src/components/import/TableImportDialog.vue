<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { errorText } from '../../services/bridge'
import type {
  Extraction,
  ImportPlacement,
  ImportPreview,
  ImportRequest,
} from '../../services/tableImportTypes'
import type { TableAsset } from '../../types'
import { useBatchTableImport } from '../../state/useBatchTableImport'
import ImportModelConfig from './ImportModelConfig.vue'
import ImportArrangement from './ImportArrangement.vue'
import ImportReviewEditor from './ImportReviewEditor.vue'
import './table-import.css'
const props = defineProps<{
  open: boolean
  tables: TableAsset[]
  anchor: () => ImportPlacement
  preview: (e: Extraction, p: ImportPlacement) => Promise<ImportPreview>
  submit: (request: ImportRequest, opened: number, cancelled: () => boolean) => Promise<unknown>
}>()
const emit = defineEmits<{ close: [] }>()
const dialog = ref<HTMLDialogElement>(),
  model = ref<InstanceType<typeof ImportModelConfig>>()
const batch = useBatchTableImport(props.preview)
const {
  sources,
  instruction,
  error,
  working,
  reading,
  processingCount,
  finishedCount,
  cancelled,
  reviewing,
  extraction,
  origins,
  plan,
  positions,
  planning,
  valid,
} = batch
const location = ref('auto'),
  selected = ref(0),
  saving = ref(false)
let editTimer: ReturnType<typeof setTimeout> | undefined
watch(
  () => props.open,
  async (open) => {
    if (!open) {
      clearTimeout(editTimer)
      if (working.value) void batch.cancel()
      else {
        cancelled.value = true
        batch.invalidate()
      }
      dialog.value?.close()
      return
    }
    cancelled.value = false
    await nextTick()
    dialog.value?.showModal()
    await model.value?.load()
  },
)
function back() {
  clearTimeout(editTimer)
  batch.invalidate()
  reviewing.value = false
}
function choose(e: Event) {
  const input = e.target as HTMLInputElement
  void batch.add([...(input.files ?? [])])
  input.value = ''
}
function paste(e: ClipboardEvent) {
  let files = [...(e.clipboardData?.files ?? [])].filter((f) => f.type.startsWith('image/'))
  if (!files.length)
    files = [...(e.clipboardData?.items ?? [])]
      .filter((i) => i.type.startsWith('image/'))
      .map((i) => i.getAsFile())
      .filter((f): f is File => !!f)
  if (saving.value || working.value) return
  if (files.length) {
    e.preventDefault()
    e.stopPropagation()
    void batch.add(files)
  }
}
async function run() {
  saving.value = true
  error.value = ''
  try {
    await model.value?.ensure()
    const a = props.anchor(),
      p =
        location.value === 'end'
          ? { page: a.page, y: 0, atEnd: true }
          : location.value === 'auto'
            ? { page: a.page, y: a.y }
            : { page: a.page, y: 0, afterId: location.value }
    selected.value = 0
    await batch.run(p)
  } catch (e) {
    error.value = errorText(e)
  } finally {
    saving.value = false
  }
}
function edited() {
  batch.invalidate()
  clearTimeout(editTimer)
  editTimer = setTimeout(() => void batch.arrange(positions.value), 350)
}
async function insert() {
  if (!valid.value || !plan.value || !positions.value) return
  saving.value = true
  error.value = ''
  try {
    await props.submit(
      {
        extraction: extraction.value,
        placement: { page: 1, y: 0, positions: positions.value },
        expectedRevision: plan.value.revision,
      },
      plan.value.opened,
      () => cancelled.value,
    )
    batch.reset()
    emit('close')
  } catch (e) {
    error.value = errorText(e)
    valid.value = false
  } finally {
    saving.value = false
  }
}
function close() {
  if (working.value) void batch.cancel()
  else if (!saving.value && !reading.value) {
    clearTimeout(editTimer)
    batch.invalidate()
    emit('close')
  }
}
onBeforeUnmount(() => clearTimeout(editTimer))
</script>
<template>
  <dialog
    ref="dialog"
    class="table-import-dialog"
    :class="{ 'has-preview': reviewing }"
    aria-label="智能导入表格"
    @cancel.prevent="close"
    @paste="paste"
  >
    <div class="import-heading">
      <h2>智能导入表格</h2>
      <button
        class="icon-button"
        aria-label="关闭表格导入"
        :disabled="working || saving || reading"
        @click="close"
      >
        ×
      </button>
    </div>
    <div
      v-if="!reviewing"
      class="import-image"
      tabindex="0"
      aria-label="粘贴或拖入表格图片"
      @dragover.prevent
      @drop.prevent="!saving && batch.add([...($event.dataTransfer?.files ?? [])])"
    >
      <span v-if="!sources.length">粘贴或拖入多张表格图片</span>
      <div v-for="(s, i) in sources" :key="s.id" class="import-source" :data-source="i">
        <img :src="s.data" :alt="s.name" /><span
          >{{ i + 1 }}. {{ s.name
          }}<small>{{
            s.processing ? '识别中' : s.result ? '已识别' : s.error || '待识别'
          }}</small></span
        ><button
          class="icon-button"
          :aria-label="`移除图片 ${i + 1}`"
          :disabled="working || saving || reading"
          @click="batch.remove(i)"
        >
          ×
        </button>
      </div>
      <label class="button"
        >添加图片<input
          type="file"
          multiple
          accept="image/png,image/jpeg"
          aria-label="表格图片文件"
          :disabled="working || saving || reading"
          @change="choose"
      /></label>
    </div>
    <template v-if="!reviewing">
      <label class="import-field"
        >插入位置<select v-model="location" :disabled="working || saving" aria-label="表格导入位置">
          <option value="auto">当前光标附近</option>
          <option value="end">当前页内容下方</option>
          <option
            v-for="(table, i) in tables.filter((t) => t.kind !== 'textbox')"
            :key="table.id"
            :value="table.id"
          >
            表格 {{ i + 1 }} 下方（第 {{ table.page + 1 }} 页）
          </option>
        </select></label
      >
      <label class="import-field"
        >补充要求（可选）<input
          v-model="instruction"
          maxlength="2000"
          :disabled="working || saving"
          placeholder="例如：测量格留空，保留合并单元格"
      /></label>
    </template>
    <div v-show="!reviewing">
      <ImportModelConfig ref="model" :disabled="working || saving" @error="error = $event" />
    </div>
    <div v-if="reviewing" class="import-preview-grid">
      <ImportArrangement
        v-if="plan && positions"
        :plan="plan"
        :positions="positions"
        :selected="selected"
        :disabled="saving || planning"
        @select="selected = $event"
        @change="batch.arrange($event)"
      />
      <div v-else class="import-empty-preview">调整识别内容后重新预览</div>
      <ImportReviewEditor
        :extraction="extraction"
        :selected="selected"
        :origins="origins"
        :sources="sources"
        :disabled="saving"
        @select="selected = $event"
        @change="edited"
      />
    </div>
    <p v-if="error" class="import-error" role="alert">{{ error }}</p>
    <p v-if="reviewing && plan?.warnings.length" class="import-warning">
      {{ [...new Set(plan.warnings)].join('；') }}
    </p>
    <div class="import-footer">
      <span v-if="working" role="status"
        >正在识别 {{ processingCount }} 张 · 已完成 {{ finishedCount }} /
        {{ sources.length }} 张</span
      >
      <span v-else-if="planning" role="status">正在排版…</span>
      <button v-if="working" class="button" @click="batch.cancel">取消识别</button>
      <template v-else-if="reviewing">
        <button class="button" :disabled="saving" @click="back">返回图片</button>
        <button class="button" :disabled="saving || planning" @click="batch.arrange()">
          自动排列
        </button>
        <button class="button" :disabled="saving || planning" @click="batch.arrange(positions)">
          重新预览
        </button>
        <button class="button primary" :disabled="saving || planning || !valid" @click="insert">
          全部插入
        </button>
      </template>
      <button
        v-else
        class="button primary"
        :disabled="!sources.length || saving || reading"
        title="将图片并行发送到所选模型接口，每张识别一次"
        @click="run"
      >
        {{ sources.some((s) => s.error) ? '重试并预览' : '识别并预览' }}
      </button>
    </div>
  </dialog>
</template>
