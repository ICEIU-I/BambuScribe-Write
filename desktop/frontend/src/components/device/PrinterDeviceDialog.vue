<script setup lang="ts">
import { computed, nextTick, reactive, ref, watch } from 'vue'
import type { PrinterDevice } from '../../types'
import AppIcon from '../AppIcon.vue'

const props = defineProps<{
  device?: PrinterDevice
  action?: 'connect' | 'save' | 'delete'
  error?: string
}>()
const emit = defineEmits<{
  close: []
  change: [PrinterDevice]
  connect: [PrinterDevice]
  disconnect: [string]
  cancel: [string]
}>()
const dialog = ref<HTMLDialogElement>(),
  nameInput = ref<HTMLInputElement>()
const form = reactive({ name: '', ip: '', accessCode: '' })
const connecting = computed(
  () => props.action === 'connect' || props.device?.state.phase === 'connecting',
)
const uploading = computed(() => props.device?.state.phase === 'uploading')
const locked = computed(() => connecting.value || uploading.value || props.action === 'delete')
const failed = computed(
  () => !!props.error || (!props.device?.state.connected && !!props.device?.state.message),
)
const value = () => ({ ...props.device!, ...form })

watch(
  () => props.device?.id,
  async (id) => {
    if (id && props.device) {
      const { name, ip, accessCode } = props.device
      Object.assign(form, { name, ip, accessCode })
    }
    await nextTick()
    if (id) {
      dialog.value?.showModal()
      nameInput.value?.focus()
      nameInput.value?.select()
    } else dialog.value?.close()
  },
  { immediate: true },
)
for (const field of ['name', 'ip', 'accessCode'] as const) {
  watch(
    () => props.device?.[field],
    (value) => {
      if (value !== undefined) form[field] = value
    },
  )
}
function close() {
  // Flush the focused field's change before closing with Escape or the backdrop.
  const active = document.activeElement
  if (active instanceof HTMLElement && dialog.value?.contains(active)) active.blur()
  emit('close')
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
    close()
}
</script>
<template>
  <Teleport to="body">
    <dialog
      ref="dialog"
      class="printer-device-dialog"
      aria-label="编辑打印机实例"
      @cancel.prevent="close"
      @click="outside"
    >
      <template v-if="device">
        <div class="instance-dialog-heading">
          <strong>编辑实例</strong>
          <button class="icon-button" aria-label="关闭实例编辑" @click="close">
            <AppIcon name="close" :size="16" />
          </button>
        </div>
        <label class="form-label"
          >名称<input
            ref="nameInput"
            v-model="form.name"
            :disabled="locked"
            autocomplete="off"
            @change="emit('change', value())"
        /></label>
        <label class="form-label"
          >IP 地址<input
            v-model.trim="form.ip"
            :disabled="locked"
            placeholder="192.168.1.100"
            autocomplete="off"
            spellcheck="false"
            @change="emit('change', value())"
        /></label>
        <label class="form-label"
          >访问码<input
            v-model="form.accessCode"
            type="text"
            :disabled="locked"
            placeholder="LAN 模式访问码"
            autocomplete="off"
            spellcheck="false"
            @change="emit('change', value())"
        /></label>
        <p
          v-if="error || device.state.message"
          class="instance-message"
          :class="{ error: failed }"
          :role="failed ? 'alert' : 'status'"
        >
          {{ error || device.state.message }}
        </p>
        <div v-if="uploading" class="instance-upload">
          <span
            >第 {{ device.state.page }} 页 · {{ device.state.uploadedPages }} /
            {{ device.state.totalPages }} 页</span
          >
          <progress
            :value="device.state.progress"
            max="100"
            :aria-label="`${device.name} 上传进度`"
          ></progress>
        </div>
        <button v-if="uploading" class="button secondary full" @click="emit('cancel', device.id)">
          取消上传
        </button>
        <button
          v-else-if="!device.state.connected"
          class="button primary full"
          :disabled="locked || !form.ip || !form.accessCode"
          @click="emit('connect', value())"
        >
          {{ connecting ? '连接中…' : '连接打印机' }}
        </button>
        <button
          v-else
          class="button secondary full"
          :disabled="locked"
          @click="emit('disconnect', device.id)"
        >
          断开连接
        </button>
      </template>
    </dialog>
  </Teleport>
</template>

<style scoped>
.printer-device-dialog {
  width: 400px;
  max-width: calc(100vw - 32px);
  max-height: calc(100vh - 32px);
  overflow-y: auto;
  padding: 20px;
  color: var(--text);
  background: var(--elevated);
  border: var(--hair) solid var(--border);
  border-radius: 14px;
  box-shadow: var(--shadow);
}
.printer-device-dialog::backdrop {
  background: rgb(0 0 0 / 30%);
}
.instance-dialog-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 22px;
  gap: 12px;
}
.instance-dialog-heading strong {
  font-size: 16px;
}
.instance-message {
  margin: 10px 0;
  font-size: 12px;
  overflow-wrap: anywhere;
}
.instance-message.error {
  color: var(--danger);
}
.printer-device-dialog .full {
  margin-top: 12px;
}
.instance-upload {
  display: grid;
  gap: 8px;
  margin-top: 10px;
  font-size: 12px;
}
.instance-upload progress {
  width: 100%;
  accent-color: var(--accent);
}
</style>
