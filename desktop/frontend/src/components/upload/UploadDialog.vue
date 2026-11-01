<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import type { Document, PrinterState, WritingSettings, PrinterDevice } from '../../types'
import { defaultUploadPrefix, validUploadPrefix, uploadPageSuffix } from '../../state/uploadNaming'
import { outputAvailability } from '../../state/outputAvailability'
import { writingOriginRange } from '../../state/writingOrigin'
import { t } from '../../i18n'
import AppIcon from '../AppIcon.vue'

const props = defineProps<{
  open: boolean
  text: string
  document: Document | null
  writing: WritingSettings
  index: number
  ready: boolean
  loading: boolean
  dirty: boolean
  busy: boolean
  state: PrinterState
  notice: string
  submit: (prefix: string, index: number | null) => Promise<string | undefined>
  cancel: () => Promise<void>
  devices: PrinterDevice[]
  deviceId: string
}>()
const emit = defineEmits<{ close: []; device: [string] }>()
const dialog = ref<HTMLDialogElement>(),
  input = ref<HTMLInputElement>()
const prefix = ref(''),
  selected = ref(0),
  submitted = ref(false),
  submitting = ref(false)
const submittedIndex = ref<number | null>(null)
const uploading = computed(() => props.state.phase === 'uploading')
const complete = computed(
  () =>
    submitted.value &&
    !submitting.value &&
    !uploading.value &&
    props.state.totalPages > 0 &&
    props.state.uploadedPages === props.state.totalPages,
)
const locked = computed(() => submitting.value || props.busy || uploading.value)
const numbers = computed(
  () =>
    props.document?.pages
      .filter((p) => p.strokes.length || p.missing.length || p.issues?.length)
      .map((p) => p.number) || [],
)
const suffix = computed(() =>
  uploadPageSuffix(
    (uploading.value || complete.value) && submittedIndex.value !== null
      ? [submittedIndex.value + 1]
      : numbers.value,
  ),
)
const specificName = computed(
  () => `${prefix.value.trim()}${uploadPageSuffix([selected.value + 1])}`,
)
const invalidName = computed(() => !validUploadPrefix(prefix.value))
const reasons = computed(() =>
  outputAvailability({
    document: props.document,
    index: selected.value,
    ready: props.ready,
    loading: props.loading,
    dirty: props.dirty,
    busy: props.busy,
  }),
)
const progress = computed(() => Math.max(0, Math.min(100, props.state.progress || 0)))
const originRange = computed(() =>
  props.ready && !props.loading && !props.dirty
    ? writingOriginRange(props.document, props.writing)
    : null,
)

watch(
  () => props.open,
  async (open) => {
    await nextTick()
    if (open) {
      prefix.value = defaultUploadPrefix(props.text, t('app.untitled'))
      selected.value = props.index
      submitted.value = false
      submittedIndex.value = null
      dialog.value?.showModal()
      await nextTick()
      input.value?.focus()
      input.value?.select()
    } else dialog.value?.close()
  },
  { immediate: true },
)

async function upload(index: number | null) {
  if (locked.value || invalidName.value) return
  submitted.value = false
  submittedIndex.value = index
  submitting.value = true
  try {
    const id = await props.submit(prefix.value.trim(), index)
    submitted.value = !!id
  } finally {
    submitting.value = false
  }
}
function outside(event: MouseEvent) {
  if (event.target !== dialog.value) return
  const bounds = dialog.value.getBoundingClientRect()
  if (
    event.clientX < bounds.left ||
    event.clientX > bounds.right ||
    event.clientY < bounds.top ||
    event.clientY > bounds.bottom
  )
    emit('close')
}
</script>

<template>
  <Teleport to="body">
    <dialog
      ref="dialog"
      class="upload-dialog"
      :aria-label="t('upload.title')"
      @cancel.prevent="emit('close')"
      @click="outside"
    >
      <div class="upload-heading">
        <strong>{{ t('upload.title') }}</strong>
        <button class="icon-button" :aria-label="t('upload.close')" @click="emit('close')">
          <AppIcon name="close" :size="16" />
        </button>
      </div>
      <label class="form-label">
        <span>打印机</span>
        <select
          :value="deviceId"
          :disabled="locked || complete"
          aria-label="上传打印机"
          @change="emit('device', ($event.target as HTMLSelectElement).value)"
        >
          <option v-if="!devices.length" value="">请在设备台新增打印机</option>
          <option v-for="device in devices" :key="device.id" :value="device.id">
            {{ device.name }} · {{ device.ip || '未填写 IP' }} ·
            {{ device.state.connected ? '已连接' : '未连接' }}
          </option>
        </select>
      </label>
      <label class="form-label">
        <span>{{ t('upload.filename') }}</span>
        <div class="upload-filename">
          <input
            ref="input"
            v-model="prefix"
            :aria-label="t('upload.prefix')"
            :disabled="locked || complete"
            maxlength="128"
            autocomplete="off"
          />
          <span class="upload-suffix">{{ suffix }}</span>
        </div>
      </label>
      <p v-if="invalidName" class="upload-error" role="alert">{{ t('upload.invalidName') }}</p>
      <p v-if="originRange" class="hint upload-origin">
        {{
          t('upload.originRange', {
            x: originRange.x.join('–'),
            y: originRange.y.join('–'),
          })
        }}
      </p>
      <template v-if="!uploading && !complete">
        <label class="form-label upload-page-field">
          <span>{{ t('upload.page') }}</span>
          <select v-model="selected" :disabled="locked" :aria-label="t('upload.page')">
            <option v-for="(page, i) in document?.pages || []" :key="page.number" :value="i">
              {{ t('upload.pageNumber', { page: page.number }) }}
            </option>
          </select>
          <span class="upload-specific-name">{{ specificName }}</span>
        </label>
        <p v-if="reasons.all && !busy" class="upload-error">{{ reasons.all }}</p>
        <p v-if="reasons.current && reasons.current !== reasons.all && !busy" class="upload-error">
          {{ reasons.current }}
        </p>
      </template>
      <div v-if="uploading || complete" class="upload-progress" aria-live="polite">
        <div class="upload-progress-label">
          <span>{{
            complete
              ? t('upload.done')
              : t('upload.currentPage', {
                  page: state.page,
                  ordinal: state.pageOrdinal,
                  total: state.totalPages,
                })
          }}</span
          ><strong>{{ progress }}%</strong>
        </div>
        <progress :value="progress" max="100" :aria-label="t('job.uploadProgress')"></progress>
        <p v-if="uploading" class="hint">
          {{ t('upload.uploadedCount', { count: state.uploadedPages, total: state.totalPages }) }}
        </p>
      </div>
      <p v-if="submitting && !uploading" class="hint" role="status">{{ t('upload.preparing') }}</p>
      <p v-if="notice && !complete" class="upload-error" role="alert">{{ notice }}</p>
      <p v-if="state.phase === 'failed'" class="upload-error" role="alert">{{ state.message }}</p>
      <div class="button-row upload-actions">
        <button
          v-if="complete"
          class="button primary"
          :title="t('upload.controlOnA1')"
          @click="emit('close')"
        >
          {{ t('upload.finish') }}
        </button>
        <button v-else-if="uploading" class="button secondary" @click="cancel()">
          {{ t('job.cancelUpload') }}
        </button>
        <template v-else>
          <button
            class="button primary"
            :disabled="locked || invalidName || !!reasons.all"
            :title="reasons.all"
            @click="upload(null)"
          >
            {{ t('upload.all') }}
          </button>
          <button
            class="button secondary"
            :disabled="locked || invalidName || !!reasons.current"
            :title="reasons.current"
            @click="upload(selected)"
          >
            {{ t('upload.specific') }}
          </button>
        </template>
      </div>
    </dialog>
  </Teleport>
</template>

<style scoped>
.upload-dialog {
  width: 440px;
  max-width: calc(100vw - 32px);
  padding: 20px;
  color: var(--text);
  background: var(--elevated);
  border: var(--edge) solid var(--border);
  border-radius: 14px;
  box-shadow: var(--shadow);
}
.upload-dialog::backdrop {
  background: rgb(0 0 0 / 30%);
}
.upload-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 22px;
  gap: 12px;
}
.upload-heading strong {
  font-size: 16px;
}
.upload-heading .icon-button {
  border: 0;
  box-shadow: none;
}
.upload-filename {
  display: flex;
  align-items: center;
  border: var(--hair) solid var(--border);
  border-radius: 7px;
  overflow: hidden;
  background: var(--surface-soft);
}
.upload-filename input {
  flex: 0 1 auto;
  width: 100px;
  max-width: 50%;
  border: 0;
}
.upload-suffix {
  padding: 0 12px 0 0;
  color: var(--text-soft);
  white-space: nowrap;
  font-family: ui-monospace, monospace;
}
.upload-page-field {
  margin-top: 18px;
}
.upload-specific-name {
  color: var(--text-soft);
  font-size: 11px;
  overflow-wrap: anywhere;
}
.upload-error {
  color: var(--danger);
  font-size: 11px;
  margin: 10px 0;
  overflow-wrap: anywhere;
}
.upload-actions {
  justify-content: flex-end;
}
.upload-progress-label {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
  font-size: 12px;
}
.upload-progress progress {
  display: block;
  width: 100%;
  height: 14px;
  accent-color: var(--accent);
}
.upload-progress .hint {
  margin-bottom: 0;
}
</style>
