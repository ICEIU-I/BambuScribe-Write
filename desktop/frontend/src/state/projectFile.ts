import type {
  ImageAsset,
  PageErasure,
  TableAsset,
  LayoutSettings,
  Preferences,
  Project,
  WritingSettings,
} from '../types'
import { normalizePreferences } from './normalizePreferences'

// The manifest revision written by the backend. Anything newer is refused there.
export const projectVersion = 6

export interface ProjectSource {
  text: string
  layout: LayoutSettings
  writing: WritingSettings
  images: ImageAsset[]
  tables?: TableAsset[]
  erasures?: PageErasure[]
}

export function projectPayload(source: ProjectSource): Project {
  return {
    version: projectVersion,
    text: source.text,
    layout: source.layout,
    writing: source.writing,
    images: source.images,
    tables: source.tables ?? [],
    erasures: source.erasures ?? [],
  }
}

// A project carries the document, not the workspace, so the theme stays as the
// user left it and everything else goes through the same normalisation as the
// startup draft. That keeps older files readable without a migration step.
export function projectPreferences(current: Preferences, project: Project): Preferences {
  return normalizePreferences({
    theme: current.theme,
    language: current.language,
    text: project.text,
    layout: project.layout,
    writing: project.writing,
    images: project.images,
    tables: project.tables ?? [],
    erasures: project.erasures ?? [],
  })
}
