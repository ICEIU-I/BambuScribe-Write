<script setup lang="ts">
import { computed } from 'vue'
import type { LayoutSettings } from '../../types'
import { defaultLayout } from '../../state/defaults'
import { fontMillimeters, fontPercent, updateLayoutSettings } from '../../state/fontSizing'
import NumberField from '../NumberField.vue'
import HandwritingVariation from './HandwritingVariation.vue'
import AppIcon from '../AppIcon.vue'
const props = defineProps<{ settings: LayoutSettings }>()
const emit = defineEmits<{ change: [LayoutSettings]; close: [] }>()
const margins: { key: keyof LayoutSettings; label: string }[] = [
  { key: 'marginTop', label: '上边距' },
  { key: 'marginBottom', label: '下边距' },
  { key: 'marginLeft', label: '左边距' },
  { key: 'marginRight', label: '右边距' },
]
const lineHeight = computed(() => {
  const height =
    (((props.settings.paperHeight - props.settings.marginTop - props.settings.marginBottom) /
      props.settings.lineCount) *
      props.settings.lineSpacing) /
    100
  return Number.isFinite(height) ? height.toFixed(2) : '—'
})
const sizePercent = computed(() =>
  Number(fontPercent(props.settings.fontSize, props.settings.lineHeight).toFixed(2)),
)
const rowCapacity = computed(() =>
  Math.max(1, Math.floor((props.settings.lineCount * 100) / props.settings.lineSpacing + 1e-8)),
)
const sizeMinimum = computed(() => Math.ceil(10000 / props.settings.lineHeight) / 100)
const sizeMaximum = computed(
  () => Math.floor(Math.min(200, 4000 / props.settings.lineHeight) * 100) / 100,
)
function update(patch: Partial<LayoutSettings>) {
  emit('change', updateLayoutSettings(props.settings, patch))
}
</script>
<template>
  <div class="page-settings-content">
    <div class="settings-columns">
      <section class="settings-section" aria-labelledby="type-settings-title">
        <h3 id="type-settings-title">文字</h3>
        <label class="font-current">
          <span>字体</span>
          <select
            aria-label="字体"
            :value="settings.fontId || 'handwriting'"
            @change="
              update({
                fontId: ($event.target as HTMLSelectElement).value as LayoutSettings['fontId'],
              })
            "
          >
            <option value="handwriting">教案专用</option>
            <option value="dc">DC体</option>
          </select>
        </label>
        <div class="fields-grid">
          <NumberField
            label="字体大小"
            :title="`当前字框 ${settings.fontSize.toFixed(2)} mm`"
            unit="%"
            :step="1"
            :model-value="sizePercent"
            :min="sizeMinimum"
            :max="sizeMaximum"
            @update:model-value="
              $event !== null && update({ fontSize: fontMillimeters($event, settings.lineHeight) })
            "
          />
          <NumberField
            label="分式字号比例"
            unit="%"
            :step="1"
            :model-value="settings.fractionScale || 85"
            :min="50"
            :max="100"
            @update:model-value="$event !== null && update({ fractionScale: $event })"
          />
          <NumberField
            label="字距"
            unit="%"
            :step="1"
            :model-value="settings.letterSpacing"
            :min="50"
            :max="200"
            @update:model-value="$event !== null && update({ letterSpacing: $event })"
          />
          <NumberField
            label="行距"
            :title="`当前行距 ${lineHeight} mm，最多 ${rowCapacity} 行`"
            unit="%"
            :step="1"
            :model-value="settings.lineSpacing"
            :min="50"
            :max="200"
            @update:model-value="$event !== null && update({ lineSpacing: $event })"
          />
          <NumberField
            label="额外字距"
            :model-value="settings.letterGap"
            :min="0"
            :max="40"
            @update:model-value="$event !== null && update({ letterGap: $event })"
          />
          <NumberField
            label="基准行数"
            unit="行"
            :step="1"
            :model-value="settings.lineCount"
            :min="1"
            :max="100"
            @update:model-value="$event !== null && update({ lineCount: $event })"
          />
          <NumberField
            label="段距"
            :model-value="settings.paragraphGap"
            :min="0"
            :max="100"
            @update:model-value="$event !== null && update({ paragraphGap: $event })"
          />
          <NumberField
            label="首行缩进"
            unit="字"
            :step="1"
            :model-value="settings.indent"
            :min="0"
            :max="10"
            @update:model-value="$event !== null && update({ indent: $event })"
          />
          <div class="alignment-field">
            <span>对齐</span>
            <div class="alignment-controls">
              <button
                v-for="(label, align) in { left: '左对齐', center: '居中', right: '右对齐' }"
                :key="align"
                class="icon-button"
                :aria-label="label"
                :aria-pressed="settings.align === align"
                :title="label"
                @click="update({ align })"
              >
                <AppIcon :name="`align-${align}`" :size="17" />
              </button>
            </div>
          </div>
        </div>
        <HandwritingVariation :settings="settings" @change="update" />
        <div class="row-mode" role="group" aria-label="行数模式">
          <label class="checkbox-row"
            ><input
              type="radio"
              name="line-mode"
              :checked="settings.lineMode === 'fixed'"
              @change="update({ lineMode: 'fixed' })"
            />固定行数</label
          >
          <label class="checkbox-row"
            ><input
              type="radio"
              name="line-mode"
              title="每页基准行数在设定值上下 1 行内变化"
              :checked="settings.lineMode === 'random'"
              @change="update({ lineMode: 'random' })"
            />随机行数</label
          >
        </div>
      </section>
      <section class="settings-section" aria-labelledby="paper-settings-title">
        <h3 id="paper-settings-title">纸张</h3>
        <div class="paper-presets" aria-label="纸张预设">
          <button
            class="button secondary compact"
            @click="update({ paperWidth: 163, paperHeight: 240 })"
          >
            163 × 240</button
          ><button
            class="button secondary compact"
            @click="update({ paperWidth: 210, paperHeight: 297 })"
          >
            A4</button
          ><button
            class="button secondary compact"
            @click="update({ paperWidth: 176, paperHeight: 250 })"
          >
            B5
          </button>
        </div>
        <div class="fields-grid">
          <NumberField
            label="纸张宽度"
            :model-value="settings.paperWidth"
            :min="20"
            :max="1000"
            @update:model-value="$event !== null && update({ paperWidth: $event })"
          />
          <NumberField
            label="纸张高度"
            :model-value="settings.paperHeight"
            :min="20"
            :max="1000"
            @update:model-value="$event !== null && update({ paperHeight: $event })"
          />
          <NumberField
            v-for="field in margins"
            :key="field.key"
            :label="field.label"
            :model-value="Number(settings[field.key])"
            :min="0"
            @update:model-value="$event !== null && update({ [field.key]: $event })"
          />
        </div>
      </section>
    </div>
    <div class="settings-footer">
      <button class="text-button" @click="emit('change', defaultLayout())">恢复默认设置</button
      ><button class="button primary compact" @click="emit('close')">完成</button>
    </div>
  </div>
</template>
