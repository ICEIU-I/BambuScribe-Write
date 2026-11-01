<script setup lang="ts">
import { nextTick, ref } from 'vue'
import AppIcon from '../AppIcon.vue'
import HoverHint from './HoverHint.vue'
const props = defineProps<{
  modelValue: string
  label: string
  vertical?: boolean
  iconsOnly?: boolean
  items: { value: string; label: string; icon: string; controls: string }[]
}>()
const emit = defineEmits<{ 'update:modelValue': [string] }>()
const root = ref<HTMLElement>()
async function keydown(event: KeyboardEvent) {
  const index = props.items.findIndex((item) => item.value === props.modelValue)
  const last = props.items.length - 1
  const next = {
    ArrowUp: (index + last) % (last + 1),
    ArrowDown: (index + 1) % (last + 1),
    ArrowLeft: (index + last) % (last + 1),
    ArrowRight: (index + 1) % (last + 1),
    Home: 0,
    End: last,
  }[event.key]
  if (next === undefined) return
  event.preventDefault()
  emit('update:modelValue', props.items[next].value)
  await nextTick()
  root.value?.querySelectorAll<HTMLButtonElement>('[role="tab"]')[next]?.focus()
}
</script>
<template>
  <nav
    ref="root"
    class="app-tabs"
    role="tablist"
    :aria-orientation="vertical ? 'vertical' : 'horizontal'"
    :aria-label="label"
    @keydown="keydown"
  >
    <HoverHint v-for="item in items" :key="item.value" v-slot="hint" :text="item.label">
      <button
        :id="`${item.value}-tab`"
        role="tab"
        :aria-label="item.label"
        :aria-describedby="hint.describedBy"
        :aria-selected="modelValue === item.value"
        :aria-controls="item.controls"
        :tabindex="modelValue === item.value ? 0 : -1"
        :class="{ active: modelValue === item.value }"
        @click="emit('update:modelValue', item.value)"
      >
        <AppIcon :name="item.icon" :size="iconsOnly ? 21 : 16" /><span v-if="!iconsOnly">{{
          item.label
        }}</span>
      </button>
    </HoverHint>
  </nav>
</template>
