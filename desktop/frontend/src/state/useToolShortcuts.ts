import { onBeforeUnmount, onMounted, type Ref } from 'vue'
import type { ImageEditTool } from '../types'

// Tool shortcuts for the writing workspace. Digits carry a modifier so they
// keep working while the paper textarea has focus; the bare letters only fire
// outside text fields, where they would otherwise be typed into the paper.
// `allow` mirrors the toolbar's disabled state so a shortcut cannot reach a
// tool the buttons refuse to offer.
export function useToolShortcuts(
  tool: Ref<ImageEditTool>,
  enabled: Ref<boolean>,
  allow: (tool: ImageEditTool) => boolean = () => true,
) {
  const byKey: Record<string, ImageEditTool> = { v: 'select', t: 'text', e: 'erase' }
  const byDigit: Record<string, ImageEditTool> = { '1': 'select', '2': 'text', '3': 'erase' }
  function pick(next: ImageEditTool) {
    if (allow(next)) tool.value = next
  }
  function keydown(event: KeyboardEvent) {
    if (event.isComposing || event.defaultPrevented || !enabled.value) return
    const target = event.target as HTMLElement | null
    if (target?.closest('dialog,[role="dialog"]')) return
    const key = event.key.toLowerCase()
    const command = event.metaKey || event.ctrlKey
    if (command && !event.altKey && byDigit[key]) {
      event.preventDefault()
      pick(byDigit[key])
      return
    }
    if (command || event.altKey) return
    // Escape returns to the pointer tool from anywhere in the workspace. It
    // leaves the event alone so the editor still clears its own selection and
    // an open text box still cancels its edit from the same keypress.
    if (key === 'escape') {
      tool.value = 'select'
      return
    }
    if (target?.closest('input,textarea,[contenteditable="true"]')) return
    if (byKey[key]) {
      event.preventDefault()
      pick(byKey[key])
    }
  }
  onMounted(() => window.addEventListener('keydown', keydown, true))
  onBeforeUnmount(() => window.removeEventListener('keydown', keydown, true))
}
