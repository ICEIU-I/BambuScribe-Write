<script setup lang="ts">
import { computed } from 'vue'
import AppIcon from '../AppIcon.vue'
import PixelTabs from '../ui/PixelTabs.vue'
import HoverHint from '../ui/HoverHint.vue'
import PageRail from '../editor/PageRail.vue'
import type { Page } from '../../types'
import { t } from '../../i18n'
defineProps<{
  tab: 'write' | 'device'
  theme: 'light' | 'dark'
  pages: Page[]
  index: number
  disabled: boolean
  actionsDisabled: boolean
}>()
const emit = defineEmits<{
  'update:tab': ['write' | 'device']
  'update:theme': ['light' | 'dark']
  open: []
  new: []
  saveAs: []
  page: [number]
  deletePage: [number]
  copyPage: [number]
  blankPage: [number]
  movePage: [number, number]
  collapse: []
  checkUpdate: []
}>()
const tabs = computed(() => [
  { value: 'write', label: t('header.write'), icon: 'pen', controls: 'write-panel' },
  { value: 'device', label: t('header.device'), icon: 'printer', controls: 'device-panel' },
])
</script>
<template>
  <aside id="workspace-sidebar" class="workspace-sidebar" aria-label="工作区导航">
    <div class="sidebar-brand">
      <HoverHint v-slot="hint" text="收起侧栏">
        <button
          class="sidebar-action"
          aria-label="收起侧栏"
          :aria-describedby="hint.describedBy"
          @click="emit('collapse')"
        >
          <AppIcon name="sidebar" :size="20" />
        </button>
      </HoverHint>
    </div>
    <div class="sidebar-file-actions">
      <HoverHint v-slot="hint" :text="`${t('header.newProject')} · ⌘N`">
        <button
          class="sidebar-action"
          :aria-label="t('header.newProject')"
          :aria-describedby="hint.describedBy"
          :disabled="disabled"
          @click="emit('new')"
        >
          <AppIcon name="file-plus" :size="21" />
        </button>
      </HoverHint>
      <HoverHint v-slot="hint" text="打开项目 · ⌘O">
        <button
          class="sidebar-action"
          aria-label="打开项目"
          :aria-describedby="hint.describedBy"
          :disabled="disabled"
          @click="emit('open')"
        >
          <AppIcon name="folder" :size="21" />
        </button>
      </HoverHint>
      <HoverHint v-slot="hint" text="另存为 · ⌘⇧S">
        <button
          class="sidebar-action"
          aria-label="另存为"
          :aria-describedby="hint.describedBy"
          :disabled="disabled"
          @click="emit('saveAs')"
        >
          <AppIcon name="save" :size="20" />
        </button>
      </HoverHint>
    </div>
    <PixelTabs
      :model-value="tab"
      :items="tabs"
      label="主要页面"
      vertical
      icons-only
      @update:model-value="emit('update:tab', $event as 'write' | 'device')"
    />
    <PageRail
      :pages="pages"
      :index="index"
      :actions-disabled="actionsDisabled"
      @page="emit('page', $event)"
      @delete-page="emit('deletePage', $event)"
      @copy-page="emit('copyPage', $event)"
      @blank-page="emit('blankPage', $event)"
      @move-page="(from, to) => emit('movePage', from, to)"
    />
    <div class="sidebar-footer">
      <HoverHint v-slot="hint" :text="t('update.check')">
        <button
          class="sidebar-action"
          :aria-label="t('update.check')"
          :aria-describedby="hint.describedBy"
          @click="emit('checkUpdate')"
        >
          <AppIcon name="download" :size="20" />
        </button>
      </HoverHint>
      <HoverHint
        v-slot="hint"
        :text="theme === 'light' ? t('header.themeToDark') : t('header.themeToLight')"
      >
        <button
          class="sidebar-action theme-toggle"
          :aria-label="theme === 'light' ? t('header.themeToDark') : t('header.themeToLight')"
          :aria-describedby="hint.describedBy"
          @click="emit('update:theme', theme === 'light' ? 'dark' : 'light')"
        >
          <AppIcon :name="theme === 'light' ? 'moon' : 'sun'" :size="20" />
        </button>
      </HoverHint>
    </div>
  </aside>
</template>
