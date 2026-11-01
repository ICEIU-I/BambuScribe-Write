import type { PrinterState } from '../types'
import { t } from '../i18n'
export function printerPhase(state: PrinterState): { label: string; tone: string } {
  const phases: Record<string, [string, string]> = {
    disconnected: ['job.phaseDisconnected', 'neutral'],
    idle: ['job.phaseIdle', 'info'],
    connecting: ['job.phaseConnecting', 'warn'],
    uploading: ['job.phaseUploading', 'warn'],
    uploaded: ['job.phaseUploaded', 'info'],
    failed: ['job.phaseFailed', 'error'],
    external: ['job.phaseExternal', 'warn'],
  }
  const [key, tone] = phases[state.phase] ?? [
    state.connected ? 'job.phaseConnected' : 'job.phaseNotConnected',
    'neutral',
  ]
  return { label: t(key), tone }
}
