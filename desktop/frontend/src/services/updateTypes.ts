export interface UpdateInfo {
  current: string
  version: string
  available: boolean
  notes: string
  url: string
}

export interface UpdateProgress {
  phase: 'downloading' | 'verifying' | 'restarting'
  percent: number
}
