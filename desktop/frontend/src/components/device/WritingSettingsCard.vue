<script setup lang="ts">
import type { WritingSettings } from '../../types'
import NumberField from '../NumberField.vue'
import AppIcon from '../AppIcon.vue'
const props = defineProps<{ settings: WritingSettings }>()
const emit = defineEmits<{ change: [WritingSettings] }>()
function update(key: keyof WritingSettings, value: number | null) {
  if (value !== null) emit('change', { ...props.settings, [key]: value })
}
</script>
<template>
  <section class="device-card writing-settings-card">
    <div class="card-heading">
      <h2><AppIcon name="settings" />书写参数</h2>
      <span class="badge">自动保存</span>
    </div>
    <div class="settings-grid">
      <NumberField
        label="抬笔位置"
        title="归零后的绝对 Z 高度，请按笔架校准"
        :model-value="settings.penUp"
        :nullable="true"
        :min="0"
        :max="256"
        :step="0.1"
        @update:model-value="update('penUp', $event)"
      /><NumberField
        label="X 轴偏移"
        title="正值向右，负值向左"
        :model-value="settings.offsetX"
        :min="-256"
        :max="256"
        :step="0.1"
        @update:model-value="update('offsetX', $event)"
      /><NumberField
        label="落笔位置"
        title="归零后的绝对 Z 高度，请按笔架校准"
        :model-value="settings.penDown"
        :nullable="true"
        :min="0"
        :max="256"
        :step="0.1"
        @update:model-value="update('penDown', $event)"
      /><NumberField
        label="Y 轴偏移"
        title="正值向下，负值向上"
        :model-value="settings.offsetY"
        :min="-256"
        :max="256"
        :step="0.1"
        @update:model-value="update('offsetY', $event)"
      />
      <label class="form-label"
        >原点模式<select
          :value="settings.originMode"
          :title="
            settings.originMode === 'auto'
              ? '以笔尖当前位置作为书写区域左上角，请先定位笔尖'
              : '沿用打印机当前 XY 坐标'
          "
          @change="
            emit('change', {
              ...settings,
              originMode: ($event.target as HTMLSelectElement)
                .value as WritingSettings['originMode'],
            })
          "
        >
          <option value="manual">None（不设置 XY 原点）</option>
          <option value="auto">自动 XY 原点</option>
        </select></label
      >
      <label class="form-label"
        >提示音<select
          :value="String(settings.sound)"
          title="书写完成时播放提示音"
          @change="
            emit('change', {
              ...settings,
              sound: ($event.target as HTMLSelectElement).value === 'true',
            })
          "
        >
          <option value="false">不启用</option>
          <option value="true">启用</option>
        </select></label
      >
    </div>
  </section>
</template>
