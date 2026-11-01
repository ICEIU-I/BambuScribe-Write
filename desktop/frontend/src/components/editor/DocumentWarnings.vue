<script setup lang="ts">
import { computed, ref } from 'vue'
import AppIcon from '../AppIcon.vue'
import type { ImageAsset } from '../../types'
import {
  readDismissedWarnings,
  saveDismissedWarnings,
  warningKeys,
} from '../../state/dismissedWarnings'

const props = defineProps<{ warnings: string[]; images: ImageAsset[] }>()
const dismissed = ref(readDismissedWarnings())
const sessionDismissed = ref(new Set<string>())
const visible = computed(() =>
  props.warnings.filter((warning) => {
    const keys = warningKeys(warning, props.images)
    return keys.length
      ? keys.some((key) => !dismissed.value.has(key))
      : !sessionDismissed.value.has(warning)
  }),
)

function dismiss() {
  for (const warning of visible.value) {
    const keys = warningKeys(warning, props.images)
    if (keys.length) for (const key of keys) dismissed.value.add(key)
    else sessionDismissed.value.add(warning)
  }
  saveDismissedWarnings(dismissed.value)
}
</script>

<template>
  <div v-if="visible.length" class="document-alert table-warning" role="status">
    <AppIcon name="alert" :size="16" />
    <span class="warning-text">{{ visible.join('；') }}</span>
    <button class="icon-button" aria-label="关闭页面提示" @click="dismiss">
      <AppIcon name="close" :size="15" />
    </button>
  </div>
</template>

<style scoped>
.warning-text {
  flex: 1;
  min-width: 0;
  overflow-wrap: anywhere;
}
.icon-button {
  flex-shrink: 0;
}
</style>
