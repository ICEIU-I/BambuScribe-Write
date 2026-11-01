<script setup lang="ts">
withDefaults(
  defineProps<{
    label: string
    modelValue: number | null
    unit?: string
    min?: number
    max?: number
    step?: number
    disabled?: boolean
    nullable?: boolean
  }>(),
  { unit: 'mm', step: 0.5 },
)
const emit = defineEmits<{ 'update:modelValue': [number | null] }>()
function change(event: Event, nullable: boolean) {
  const input = event.target as HTMLInputElement
  if (Number.isFinite(input.valueAsNumber)) emit('update:modelValue', input.valueAsNumber)
  else if (nullable) emit('update:modelValue', null)
}
</script>
<template>
  <label class="number-field"
    ><span>{{ label }}</span>
    <div class="number-control">
      <input
        type="number"
        :aria-label="label"
        :value="modelValue ?? ''"
        :min="min"
        :max="max"
        :step="step"
        :disabled="disabled"
        :placeholder="nullable ? '待设置' : ''"
        @change="change($event, !!nullable)"
      /><small>{{ unit }}</small>
    </div></label
  >
</template>
