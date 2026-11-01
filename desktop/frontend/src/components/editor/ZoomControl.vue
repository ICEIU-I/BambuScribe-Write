<script setup lang="ts">
import AppIcon from '../AppIcon.vue'
defineProps<{ modelValue: number }>()
const emit = defineEmits<{ 'update:modelValue': [number] }>()
</script>
<template>
  <div class="zoom-controls" title="触摸板捏合或 Ctrl/⌘ + 滚轮缩放">
    <button
      class="icon-button"
      aria-label="缩小"
      :disabled="modelValue <= 30"
      @click="emit('update:modelValue', Math.max(30, modelValue - 10))"
    >
      <AppIcon name="minus" :size="14" />
    </button>
    <select
      :value="modelValue"
      aria-label="页面缩放"
      @change="emit('update:modelValue', Number(($event.target as HTMLSelectElement).value))"
    >
      <option
        v-for="value in [...new Set([30, 50, 65, 75, 100, 125, 150, modelValue])].sort(
          (a, b) => a - b,
        )"
        :key="value"
        :value="value"
      >
        {{ Math.round(value) }}%
      </option>
    </select>
    <button
      class="icon-button"
      aria-label="放大"
      :disabled="modelValue >= 150"
      @click="emit('update:modelValue', Math.min(150, modelValue + 10))"
    >
      <AppIcon name="plus" :size="14" />
    </button>
  </div>
</template>
