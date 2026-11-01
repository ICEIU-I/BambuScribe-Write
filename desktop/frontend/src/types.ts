import type { CalculationRecord } from './services/calculationTypes'
import type { TableAsset, TablePlacement } from './state/tableTypes'
import type { DiagramScene } from './services/diagramTypes'
export type {
  TableAsset,
  TableCell,
  TablePlacement,
  TableSelection,
  TableInsertOptions,
} from './state/tableTypes'
export interface Point {
  x: number
  y: number
}
export interface InkLayer {
  objectId: string
  part: string
  source: string
  strokes: Point[][]
}
export interface InkCut {
  objectId: string
  part: string
  source: string
  stroke: number
  ranges: [number, number][]
}
export interface PageErasure {
  cuts?: InkCut[]
  id: string
  page: number
  x: number
  y: number
  width: number
  height: number
  path?: Point[]
  radius?: number
  objectIds?: string[]
}
export interface RegionBounds {
  x: number
  y: number
  width: number
  height: number
}
export interface ImageTextRegion {
  id: string
  text: string
  x: number
  y: number
  width: number
  height: number
  mode: 'handwriting' | 'erase' | 'keep'
  source?: 'ocr' | 'manual'
  confidence?: number
  sourceBounds?: RegionBounds
  ocrVersion?: number
  textFormat?: 'math'
  sourceStrokes?: Point[][]
  strokes?: Point[][]
  erasePath?: Point[]
  eraseRadius?: number
}
// 橡皮擦的两种取范围方式：框选移除矩形内部，自由沿轨迹移除笔刷范围。
export type EraseMode = 'box' | 'free'
export type ImageEditTool = 'select' | 'text' | 'erase'
export interface ImageAsset {
  diagram?: DiagramScene
  id: string
  page: number
  data: string
  x: number
  y: number
  width: number
  height: number
  rotation: number
  mode: 'line' | 'hatch' | 'stipple'
  textRegions?: ImageTextRegion[]
}
export interface ImagePlacement {
  id: string
  x: number
  y: number
  width: number
  height: number
  rotation: number
  mode: ImageAsset['mode']
  strokes: Point[][]
  graphicStrokes?: Point[][]
  sourceStrokes?: Point[][]
  textRegions?: ImageTextRegion[]
}
export interface LayoutSettings {
  handwritingVariation?: number
  handwritingSeed?: number
  fractionScale?: number
  fontId?: 'handwriting' | 'dc'
  pageLayouts?: PageLayout[]
  templateName: string
  lineMode: 'fixed' | 'random'
  lineCount: number
  paperWidth: number
  paperHeight: number
  marginTop: number
  marginRight: number
  marginBottom: number
  marginLeft: number
  fontSize: number
  letterGap: number
  letterSpacing: number
  lineSpacing: number
  lineHeight: number
  paragraphGap: number
  indent: number
  align: 'left' | 'center' | 'right'
}
export interface PageLayout {
  lineCount: number
  continuation?: boolean
}
export interface MissingGlyph extends Point {
  char: string
  size: number
}
export interface CaretStop extends Point {
  index: number
}
export interface GlyphCell extends Point {
  start: number
  end: number
  width: number
  height: number
}
export interface FormulaPlacement extends RegionBounds {
  start: number
  end: number
  source: string
}
export interface Page {
  formulas?: FormulaPlacement[]
  ink?: InkLayer[]
  lineHeight: number
  lineCount: number
  number: number
  strokes: Point[][]
  textStrokes?: Point[][]
  images: ImagePlacement[]
  tables?: TablePlacement[]
  issues?: string[]
  warnings?: string[]
  missing: MissingGlyph[]
  text: string
  cells: GlyphCell[]
  carets: CaretStop[]
  blankCarets?: (CaretStop & { blankLines: number })[]
}
export interface Document {
  erasures?: PageErasure[]
  objectLayout?: ObjectPosition[]
  id: string
  settings: LayoutSettings
  pages: Page[]
  missing: string[]
  characterCount: number
  fontName: string
}
export interface ObjectPosition extends Point {
  id: string
  page: number
  source: RegionBounds
  sourcePage: number
  rotation: number
}
export type LanguagePreference = 'system' | 'zh' | 'en'
export interface MenuState {
  ruled: boolean
  dark: boolean
  device: boolean
  zoom: number
  lang: LanguagePreference
}
export interface Preferences {
  calculations?: CalculationRecord[]
  erasures: PageErasure[]
  theme: 'light' | 'dark'
  language: LanguagePreference
  text: string
  layout: LayoutSettings
  writing: WritingSettings
  images: ImageAsset[]
  tables: TableAsset[]
}
// Editable state stored in a .ccf project file. The manifest revision is
// written by the backend, so the value sent from the editor is only a default.
export interface Project {
  calculations?: CalculationRecord[]
  erasures?: PageErasure[]
  version: number
  text: string
  layout: LayoutSettings
  writing: WritingSettings
  images: ImageAsset[]
  tables?: TableAsset[]
}
export interface ProjectFile {
  path: string
  project: Project
}
export type ProjectAction = 'saveAs' | 'open' | 'reveal'
export interface WritingSettings {
  offsetX: number
  offsetY: number
  penUp: number | null
  penDown: number | null
  originMode: 'manual' | 'auto'
  sound: boolean
}
export interface PrinterConfig {
  ip: string
  accessCode: string
}
export interface PrinterSettings {
  ip: string
  serial: string
  hasAccessCode: boolean
}
export interface PrinterState {
  deviceId?: string
  connected: boolean
  phase: string
  message: string
  progress: number
  jobId: string
  documentId: string
  page: number
  deviceState: string
  totalPages: number
  pageOrdinal: number
  uploadedPages: number
}
export interface Bootstrap {
  projectPath?: string
  preferences: Preferences
  printer: PrinterSettings
  state: PrinterState
  devices?: PrinterDevice[]
  selectedDeviceId: string
  warning: string
}

export interface PrinterDevice {
  id: string
  name: string
  ip: string
  accessCode: string
  state: PrinterState
}
export interface PrinterList {
  devices: PrinterDevice[]
  selectedDeviceId: string
}
