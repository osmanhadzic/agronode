import { useCallback, useSyncExternalStore } from 'react'

import { bs } from './locales/bs'
import { de } from './locales/de'
import { en, type TranslationKey, type Translations } from './locales/en'
import { es } from './locales/es'
import { fr } from './locales/fr'
import { tr } from './locales/tr'

export type Language = 'bs' | 'en' | 'de' | 'es' | 'fr' | 'tr'

export type { TranslationKey }

export const LANGUAGES: { code: Language; label: string; locale: string }[] = [
  { code: 'bs', label: 'Bosanski', locale: 'bs-BA' },
  { code: 'en', label: 'English', locale: 'en-GB' },
  { code: 'de', label: 'Deutsch', locale: 'de-DE' },
  { code: 'es', label: 'Español', locale: 'es-ES' },
  { code: 'fr', label: 'Français', locale: 'fr-FR' },
  { code: 'tr', label: 'Türkçe', locale: 'tr-TR' },
]

const dictionaries: Record<Language, Translations> = { bs, en, de, es, fr, tr }

const STORAGE_KEY = 'agronode.language'
const DEFAULT_LANGUAGE: Language = 'bs'

function isLanguage(value: string | null | undefined): value is Language {
  return LANGUAGES.some((language) => language.code === value)
}

function detectInitialLanguage(): Language {
  try {
    const stored = window.localStorage.getItem(STORAGE_KEY)
    if (isLanguage(stored)) {
      return stored
    }
  } catch {
    // localStorage may be unavailable; fall through to browser detection.
  }

  const browserLanguage = navigator.language?.slice(0, 2).toLowerCase()
  if (isLanguage(browserLanguage)) {
    return browserLanguage
  }

  if (browserLanguage === 'hr' || browserLanguage === 'sr') {
    return 'bs'
  }

  return DEFAULT_LANGUAGE
}

let currentLanguage: Language = detectInitialLanguage()
const listeners = new Set<() => void>()

function subscribe(listener: () => void) {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}

export function getLanguage(): Language {
  return currentLanguage
}

export function setLanguage(language: Language) {
  if (language === currentLanguage) {
    return
  }

  currentLanguage = language
  try {
    window.localStorage.setItem(STORAGE_KEY, language)
  } catch {
    // Persisting the choice is best-effort.
  }
  document.documentElement.lang = language
  listeners.forEach((listener) => listener())
}

export function getLocaleTag(language: Language = currentLanguage): string {
  return LANGUAGES.find((item) => item.code === language)?.locale ?? 'en-GB'
}

export type TranslationParams = Record<string, string | number>

function format(language: Language, key: TranslationKey, params?: TranslationParams): string {
  const template = dictionaries[language][key] ?? en[key]
  if (!params) {
    return template
  }

  return template.replace(/\{(\w+)\}/g, (match, name: string) =>
    name in params ? String(params[name]) : match,
  )
}

// Non-reactive translation for event handlers and async callbacks.
export function translate(key: TranslationKey, params?: TranslationParams): string {
  return format(currentLanguage, key, params)
}

export function useI18n() {
  const language = useSyncExternalStore(subscribe, getLanguage, getLanguage)
  const t = useCallback(
    (key: TranslationKey, params?: TranslationParams) => format(language, key, params),
    [language],
  )

  return { t, language, setLanguage, locale: getLocaleTag(language) }
}

if (typeof document !== 'undefined') {
  document.documentElement.lang = currentLanguage
}
