import { derived, writable } from 'svelte/store';
import { browser } from '$app/environment';
import { type Locale, translations } from './translations';

export type { Locale };
export const LOCALES: Locale[] = ['ja', 'en', 'zh-TW', 'zh-CN', 'ko', 'es', 'fr', 'de'];

const STORAGE_KEY = 'dora-yaki-locale';

// Get default locale from environment variable (環境変数からデフォルトロケール取得)
function getDefaultLocale(): Locale {
	const envLocale = import.meta.env.VITE_DEFAULT_LOCALE;
	if (envLocale && LOCALES.includes(envLocale as Locale)) return envLocale as Locale;
	return 'ja';
}

// Detect locale from browser language (ブラウザ言語からロケール検出)
function detectLocale(): Locale {
	const fallback = getDefaultLocale();
	if (!browser) return fallback;
	const stored = localStorage.getItem(STORAGE_KEY);
	if (stored && LOCALES.includes(stored as Locale)) return stored as Locale;

	const lang = navigator.language;
	if (lang.startsWith('ja')) return 'ja';
	if (lang === 'zh-TW' || lang === 'zh-Hant') return 'zh-TW';
	if (lang.startsWith('zh')) return 'zh-CN';
	if (lang.startsWith('ko')) return 'ko';
	if (lang.startsWith('es')) return 'es';
	if (lang.startsWith('fr')) return 'fr';
	if (lang.startsWith('de')) return 'de';
	if (lang.startsWith('en')) return 'en';
	return fallback;
}

function createLocaleStore() {
	const { subscribe, set: _set } = writable<Locale>(getDefaultLocale());

	return {
		subscribe,
		init() {
			const detected = detectLocale();
			_set(detected);
			if (browser) {
				document.documentElement.lang = detected;
			}
		},
		set(locale: Locale) {
			_set(locale);
			if (browser) {
				localStorage.setItem(STORAGE_KEY, locale);
				document.documentElement.lang = locale;
			}
		},
	};
}

export const locale = createLocaleStore();

// Retrieve value from nested keys (ネストされたキーで値取得)
function getNestedValue(obj: Record<string, unknown>, path: string): string | undefined {
	const keys = path.split('.');
	let current: unknown = obj;
	for (const key of keys) {
		if (current == null || typeof current !== 'object') return undefined;
		current = (current as Record<string, unknown>)[key];
	}
	return typeof current === 'string' ? current : undefined;
}

// Interpolate: replace {name} with params.name (補間処理)
function interpolate(text: string, params?: Record<string, string | number>): string {
	if (!params) return text;
	return text.replace(/\{(\w+)\}/g, (_, key) => {
		return params[key] != null ? String(params[key]) : `{${key}}`;
	});
}

// t derived store: returns (key, params?) => string
export const t = derived(locale, ($locale) => {
	return (key: string, params?: Record<string, string | number>): string => {
		// Fallback chain: current locale -> ja -> key name
		const value =
			getNestedValue(translations[$locale] || {}, key) ??
			getNestedValue(translations.ja || {}, key) ??
			key;
		return interpolate(value, params);
	};
});

// Locale display names
export const LOCALE_NAMES: Record<Locale, string> = {
	ja: '日本語',
	en: 'English',
	'zh-TW': '繁體中文',
	'zh-CN': '简体中文',
	ko: '한국어',
	es: 'Español',
	fr: 'Français',
	de: 'Deutsch',
};
