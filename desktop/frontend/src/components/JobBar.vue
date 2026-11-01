<script setup lang="ts">
import { computed } from 'vue'
import type { PrinterState } from '../types'
import AppIcon from './AppIcon.vue'
import { printerPhase } from '../state/printerPresentation'
import { t } from '../i18n'
import type { SaveStatus } from '../state/useWorkspaceAutosave'
const props = defineProps<{
  state: PrinterState
  saveStatus: SaveStatus
  projectPath: string
  deviceName?: string
}>()
const saveLabel = computed(() =>
  props.saveStatus
    ? t(
        props.saveStatus === 'saved' && !props.projectPath
          ? 'status.draftSaved'
          : `status.${props.saveStatus === 'error' ? 'unsaved' : props.saveStatus}`,
      )
    : '',
)
const emit = defineEmits<{ cancel: []; device: [] }>()
const uploading = computed(() => props.state.phase === 'uploading')
const progress = computed(() => Math.min(100, Math.max(0, props.state.progress || 0)))
const phase = computed(() => printerPhase(props.state))
const pageProgress = computed(() =>
  t('job.batchProgress', {
    page: props.state.page,
    ordinal: props.state.pageOrdinal || 1,
    total: props.state.totalPages || 1,
    progress: progress.value,
  }),
)
</script>
<template>
  <footer class="job-bar">
    <div class="connection-status">
      <span class="status-dot" :class="{ online: state.connected }" aria-hidden="true"></span>
      <button class="text-button" @click="emit('device')">
        {{ deviceName || 'A1' }} ·
        {{ state.connected ? t('job.connected') : t('job.disconnected') }}
      </button>
      <template
        v-if="state.message && state.phase !== 'disconnected' && state.phase !== 'connecting'"
      >
        <span class="status-divider" aria-hidden="true"></span>
        <span class="status-message" :title="state.message">{{ state.message }}</span>
      </template>
    </div>
    <span
      class="save-status"
      :class="{ error: saveStatus === 'error', saving: saveStatus === 'saving' }"
      role="status"
      >{{ saveLabel }}</span
    >
    <div class="job-summary" aria-live="polite">
      <span class="badge" :class="phase.tone">{{ phase.label }}</span>
      <span v-if="uploading" class="task-page">{{ pageProgress }}</span>
      <span v-else-if="state.phase === 'uploaded'" class="task-page">{{
        t('job.uploadedPages', { count: state.uploadedPages })
      }}</span>
    </div>
    <div v-if="uploading" class="job-actions">
      <button class="button compact danger-quiet" @click="emit('cancel')">
        <AppIcon name="stop" :size="14" />{{ t('job.cancelUpload') }}
      </button>
    </div>
    <div
      v-if="uploading"
      class="job-progress"
      :style="{ '--progress-width': `${progress}%` }"
      role="progressbar"
      :aria-label="t('job.uploadProgress')"
      :aria-valuenow="progress"
      :aria-valuemin="0"
      :aria-valuemax="100"
    ></div>
  </footer>
</template>
