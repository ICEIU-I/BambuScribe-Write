import type { ImageAsset } from '../types'

const storageKey = 'bambuscribe.dismissed-image-warnings.v1'
const limit = 500

// Image IDs survive moving, resizing, saving and reopening. A new image or
// changed recognition note gets its own acknowledgement.
export function warningKeys(warning: string, images: ImageAsset[]): string[] {
  const matching = images.filter((image) =>
    image.diagram?.uncertainties?.some((note) => `图像识别提示：${note}` === warning),
  )
  return (matching.length ? matching : images).map((image) => JSON.stringify([image.id, warning]))
}

export function readDismissedWarnings(): Set<string> {
  try {
    const stored: unknown = JSON.parse(window.localStorage.getItem(storageKey) ?? '[]')
    if (Array.isArray(stored))
      return new Set(
        stored.filter((value): value is string => typeof value === 'string').slice(-limit),
      )
  } catch {
    // A disabled or damaged local store must not interrupt editing.
  }
  return new Set()
}

export function saveDismissedWarnings(dismissed: Set<string>) {
  try {
    window.localStorage.setItem(storageKey, JSON.stringify([...dismissed].slice(-limit)))
  } catch {
    // Keep the in-memory dismissal when local storage is unavailable.
  }
}
