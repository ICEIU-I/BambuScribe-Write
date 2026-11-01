import type { Preferences, PrinterState } from '../types'
import { defaultFontPercent, fontMillimeters } from './fontSizing'

export const defaultLayout = (): Preferences['layout'] => ({
  templateName: '新建笔记',
  lineMode: 'fixed',
  lineCount: 16,
  paperWidth: 163,
  paperHeight: 240,
  marginTop: 0,
  marginRight: 0,
  marginBottom: 0,
  marginLeft: 0,
  fontSize: fontMillimeters(defaultFontPercent, 15),
  letterGap: 0,
  letterSpacing: 100,
  lineSpacing: 100,
  lineHeight: 15,
  paragraphGap: 0,
  indent: 0,
  align: 'left',
})
export const defaultPreferences = (): Preferences => ({
  theme: 'light',
  language: 'system',
  text: '',
  layout: defaultLayout(),
  writing: { offsetX: 0, offsetY: 0, penUp: 7, penDown: 3.4, originMode: 'auto', sound: true },
  images: [],
  tables: [],
  erasures: [],
})
export const defaultPrinter = (): PrinterState => ({
  connected: false,
  phase: 'disconnected',
  message: '',
  progress: 0,
  jobId: '',
  documentId: '',
  page: 0,
  deviceState: '',
  totalPages: 0,
  pageOrdinal: 0,
  uploadedPages: 0,
})
