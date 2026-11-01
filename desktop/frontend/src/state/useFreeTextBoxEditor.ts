import { nextTick, onBeforeUnmount, ref, watch, type Ref } from 'vue'
import type { TableAsset } from './tableTypes'
import { setCellText } from './tableOperations'
import { flushEditorEvent } from './editorCommands'

export function useFreeTextBoxEditor(
  props: { asset: TableAsset; selected: boolean; disabled: boolean; startEditing: boolean },
  host: Ref<HTMLElement | undefined>,
  select: () => void,
  change: (asset: TableAsset) => void,
  editEnd: () => void,
) {
  const input = ref<HTMLTextAreaElement>(),
    editing = ref(false),
    text = ref('')
  async function edit() {
    if (props.disabled) return
    select()
    text.value = props.asset.cells[0]!.text
    editing.value = true
    await nextTick()
    input.value?.focus({ preventScroll: true })
    input.value?.select()
  }
  function save() {
    if (!editing.value) return
    editing.value = false
    editEnd()
    if (text.value !== props.asset.cells[0]!.text)
      change(setCellText(props.asset, 0, 0, text.value))
  }
  function key(e: KeyboardEvent) {
    if (e.isComposing || props.disabled) return
    const k = e.key.toLowerCase()
    if ((e.metaKey || e.ctrlKey) && (k === 's' || k === 'o')) {
      save()
      return
    }
    if (k === 'escape') {
      e.preventDefault()
      e.stopPropagation()
      editing.value = false
      editEnd()
      host.value?.focus({ preventScroll: true })
    } else if ((k === 'enter' && !e.shiftKey) || k === 'tab') {
      e.preventDefault()
      e.stopPropagation()
      save()
      host.value?.focus({ preventScroll: true })
    }
  }
  watch(
    () => props.startEditing,
    (open) => {
      if (open) void edit()
    },
    { immediate: true },
  )
  watch(
    () => props.selected,
    (selected) => {
      if (!selected) save()
    },
  )
  window.addEventListener(flushEditorEvent, save)
  onBeforeUnmount(() => window.removeEventListener(flushEditorEvent, save))
  return { input, editing, text, edit, save, key }
}
