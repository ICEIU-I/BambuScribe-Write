// Native menus and global shortcuts must commit an active inline cell editor
// before taking a project snapshot or checking whether the document is dirty.
export const flushEditorEvent = 'bambuscribe:flush-editors'
export function flushEditors() {
  window.dispatchEvent(new Event(flushEditorEvent))
}
