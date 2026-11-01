import { ref } from 'vue'
import { en } from './en'
import { zh } from './zh'

export type Language = 'zh' | 'en'
export type LanguagePreference = 'system' | 'zh' | 'en'

const tables: Record<Language, Record<string, string>> = { zh, en }
const language = ref<Language>('zh')

// The webview is the only side that can read the OS locale, which is why
// "system" is resolved here and the concrete language is what the backend is
// told about.
export function systemLanguage(): Language {
  return (navigator.language || '').toLowerCase().startsWith('zh') ? 'zh' : 'en'
}

export function resolveLanguage(preference: LanguagePreference): Language {
  return preference === 'system' ? systemLanguage() : preference
}

export function setLanguage(preference: LanguagePreference) {
  language.value = resolveLanguage(preference)
}

export function currentLanguage(): Language {
  return language.value
}

// t renders a message in the active language. Reading the locale ref here is
// what makes every template that calls t() re-render on a language change.
// An unknown key renders as the key so a missing translation shows up instead
// of turning into an empty label.
export function t(key: string, args?: Record<string, string | number>): string {
  const format = tables[language.value][key] ?? tables.zh[key] ?? key
  if (!args) return format
  return format.replace(/\{(\w+)\}/g, (_, name: string) =>
    name in args ? String(args[name]) : `{${name}}`,
  )
}
