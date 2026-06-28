import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';
import zh from './locales/zh.json';
import en from './locales/en.json';

export const SUPPORTED_LANGUAGES = ['zh', 'en'];
export const DEFAULT_LANGUAGE = 'en';
export const LANGUAGE_STORAGE_KEY = 'octolink.lang';

const resources = {
  zh: { translation: zh },
  en: { translation: en },
};

if (!i18n.isInitialized) {
  i18n
    .use(LanguageDetector)
    .use(initReactI18next)
    .init({
      resources,
      // Default to English when no preference is stored.
      fallbackLng: DEFAULT_LANGUAGE,
      supportedLngs: SUPPORTED_LANGUAGES,
      // Map zh-CN / zh-* down to the zh catalog.
      load: 'languageOnly',
      nonExplicitSupportedLngs: true,
      interpolation: {
        // React already escapes; allow rich text via <Trans>.
        escapeValue: false,
      },
      detection: {
        // No 'navigator': first visit is deterministically English (DEFAULT_LANGUAGE),
        // regardless of browser locale. A user's saved choice (localStorage/cookie via
        // the header switcher) still wins and persists.
        order: ['localStorage', 'cookie'],
        lookupLocalStorage: LANGUAGE_STORAGE_KEY,
        lookupCookie: LANGUAGE_STORAGE_KEY,
        caches: ['localStorage', 'cookie'],
      },
      returnEmptyString: false,
      react: {
        useSuspense: false,
      },
    });
}

export default i18n;
