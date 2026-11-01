<script setup lang="ts">
import '../../styles/printer-instances.css'
import { computed, ref } from 'vue'
import type { useWorkspace } from '../../state/useWorkspace'
import AppIcon from '../AppIcon.vue'
import ConnectionCard from './ConnectionCard.vue'
import PrinterDeviceDialog from './PrinterDeviceDialog.vue'
import WritingSettingsCard from './WritingSettingsCard.vue'
const props = defineProps<{ workspace: ReturnType<typeof useWorkspace> }>()
const w = props.workspace
const connected = computed(() => w.devices.value.filter((device) => device.state.connected).length)
const editingDeviceId = ref('')
const editingDevice = computed(() =>
  w.devices.value.find((device) => device.id === editingDeviceId.value),
)
async function addDevice(event: MouseEvent) {
  ;(event.currentTarget as HTMLButtonElement).focus()
  const id = await w.addDevice()
  if (id) editingDeviceId.value = id
}
</script>
<template>
  <main class="device-page">
    <div class="device-page-heading">
      <div>
        <h1>设备台</h1>
      </div>
      <div class="device-summary">
        <span class="pixel-badge">{{ connected }} / {{ w.devices.value.length }} 已连接</span>
      </div>
    </div>
    <div class="device-settings-grid">
      <section class="connection-only device-card">
        <div class="card-heading">
          <h2 title="打印机需开启 LAN 模式及固件支持的开发者模式">
            <AppIcon name="wifi" />打印机连接
          </h2>
          <button
            class="button secondary instance-symbol"
            aria-label="新增实例"
            title="新增实例"
            @click="addDevice"
          >
            +
          </button>
        </div>
        <ConnectionCard
          v-for="device in w.devices.value"
          :key="device.id"
          :device="device"
          :selected="device.id === w.selectedDeviceId.value"
          :action="w.deviceActions[device.id]"
          :error="w.deviceErrors[device.id]"
          @edit="editingDeviceId = $event"
          @remove="w.deleteDevice"
          @select="w.selectDevice"
        />
      </section>
      <WritingSettingsCard
        :settings="w.preferences.writing"
        @change="w.preferences.writing = $event"
      />
    </div>
    <PrinterDeviceDialog
      :device="editingDevice"
      :action="w.deviceActions[editingDeviceId]"
      :error="w.deviceErrors[editingDeviceId]"
      @close="editingDeviceId = ''"
      @change="w.editDevice"
      @connect="w.connectDevice($event)"
      @disconnect="w.disconnectDevice"
      @cancel="w.cancelUpload($event)"
    />
  </main>
</template>
