<script setup lang="ts">
import { computed } from 'vue'
import type { PrinterDevice } from '../../types'
const props = defineProps<{
  device: PrinterDevice
  selected: boolean
  action?: 'connect' | 'save' | 'delete'
  error?: string
}>()
const emit = defineEmits<{ edit: [string]; remove: [string]; select: [string] }>()
const connecting = computed(
  () => props.action === 'connect' || props.device.state.phase === 'connecting',
)
const uploading = computed(() => props.device.state.phase === 'uploading')
const locked = computed(() => connecting.value || uploading.value || props.action === 'delete')
const failed = computed(
  () => !!props.error || (!props.device.state.connected && !!props.device.state.message),
)
function edit(event: MouseEvent) {
  // WebKit does not focus clicked buttons automatically; dialog close returns here.
  ;(event.currentTarget as HTMLButtonElement).focus()
  emit('edit', props.device.id)
}
</script>
<template>
  <article class="printer-instance" :class="{ selected }" :data-device-id="device.id">
    <input
      class="instance-target"
      type="radio"
      name="upload-target"
      :checked="selected"
      :aria-label="`使用 ${device.name} 上传`"
      title="上传目标"
      @change="emit('select', device.id)"
    />
    <button class="instance-open" :aria-label="`编辑 ${device.name}`" @click="edit">
      <span class="instance-name">{{ device.name }}</span>
      <span
        class="badge"
        :class="
          connecting ? 'warn' : device.state.connected ? 'info' : failed ? 'error' : 'neutral'
        "
      >
        {{
          connecting
            ? '连接中'
            : uploading
              ? '上传中'
              : device.state.connected
                ? '已连接'
                : failed
                  ? '连接失败'
                  : '未连接'
        }}
      </span>
    </button>
    <button
      class="button secondary instance-symbol"
      :aria-label="`删除 ${device.name}`"
      title="删除实例"
      :disabled="locked"
      @click="emit('remove', device.id)"
    >
      −
    </button>
  </article>
</template>

<style scoped>
.printer-instance {
  display: flex;
  align-items: center;
  gap: 10px;
  border: var(--hair) solid var(--border-soft);
  border-radius: 8px;
  padding: 8px 10px;
  margin: 10px 0;
  background: var(--surface-soft);
}
.printer-instance.selected {
  border-color: var(--border);
}
.instance-target {
  flex: 0 0 15px;
  width: 15px;
  height: 15px;
  margin: 0;
  accent-color: var(--accent);
}
.instance-open {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  border: 0;
  border-radius: 5px;
  padding: 7px 0;
  background: transparent;
  color: var(--text);
  text-align: left;
  cursor: pointer;
}
.instance-open:hover .instance-name {
  text-decoration: underline;
}
.instance-name {
  overflow-wrap: anywhere;
  font-size: 13px;
  font-weight: 500;
}
.instance-open .badge {
  flex-shrink: 0;
}
</style>
