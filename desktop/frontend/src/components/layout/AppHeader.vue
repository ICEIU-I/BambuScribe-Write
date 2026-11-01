<script setup lang="ts">
import { computed, ref } from 'vue'
import type { PrinterState } from '../../types'
import AppIcon from '../AppIcon.vue'
import PixelPopover from '../ui/PixelPopover.vue'
import { t } from '../../i18n'
const props = defineProps<{
  tab: 'write' | 'device'
  title: string
  modified: boolean
  sidebarOpen: boolean
  canOutput: boolean
  outputReasons: { current: string; all: string }
  printer: PrinterState
  pages: number
  characters: number
}>()
const emit = defineEmits<{ sidebar: []; upload: []; export: [boolean] }>()
const exportOpen = ref(false),
  exportButton = ref<HTMLButtonElement>()
const canUpload = computed(
  () =>
    props.canOutput &&
    ['idle', 'finished', 'failed', 'uploaded', 'disconnected'].includes(props.printer.phase),
)
function exportDocument(all: boolean) {
  exportOpen.value = false
  emit('export', all)
}
</script>
<template>
  <header class="app-header">
    <button
      v-if="!sidebarOpen"
      class="icon-button sidebar-toggle"
      aria-label="展开侧栏"
      aria-controls="workspace-sidebar"
      :aria-expanded="sidebarOpen"
      @click="emit('sidebar')"
    >
      <AppIcon name="sidebar" :size="18" />
    </button>
    <div class="document-heading">
      <div class="document-breadcrumb">
        <span>{{ tab === 'write' ? t('header.write') : t('header.device') }}</span>
        <AppIcon name="chevronRight" :size="12" />
        <strong :title="title">{{ title }}</strong>
        <span
          v-if="modified"
          class="unsaved-mark"
          :title="t('app.modified')"
          :aria-label="t('app.modified')"
        ></span>
      </div>
      <div class="document-meta">{{ pages }} 页<span>·</span>{{ characters }} 字</div>
    </div>
    <div class="header-actions">
      <button
        ref="exportButton"
        class="button secondary export-button"
        :aria-expanded="exportOpen"
        aria-haspopup="dialog"
        @click="exportOpen = true"
      >
        <AppIcon name="download" :size="15" />导出
      </button>
      <button
        class="button primary upload-button"
        :disabled="!canUpload"
        aria-haspopup="dialog"
        :title="
          canOutput
            ? printer.connected
              ? t('header.uploadHint')
              : t('header.uploadHintOffline')
            : outputReasons.all
        "
        @click="emit('upload')"
      >
        <AppIcon name="upload" :size="15" />{{
          printer.phase === 'uploading' ? t('header.uploading') : t('header.upload')
        }}
      </button>
    </div>
    <PixelPopover
      :open="exportOpen"
      :anchor="exportButton"
      :width="320"
      label="导出文档"
      @close="exportOpen = false"
    >
      <div class="export-options">
        <div
          v-for="option in [
            { all: false, label: '导出当前页', type: 'G-code', reason: outputReasons.current },
            { all: true, label: '导出全部页面', type: 'ZIP', reason: outputReasons.all },
          ]"
          :key="String(option.all)"
          class="export-option"
        >
          <button :disabled="!!option.reason" @click="exportDocument(option.all)">
            <span>{{ option.label }}</span
            ><small>{{ option.type }}</small>
          </button>
          <p v-if="option.reason">{{ option.reason }}</p>
        </div>
      </div>
    </PixelPopover>
  </header>
</template>
