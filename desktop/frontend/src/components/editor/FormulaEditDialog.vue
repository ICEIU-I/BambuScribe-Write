<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import type { Document, LayoutSettings } from '../../types'
import { errorText, native } from '../../services/bridge'
import { formulaFieldText, parseFormulaFields, type FormulaField } from '../../state/formulaFields'
import FormulaFields from './FormulaFields.vue'
const props = defineProps<{ source: string | null; settings: LayoutSettings; disabled: boolean }>()
const emit = defineEmits<{ close: []; save: [string] }>()
const dialog = ref<HTMLDialogElement>(),
  nodes = ref<FormulaField[]>([]),
  preview = ref<Document>(),
  error = ref(''),
  busy = ref(false)
let generation = 0,
  timer: ReturnType<typeof setTimeout> | undefined
function changed() {
  generation++
  preview.value = undefined
  clearTimeout(timer)
  timer = setTimeout(() => void render(), 200)
}
async function render() {
  const ticket = ++generation
  busy.value = true
  error.value = ''
  try {
    const result = await native().Preview(`\\(${formulaFieldText(nodes.value)}\\)`, props.settings)
    if (ticket === generation) preview.value = result
  } catch (e) {
    if (ticket === generation) error.value = errorText(e)
  } finally {
    if (ticket === generation) busy.value = false
  }
}
watch(
  () => props.source,
  async (source) => {
    generation++
    clearTimeout(timer)
    preview.value = undefined
    error.value = ''
    if (source === null) {
      dialog.value?.close()
      return
    }
    try {
      nodes.value = parseFormulaFields(source)
    } catch (e) {
      error.value = errorText(e)
      nodes.value = []
    }
    await nextTick()
    dialog.value?.showModal()
    if (nodes.value.length) void render()
  },
)
onBeforeUnmount(() => {
  generation++
  clearTimeout(timer)
})
</script>
<template>
  <dialog
    ref="dialog"
    class="formula-edit-dialog"
    aria-label="编辑公式"
    @cancel.prevent="emit('close')"
  >
    <div class="calculation-heading">
      <h2>编辑公式</h2>
      <button class="icon-button" aria-label="关闭公式编辑" @click="emit('close')">×</button>
    </div>
    <div class="formula-edit-fields">
      <FormulaFields :nodes="nodes" label="公式文字" :disabled="disabled" @change="changed" />
    </div>
    <svg
      v-for="page in preview?.pages ?? []"
      :key="page.number"
      class="formula-edit-preview"
      :viewBox="`0 0 ${settings.paperWidth} ${Math.max(25, ...(page.formulas ?? []).map((f) => f.y + f.height + 5))}`"
      role="img"
      aria-label="公式预览"
    >
      <path
        v-for="(stroke, i) in page.strokes"
        :key="i"
        :d="stroke.map((p, j) => `${j ? 'L' : 'M'}${p.x},${p.y}`).join(' ')"
        fill="none"
        stroke="currentColor"
        stroke-width="0.26"
        stroke-linecap="round"
      />
    </svg>
    <p v-if="error" role="alert" class="import-error">{{ error }}</p>
    <div class="calculation-footer">
      <button class="button" @click="emit('close')">取消</button
      ><button
        class="button primary"
        :disabled="disabled || busy || !preview"
        @click="emit('save', formulaFieldText(nodes))"
      >
        保存公式
      </button>
    </div>
  </dialog>
</template>
<style scoped>
.formula-edit-dialog {
  width: min(900px, calc(100vw - 48px));
  max-height: 80vh;
  padding: 24px;
  border: 1px solid var(--border);
  border-radius: 14px;
  background: var(--surface);
  color: var(--text);
  overflow: auto;
}
.formula-edit-dialog::backdrop {
  background: rgb(0 0 0 / 30%);
}
.formula-edit-fields {
  padding: 14px 0 22px;
  overflow: auto;
}
.formula-edit-preview {
  display: block;
  width: 100%;
  margin-top: 20px;
}
</style>
