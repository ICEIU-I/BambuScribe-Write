import type { CalculationRecord } from '../services/calculationTypes'
import type { ImageAsset, TableAsset, PageErasure, PageLayout } from '../types'

export interface EditorSnapshot {
  calculations?: CalculationRecord[]
  pageLayouts?: PageLayout[]
  erasures?: PageErasure[]
  text: string
  images: ImageAsset[]
  tables?: TableAsset[]
  tableId?: string
  imageId: string
  regionId: string
  start: number
  end: number
}
interface Entry {
  before: EditorSnapshot
  after: EditorSnapshot
  group?: string
  time: number
}

// One chronology for text and objects. Mouse gestures commit only at pointerup;
// adjacent typing events may share a step, but never cross an object edit.
export function createEditorHistory(limit = 80) {
  const past: Entry[] = [],
    future: Entry[] = []
  let boundary = true
  return {
    record(before: EditorSnapshot, after: EditorSnapshot, group?: string, time = Date.now()) {
      if (
        before.calculations === after.calculations &&
        before.text === after.text &&
        before.images === after.images &&
        before.tables === after.tables &&
        before.erasures === after.erasures &&
        before.pageLayouts === after.pageLayouts
      )
        return
      const last = past.at(-1)
      if (
        !boundary &&
        group &&
        last?.group === group &&
        time - last.time < 700 &&
        last.after.calculations === before.calculations &&
        last.after.text === before.text &&
        last.after.images === before.images &&
        last.after.tables === before.tables &&
        last.after.erasures === before.erasures &&
        last.after.pageLayouts === before.pageLayouts
      ) {
        last.after = after
        last.time = time
      } else {
        past.push({ before, after, group, time })
        if (past.length > limit) past.shift()
      }
      boundary = false
      future.length = 0
    },
    undo() {
      boundary = true
      const e = past.pop()
      if (!e) return
      future.push(e)
      return e.before
    },
    redo() {
      boundary = true
      const e = future.pop()
      if (!e) return
      past.push(e)
      return e.after
    },
    breakGroup() {
      boundary = true
    },
    reset() {
      past.length = 0
      future.length = 0
      boundary = true
    },
  }
}
