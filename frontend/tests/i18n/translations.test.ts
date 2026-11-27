import assert from 'node:assert/strict';
import { describe, it } from 'node:test';
import { translations } from '../../src/lib/i18n/translations.ts';

describe('translations', () => {
	it('provides the mobile menu toggle label in every locale', () => {
		for (const [locale, dict] of Object.entries(translations)) {
			const nav = dict.nav as Record<string, string>;
			assert.ok(nav.menu, `nav.menu is missing in ${locale}`);
		}
	});
});
