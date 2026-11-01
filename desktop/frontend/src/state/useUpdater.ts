import { computed, onBeforeUnmount, ref, type Ref } from 'vue'
import { errorText, native, onUpdateProgress } from '../services/bridge'
import type { UpdateInfo } from '../services/updateTypes'
import { t } from '../i18n'

interface Workspace {
  ready: Ref<boolean>
  busy: Ref<boolean>
  flushPersistence(): Promise<void>
}

export function useUpdater(w: Workspace) {
  const open = ref(false)
  const info = ref<UpdateInfo>()
  const phase = ref('idle')
  const percent = ref(0)
  const error = ref('')
  let cancelled = false
  const installing = computed(() =>
    ['saving', 'downloading', 'verifying', 'restarting'].includes(phase.value),
  )
  const checking = computed(() => phase.value === 'checking')
  const stop = onUpdateProgress((progress) => {
    if (!installing.value) return
    phase.value = progress.phase
    percent.value = Math.min(100, Math.max(0, progress.percent))
  })
  onBeforeUnmount(stop)

  async function check() {
    open.value = true
    if (checking.value || installing.value) return
    phase.value = 'checking'
    error.value = ''
    info.value = undefined
    try {
      info.value = await native().CheckUpdate()
      phase.value = info.value.available ? 'available' : 'current'
    } catch (e) {
      error.value = errorText(e)
      phase.value = 'idle'
    }
  }

  async function install() {
    if (checking.value || installing.value || !info.value?.available) return
    if (!w.ready.value || w.busy.value) {
      error.value = t('update.busy')
      return
    }
    w.busy.value = true
    cancelled = false
    phase.value = 'saving'
    error.value = ''
    percent.value = 0
    try {
      await w.flushPersistence()
      phase.value = 'downloading'
      await native().InstallUpdate()
      if (cancelled) throw new Error(t('update.cancelled'))
      phase.value = 'saving'
      await w.flushPersistence()
      await native().RestartUpdate()
    } catch (e) {
      error.value = errorText(e)
      phase.value = 'available'
      await native()
        .CancelUpdate()
        .catch(() => {})
    } finally {
      if (phase.value !== 'restarting') w.busy.value = false
    }
  }

  async function cancel() {
    if (phase.value === 'saving' || phase.value === 'restarting') return
    if (installing.value) cancelled = true
    try {
      await native().CancelUpdate()
    } catch (e) {
      error.value = errorText(e)
    }
  }

  function close() {
    if (installing.value) return
    if (checking.value) void cancel()
    open.value = false
  }

  return { open, info, phase, percent, error, installing, checking, check, install, cancel, close }
}
