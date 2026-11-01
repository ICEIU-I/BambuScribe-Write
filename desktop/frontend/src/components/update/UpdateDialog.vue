<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import AppIcon from '../AppIcon.vue'
import { t } from '../../i18n'
import type { useUpdater } from '../../state/useUpdater'
const props = defineProps<{ updater: ReturnType<typeof useUpdater> }>()
const dialog = ref<HTMLDialogElement>()
watch(
  () => props.updater.open.value,
  async (open) => {
    await nextTick()
    if (open) dialog.value?.showModal()
    else dialog.value?.close()
  },
  { immediate: true },
)
</script>

<template>
  <Teleport to="body">
    <dialog
      ref="dialog"
      class="update-dialog"
      :aria-label="t('update.title')"
      @cancel.prevent="updater.close()"
    >
      <div class="update-heading">
        <strong>{{ t('update.title') }}</strong>
        <button
          class="icon-button"
          :aria-label="t('update.close')"
          :disabled="updater.installing.value"
          @click="updater.close()"
        >
          <AppIcon name="close" :size="16" />
        </button>
      </div>
      <p v-if="updater.info.value" class="update-version">
        {{ t('update.currentVersion', { version: updater.info.value.current }) }}
      </p>
      <p v-if="updater.checking.value" role="status">{{ t('update.checking') }}</p>
      <p v-else-if="updater.phase.value === 'current'" role="status">{{ t('update.latest') }}</p>
      <template v-if="updater.info.value?.available">
        <p class="update-version">
          <strong>{{ t('update.newVersion', { version: updater.info.value.version }) }}</strong>
        </p>
        <details v-if="updater.info.value.notes">
          <summary>{{ t('update.notes') }}</summary>
          <p class="update-notes">{{ updater.info.value.notes }}</p>
        </details>
        <p v-if="!updater.installing.value" class="hint">{{ t('update.restartHint') }}</p>
      </template>
      <div v-if="updater.installing.value" class="update-progress" role="status" aria-live="polite">
        <p>
          {{ t(`update.${updater.phase.value}`) }}
          <span v-if="updater.phase.value === 'downloading'">{{ updater.percent.value }}%</span>
        </p>
        <progress
          v-if="updater.phase.value === 'downloading'"
          :value="updater.percent.value"
          max="100"
          :aria-label="t('update.downloading')"
        ></progress>
      </div>
      <p v-if="updater.error.value" class="update-error" role="alert">{{ updater.error.value }}</p>
      <div class="button-row update-actions">
        <button
          v-if="updater.phase.value === 'downloading' || updater.phase.value === 'verifying'"
          class="button secondary"
          @click="updater.cancel()"
        >
          {{ t('update.cancel') }}
        </button>
        <button
          v-if="updater.info.value?.available"
          class="button primary"
          :disabled="updater.installing.value || updater.checking.value"
          @click="updater.install()"
        >
          {{ t('update.install') }}
        </button>
        <button
          v-else
          class="button primary"
          :disabled="updater.checking.value"
          @click="updater.check()"
        >
          {{ t('update.check') }}
        </button>
      </div>
    </dialog>
  </Teleport>
</template>

<style scoped>
.update-dialog {
  width: 440px;
  max-width: calc(100vw - 32px);
  padding: 20px;
  color: var(--text);
  background: var(--elevated);
  border: var(--edge) solid var(--border);
  border-radius: 14px;
  box-shadow: var(--shadow);
}
.update-dialog::backdrop {
  background: rgb(0 0 0 / 30%);
}
.update-heading {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  margin-bottom: 20px;
}
.update-heading strong {
  font-size: 16px;
}
.update-version {
  margin: 12px 0;
}
.update-actions {
  justify-content: flex-end;
  margin-top: 20px;
}
.update-notes {
  white-space: pre-wrap;
  max-height: 240px;
  overflow: auto;
  font-size: 12px;
  line-height: 1.6;
  overflow-wrap: anywhere;
}
summary {
  cursor: pointer;
  font-size: 12px;
}
.update-progress progress {
  display: block;
  width: 100%;
  accent-color: var(--accent);
}
.update-error {
  color: var(--danger);
  font-size: 12px;
  overflow-wrap: anywhere;
}
</style>
