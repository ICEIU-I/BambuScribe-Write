import type { CalculationSources, CalculationPlan, CalculationPreview } from './calculationTypes'
import type { AgentConfig, Extraction, ImportPlacement, ImportPlan } from './tableImportTypes'
import type { DiagramScene } from './diagramTypes'
import type { EditorPreview } from './editorPreview'
import type { UpdateInfo, UpdateProgress } from './updateTypes'
import type {
  Bootstrap,
  LanguagePreference,
  Document,
  LayoutSettings,
  ImageAsset,
  TableAsset,
  PageErasure,
  Preferences,
  PrinterConfig,
  PrinterState,
  PrinterDevice,
  PrinterList,
  MenuState,
  Project,
  ProjectFile,
  WritingSettings,
} from '../types'

interface NativeAPI {
  PreviewEditor?(
    text: string,
    settings: LayoutSettings,
    images: ImageAsset[],
    tables: TableAsset[],
    erasures: PageErasure[],
  ): Promise<EditorPreview>
  RecognizeCalculations(sources: CalculationSources): Promise<CalculationPlan>
  CancelCalculationRecognition(): Promise<void>
  PreviewCalculations(plan: CalculationPlan, settings: LayoutSettings): Promise<CalculationPreview>
  CheckUpdate(): Promise<UpdateInfo>
  InstallUpdate(): Promise<void>
  RestartUpdate(): Promise<void>
  CancelUpdate(): Promise<void>
  RecognizeDiagram(data: string, instruction: string): Promise<DiagramScene>
  CancelDiagramRecognition(): Promise<void>
  AgentConfig(): Promise<AgentConfig>
  SaveAgentConfig(config: AgentConfig): Promise<void>
  RecognizeTables(data: string, instruction: string): Promise<Extraction>
  CancelTableRecognition(): Promise<void>
  PlanTableImport(
    project: Project,
    extraction: Extraction,
    placement: ImportPlacement,
  ): Promise<ImportPlan>
  EditorRequestActive(id: string): Promise<boolean>
  SetEditorReady(ready: boolean): Promise<void>
  CompleteEditorRequest(id: string, reply: { result?: unknown; error?: string }): Promise<void>
  Bootstrap(): Promise<Bootstrap>
  Preview(text: string, settings: LayoutSettings): Promise<Document>
  PreviewWritingRule(text: string, fontId: string): Promise<Document>
  PreviewWithImages(text: string, settings: LayoutSettings, images: ImageAsset[]): Promise<Document>
  PreviewWithObjects(
    text: string,
    settings: LayoutSettings,
    images: ImageAsset[],
    tables: TableAsset[],
  ): Promise<Document>
  PreviewWithErasures(
    text: string,
    settings: LayoutSettings,
    images: ImageAsset[],
    tables: TableAsset[],
    erasures: PageErasure[],
  ): Promise<Document>
  SavePreferences(preferences: Preferences, projectPath: string): Promise<void>
  CompleteClose(): Promise<void>
  ExportPage(id: string, index: number, writing: WritingSettings): Promise<string>
  ExportAll(id: string, writing: WritingSettings): Promise<string>
  SaveProject(project: Project): Promise<string>
  SaveProjectAs(project: Project): Promise<string>
  OpenProject(): Promise<ProjectFile | null>
  NewProject(): Promise<void>
  RevealProject(path: string): Promise<void>
  SetLanguage(preference: LanguagePreference, resolved: string): Promise<void>
  SetMenuState(state: MenuState): Promise<void>
  ConfirmDiscardProject(): Promise<'save' | 'discard' | 'cancel'>
  ConnectPrinter(config: PrinterConfig): Promise<void>
  Printers(): Promise<PrinterList>
  SavePrinterDevice(device: Omit<PrinterDevice, 'state'>): Promise<PrinterList>
  DeletePrinterDevice(id: string): Promise<PrinterList>
  SelectPrinter(id: string): Promise<PrinterList>
  ConnectDevice(id: string): Promise<void>
  DisconnectDevice(id: string): Promise<void>
  CancelDeviceUpload(id: string): Promise<void>
  UploadDevicePages(
    deviceId: string,
    id: string,
    prefix: string,
    indices: number[],
    writing: WritingSettings,
  ): Promise<string>
  DisconnectPrinter(): Promise<void>
  PrinterState(): Promise<PrinterState>
  CancelUpload(): Promise<void>
  UploadPages(
    id: string,
    prefix: string,
    indices: number[],
    writing: WritingSettings,
  ): Promise<string>
}

declare global {
  interface Window {
    go?: { application: { App: NativeAPI } }
    runtime?: {
      EventsOn(name: string, callback: (payload: never) => void): () => void
      WindowSetTitle(title: string): void
    }
  }
}

export function native(): NativeAPI {
  const api = window.go?.application?.App
  if (!api) throw new Error('桌面服务尚未就绪，请从应用程序打开。')
  return api
}

export function onPrinterState(callback: (state: PrinterState) => void): () => void {
  return window.runtime?.EventsOn('printer:state', callback) ?? (() => {})
}

// The native menu runs on the Go side; it only sends the action name.
export function onMenuAction(callback: (action: string) => void): () => void {
  return window.runtime?.EventsOn('menu:action', callback) ?? (() => {})
}

export function onUpdateProgress(callback: (progress: UpdateProgress) => void): () => void {
  return window.runtime?.EventsOn('update:progress', callback) ?? (() => {})
}

export function setWindowTitle(title: string) {
  window.runtime?.WindowSetTitle(title)
}

export const snapshot = <T>(value: T): T => JSON.parse(JSON.stringify(value)) as T
export const errorText = (error: unknown): string =>
  error instanceof Error ? error.message : String(error)
