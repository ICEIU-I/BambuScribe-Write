import type { Project, TableAsset, TablePlacement, LayoutSettings } from '../types'
import type { TableBorders } from '../state/tableTypes'
export interface RecognizedCell {
  row: number
  col: number
  rowSpan: number
  colSpan: number
  text: string
  uncertain: boolean
  borders: TableBorders | null
}
export interface Extraction {
  tables: {
    title: string
    notes: string[]
    rows: number
    columns: number[]
    style: string
    cells: RecognizedCell[]
  }[]
}
export interface ImportPlacement {
  page: number
  y: number
  afterId?: string
  atEnd?: boolean
  positions?: ImportPosition[]
}
export interface ImportPosition {
  page: number
  x: number
  y: number
  width: number
}
export interface ImportGroup extends ImportPosition {
  index: number
  title: string
  height: number
  objectIds: string[]
}
export interface ImportPreview extends ImportPlan {
  settings: LayoutSettings
  revision: number
  opened: number
  occupied: { page: number; x: number; y: number; width: number; height: number }[]
}
export interface ImportPlan {
  objects: TableAsset[]
  warnings: string[]
  pages: number[]
  groups: ImportGroup[]
  preview: TablePlacement[]
}
export interface AgentConfig {
  baseURL: string
  model: string
  protocol: 'responses' | 'chat-completions'
  apiKey?: string
  hasKey?: boolean
}
export interface ImportRequest {
  extraction: Extraction
  placement?: ImportPlacement
  expectedRevision?: number
}
export interface LiveRequest {
  id: string
  action: 'snapshot' | 'import'
  params: ImportRequest & { full?: boolean }
}
export interface EditorSnapshot {
  project: Project
  revision: number
  page: number
  projectPath: string
}
