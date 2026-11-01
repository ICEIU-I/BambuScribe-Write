<script setup lang="ts">
import { computed, ref } from 'vue'
import type { LayoutSettings } from '../../types'

const props = defineProps<{ settings: LayoutSettings }>()
const emit = defineEmits<{ change: [Partial<LayoutSettings>] }>()
const enabled = computed(() => (props.settings.handwritingVariation ?? 0) > 0)
const previous = ref(20)
function newSeed() {
  const seed = crypto.getRandomValues(new Uint32Array(1))[0]!
  return seed === props.settings.handwritingSeed ? (seed + 1) >>> 0 : seed
}
function toggle() {
  if (enabled.value) previous.value = props.settings.handwritingVariation!
  emit('change', {
    handwritingVariation: enabled.value ? 0 : previous.value,
    handwritingSeed: props.settings.handwritingSeed ?? newSeed(),
  })
}
</script>
<template>
  <div class="handwriting-variation">
    <label class="checkbox-row">
      <input type="checkbox" :checked="enabled" @change="toggle" />手写变化
    </label>
    <div v-if="enabled" class="variation-controls">
      <label>
        <span title="同时微调字形、字距和行距">变化强度</span>
        <input
          type="range"
          aria-label="手写变化强度"
          min="1"
          max="100"
          step="1"
          :value="settings.handwritingVariation"
          @input="
            emit('change', {
              handwritingVariation: Number(($event.target as HTMLInputElement).value),
            })
          "
        />
        <output>{{ settings.handwritingVariation }}%</output>
      </label>
      <button
        class="button secondary compact"
        type="button"
        @click="emit('change', { handwritingSeed: newSeed() })"
      >
        换一组
      </button>
    </div>
  </div>
</template>
<style scoped>
.handwriting-variation {
  display: grid;
  gap: 10px;
  margin-top: 14px;
}
.variation-controls {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}
.variation-controls label {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  font-size: 12px;
}
.variation-controls input {
  width: 64px;
  min-width: 48px;
  flex: 1;
  accent-color: var(--accent);
}
.variation-controls output {
  min-width: 4ch;
  font-variant-numeric: tabular-nums;
}
</style>
